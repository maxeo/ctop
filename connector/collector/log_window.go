package collector

import (
	"time"

	"github.com/maxeo/ctop/models"
)

// LogCursor is a position in the log: the time of a loaded line and how many
// loaded lines have exactly that time. Lines can share a timestamp, so the
// time alone would skip or repeat some of them.
type LogCursor struct {
	Time time.Time
	Same int
}

// The collectors below read the log forward, one line at a time, and keep at
// most a page of lines: memory does not grow with the size of the log. add
// returns false once the rest of the log does not matter.

// pageBefore keeps the last n lines older than cur. The last cur.Same lines
// with the cursor's time are the loaded ones: they are left out.
type pageBefore struct {
	cur     LogCursor
	n       int
	lines   []models.Log
	dropped bool // older lines than the ones kept exist
}

func (p *pageBefore) add(e models.Log) bool {
	if e.Timestamp.After(p.cur.Time) {
		return false
	}
	p.lines = append(p.lines, e)
	if len(p.lines) > p.n+p.cur.Same {
		p.lines = p.lines[1:]
		p.dropped = true
	}
	return true
}

// result returns the page, oldest first, and whether older lines exist.
func (p *pageBefore) result() ([]models.Log, bool) {
	lines := dropLoaded(p.lines, p.cur)
	more := p.dropped
	if len(lines) > p.n {
		lines = lines[len(lines)-p.n:]
		more = true
	}
	return lines, more
}

// dropLoaded removes from the end of lines the cur.Same lines with the
// cursor's time.
func dropLoaded(lines []models.Log, cur LogCursor) []models.Log {
	for k := 0; k < cur.Same && len(lines) > 0 && lines[len(lines)-1].Timestamp.Equal(cur.Time); k++ {
		lines = lines[:len(lines)-1]
	}
	return lines
}

// afterCursor skips the lines up to cur: older ones, and the first cur.Same
// lines with the cursor's time (the loaded ones). A zero cursor skips nothing.
type afterCursor struct {
	cur  LogCursor
	seen int
}

func (a *afterCursor) skip(e models.Log) bool {
	if a.cur.Time.IsZero() {
		return false
	}
	if e.Timestamp.Before(a.cur.Time) {
		return true
	}
	if e.Timestamp.Equal(a.cur.Time) && a.seen < a.cur.Same {
		a.seen++
		return true
	}
	return false
}

// pageAfter keeps the first n lines after cur.
type pageAfter struct {
	after afterCursor
	n     int
	lines []models.Log
	more  bool // newer lines than the ones kept exist
}

func (p *pageAfter) add(e models.Log) bool {
	if p.after.skip(e) {
		return true
	}
	if len(p.lines) == p.n {
		p.more = true
		return false
	}
	p.lines = append(p.lines, e)
	return true
}

// findBefore keeps the newest line older than cur that matches, with up to
// half lines before and after it.
type findBefore struct {
	cur   LogCursor
	match func(string) bool
	half  int
	ring  []models.Log // the last half lines read
	delay []models.Log // the last cur.Same lines read, not looked at yet
	win   []models.Log
	tail  int // lines still to add after the match
	found bool
}

func newFindBefore(cur LogCursor, match func(string) bool, half int) *findBefore {
	return &findBefore{cur: cur, match: match, half: half}
}

func (f *findBefore) add(e models.Log) bool {
	if e.Timestamp.After(f.cur.Time) {
		return false
	}
	// the last cur.Same lines are the loaded ones: look at a line only once
	// cur.Same newer ones have been read
	if f.cur.Same > 0 {
		f.delay = append(f.delay, e)
		if len(f.delay) <= f.cur.Same {
			return true
		}
		e = f.delay[0]
		f.delay = f.delay[1:]
	}
	if f.match(e.Message) {
		f.win = append(append([]models.Log{}, f.ring...), e)
		f.tail = f.half
		f.found = true
	} else if f.tail > 0 {
		f.win = append(f.win, e)
		f.tail--
	}
	f.ring = append(f.ring, e)
	if len(f.ring) > f.half {
		f.ring = f.ring[1:]
	}
	return true
}

// findAfter keeps the oldest line after cur that matches, with up to half
// lines before and after it.
type findAfter struct {
	after afterCursor
	match func(string) bool
	half  int
	ring  []models.Log
	win   []models.Log
	tail  int
	found bool
}

func newFindAfter(cur LogCursor, match func(string) bool, half int) *findAfter {
	return &findAfter{after: afterCursor{cur: cur}, match: match, half: half}
}

func (f *findAfter) add(e models.Log) bool {
	if f.after.skip(e) {
		return true
	}
	if f.found {
		if f.tail == 0 {
			return false
		}
		f.win = append(f.win, e)
		f.tail--
		return true
	}
	if f.match(e.Message) {
		f.win = append(append([]models.Log{}, f.ring...), e)
		f.tail = f.half
		f.found = true
		return true
	}
	f.ring = append(f.ring, e)
	if len(f.ring) > f.half {
		f.ring = f.ring[1:]
	}
	return true
}
