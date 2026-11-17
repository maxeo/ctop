package collector

import (
	"bufio"
	"context"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/eqms/ctop/models"
	api "github.com/fsouza/go-dockerclient"
)

// ANSI escape sequences (colors, cursor moves) written by many containers,
// e.g. Home Assistant: the log view shows plain text, so they are removed
var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;?]*[ -/]*[@-~]|\x1b[@-Z\-_]`)

const (
	// lines loaded when the log view opens
	logInitialTail = 20
	// lines loaded per page when scrolling back
	logPageSize = 200
)

type DockerLogs struct {
	id     string
	client *api.Client
	done   chan bool
}

func NewDockerLogs(id string, client *api.Client) *DockerLogs {
	return &DockerLogs{
		id:     id,
		client: client,
		done:   make(chan bool),
	}
}

func (l *DockerLogs) Stream() chan models.Log {
	r, w := io.Pipe()
	logCh := make(chan models.Log)
	ctx, cancel := context.WithCancel(context.Background())

	opts := api.LogsOptions{
		Context:      ctx,
		Container:    l.id,
		OutputStream: newDemuxWriter(w),
		//ErrorStream:  w,
		Stdout:      true,
		Stderr:      true,
		Tail:        strconv.Itoa(logInitialTail),
		Follow:      true,
		Timestamps:  true,
		RawTerminal: true,
	}

	// read io pipe into channel
	go func() {
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			// don't block forever once the reader is gone
			select {
			case logCh <- l.parseLine(scanner.Text()):
			case <-ctx.Done():
				return
			}
		}
	}()

	// connect to container log stream
	go func() {
		err := l.client.Logs(opts)
		w.Close() // unblock the scanner goroutine
		if err != nil {
			log.Errorf("error reading container logs: %s", err)
		}
		log.Infof("log reader stopped for container: %s", l.id)
	}()

	go func() {
		<-l.done
		cancel()
	}()

	log.Infof("log reader started for container: %s", l.id)
	return logCh
}

func (l *DockerLogs) Stop() { l.done <- true }

// Older implements LogHistory. Docker applies "tail" before any time filter
// and has no "until" in this client, so when the loaded lines end at the
// tail (skip >= 0) it reads the last skip+page lines; otherwise, or if those
// are all newer than before, it reads the log from the start up to before.
// Both keep at most a page in memory.
func (l *DockerLogs) Older(before LogCursor, skip int) ([]models.Log, bool, error) {
	if skip >= 0 {
		tail := skip + before.Same + logPageSize
		p := pageBefore{cur: before, n: logPageSize}
		read := 0
		err := l.stream(time.Time{}, strconv.Itoa(tail), func(e models.Log) bool {
			read++
			p.add(e) // a line out of time order does not end the read
			return true
		})
		if err != nil {
			return nil, false, err
		}
		lines, more := p.result()
		if read < tail {
			// the whole log has been read
			return lines, more, nil
		}
		if len(lines) > 0 {
			return lines, true, nil
		}
	}

	p := pageBefore{cur: before, n: logPageSize}
	if err := l.stream(time.Time{}, "all", p.add); err != nil {
		return nil, false, err
	}
	lines, more := p.result()
	return lines, more, nil
}

// Newer implements LogHistory. Docker has no "head" option: it reads the log
// forward (from the second of after, via "since") and stops the request once
// a page plus one line is read.
func (l *DockerLogs) Newer(after LogCursor) ([]models.Log, bool, error) {
	p := pageAfter{after: afterCursor{cur: after}, n: logPageSize}
	if err := l.stream(after.Time, "all", p.add); err != nil {
		return nil, false, err
	}
	return p.lines, p.more, nil
}

// FindOlder implements LogHistory. It reads the log from the start up to
// before, keeping only the lines around the last match.
func (l *DockerLogs) FindOlder(before LogCursor, match func(string) bool) ([]models.Log, bool, error) {
	f := newFindBefore(before, match, logPageSize/2)
	if err := l.stream(time.Time{}, "all", f.add); err != nil {
		return nil, false, err
	}
	return f.win, f.found, nil
}

// FindNewer implements LogHistory. It reads the log from after and stops
// shortly after the first match.
func (l *DockerLogs) FindNewer(after LogCursor, match func(string) bool) ([]models.Log, bool, error) {
	f := newFindAfter(after, match, logPageSize/2)
	if err := l.stream(after.Time, "all", f.add); err != nil {
		return nil, false, err
	}
	return f.win, f.found, nil
}

// most bytes in a log line; longer lines end the read with an error
const maxLogLine = 1 << 20

// stream reads the log without following it: the last tail lines ("all" for
// every line), from the second of since if not zero. It calls fn for each
// line, oldest first, and stops the request as soon as fn returns false.
func (l *DockerLogs) stream(since time.Time, tail string, fn func(models.Log) bool) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	r, w := io.Pipe()
	opts := api.LogsOptions{
		Context:      ctx,
		Container:    l.id,
		OutputStream: newDemuxWriter(w),
		Stdout:       true,
		Stderr:       true,
		Tail:         tail,
		Timestamps:   true,
		RawTerminal:  true,
	}
	if !since.IsZero() {
		opts.Since = since.Unix()
	}
	errCh := make(chan error, 1)
	go func() {
		err := l.client.Logs(opts)
		w.Close()
		errCh <- err
	}()

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), maxLogLine)
	stopped := false
	for scanner.Scan() {
		if !fn(l.parseLine(scanner.Text())) {
			stopped = true
			break
		}
	}
	if scanErr := scanner.Err(); scanErr != nil || stopped {
		// end the request early and unblock its writer
		cancel()
		r.CloseWithError(context.Canceled)
		<-errCh
		return scanErr
	}
	return <-errCh
}

func (l *DockerLogs) parseLine(line string) models.Log {
	line = ansiEscape.ReplaceAllString(line, "")
	parts := strings.SplitN(line, " ", 2)
	if len(parts) < 2 {
		return models.Log{Timestamp: l.parseTime(""), Message: parts[0]}
	}
	return models.Log{Timestamp: l.parseTime(parts[0]), Message: parts[1]}
}

func (l *DockerLogs) parseTime(s string) time.Time {
	ts, err := time.Parse(time.RFC3339Nano, s)
	if err == nil {
		return ts
	}

	ts, err2 := time.Parse(time.RFC3339Nano, l.stripPfx(s))
	if err2 == nil {
		return ts
	}

	log.Errorf("failed to parse container log: %s", err)
	log.Errorf("failed to parse container log2: %s", err2)
	return time.Now()
}

// attempt to strip message header prefix from a given raw docker log string
func (l *DockerLogs) stripPfx(s string) string {
	b := []byte(s)
	if len(b) > 8 {
		return string(b[8:])
	}
	return s
}
