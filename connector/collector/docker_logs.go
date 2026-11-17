package collector

import (
	"bufio"
	"bytes"
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

// Older implements LogHistory. Docker cannot combine "tail" with "until"
// (tail is applied first), so it asks for the last skip+page lines and keeps
// the ones older than before.
func (l *DockerLogs) Older(before time.Time, skip int) ([]models.Log, bool, error) {
	tail := skip + logPageSize
	older, fetched, err := l.readBefore(before, strconv.Itoa(tail))
	if err != nil {
		return nil, false, err
	}
	if len(older) > logPageSize {
		older = older[len(older)-logPageSize:]
	}
	// fewer lines than requested: the whole log has been read
	return older, fetched >= tail, nil
}

// AllBefore implements LogHistory.
func (l *DockerLogs) AllBefore(before time.Time) ([]models.Log, error) {
	older, _, err := l.readBefore(before, "all")
	return older, err
}

// Newer implements LogHistory. Docker has no "head" option: it reads the log
// forward (from the second of after, via "since") and stops the request once
// a page plus one line is read.
func (l *DockerLogs) Newer(after time.Time) ([]models.Log, bool, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	r, w := io.Pipe()
	opts := api.LogsOptions{
		Context:      ctx,
		Container:    l.id,
		OutputStream: newDemuxWriter(w),
		Stdout:       true,
		Stderr:       true,
		Tail:         "all",
		Timestamps:   true,
		RawTerminal:  true,
	}
	if !after.IsZero() {
		opts.Since = after.Unix()
	}
	errCh := make(chan error, 1)
	go func() {
		err := l.client.Logs(opts)
		w.Close()
		errCh <- err
	}()

	var page []models.Log
	more := false
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		entry := l.parseLine(scanner.Text())
		if !after.IsZero() && !entry.Timestamp.After(after) {
			continue
		}
		if len(page) == logPageSize {
			more = true
			break
		}
		page = append(page, entry)
	}
	if more {
		cancel()
		r.CloseWithError(context.Canceled) // unblock the writer
		<-errCh
		return page, true, nil
	}
	if err := <-errCh; err != nil {
		return nil, false, err
	}
	return page, false, nil
}

// readBefore reads the last tail lines (non-following) and returns those
// logged before the given time, plus how many lines were read in total.
func (l *DockerLogs) readBefore(before time.Time, tail string) ([]models.Log, int, error) {
	var buf bytes.Buffer
	opts := api.LogsOptions{
		Container:    l.id,
		OutputStream: newDemuxWriter(&buf),
		Stdout:       true,
		Stderr:       true,
		Tail:         tail,
		Timestamps:   true,
		RawTerminal:  true,
	}
	if err := l.client.Logs(opts); err != nil {
		return nil, 0, err
	}

	var older []models.Log
	fetched := 0
	scanner := bufio.NewScanner(&buf)
	for scanner.Scan() {
		fetched++
		entry := l.parseLine(scanner.Text())
		if entry.Timestamp.Before(before) {
			older = append(older, entry)
		}
	}
	return older, fetched, nil
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
