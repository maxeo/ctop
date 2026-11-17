package main

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/maxeo/ctop/connector/collector"
	"github.com/maxeo/ctop/models"
	"github.com/maxeo/ctop/widgets"
)

// logHistory loads log lines that are not in the log view yet.
//
// The view normally holds the end of the log (following new lines) and
// LoadPage adds older pages when the user scrolls past the oldest line.
// Head replaces it with the start of the log, detached from the tail;
// LoadNext then adds newer pages when the user scrolls down, until the view
// reaches the tail and follows new lines again. Tail goes back to the end.
// FindOlder and FindNewer replace it with the lines around a match.
// Only one load runs at a time.
//
// The view keeps a limited number of lines: the ones furthest from the screen
// are dropped and loaded again from the log when the user goes back to them.
type logHistory struct {
	view   *widgets.TextView
	source collector.LogHistory
	label  string
	loadMu sync.Mutex // held while a load runs
	// the first line of the log, while it is the oldest one in the view
	// (nil if the start of the log is not loaded, or was dropped since)
	start atomic.Pointer[toggleLog]
}

func newLogHistory(view *widgets.TextView, source collector.LogHistory, label string) *logHistory {
	return &logHistory{view: view, source: source, label: label}
}

// LoadPage prepends the page of lines before the oldest one shown.
func (h *logHistory) LoadPage() {
	// skip if another load is running: it will move the view anyway
	if h.atStart() || !h.loadMu.TryLock() {
		return
	}
	defer h.loadMu.Unlock()
	h.loadPage()
}

func (h *logHistory) loadPage() bool {
	cur, loaded, ok := h.cursor(false)
	if !ok {
		return false
	}
	skip := loaded
	if h.view.Detached() {
		skip = -1 // the loaded lines don't end at the tail
	}
	older, more, err := h.source.Older(cur, skip)
	if err != nil {
		log.Errorf("loading older logs: %s", err)
		return false
	}
	h.view.Prepend(toToggleText(older))
	if !more {
		h.markStart()
	}
	return len(older) > 0
}

// FindOlder shows the lines around the newest line older than the loaded
// ones that matches. It reports whether one was found.
func (h *logHistory) FindOlder(match func(string) bool) bool {
	h.loadMu.Lock()
	defer h.loadMu.Unlock()

	cur, _, ok := h.cursor(false)
	if !ok || h.atStart() {
		return false
	}
	h.busy(true)
	lines, found, err := h.source.FindOlder(cur, match)
	h.busy(false)
	if err != nil {
		log.Errorf("searching older logs: %s", err)
		return false
	}
	if !found {
		return false
	}
	h.start.Store(nil)
	h.view.Replace(toToggleText(lines))
	return true
}

// FindNewer shows the lines around the oldest line newer than the loaded
// ones that contains query, while the view is detached from the tail. It
// reports whether one was found.
func (h *logHistory) FindNewer(match func(string) bool) bool {
	h.loadMu.Lock()
	defer h.loadMu.Unlock()

	cur, _, ok := h.cursor(true)
	if !ok || !h.view.Detached() {
		return false
	}
	h.busy(true)
	lines, found, err := h.source.FindNewer(cur, match)
	h.busy(false)
	if err != nil {
		log.Errorf("searching newer logs: %s", err)
		return false
	}
	if !found {
		return false
	}
	h.start.Store(nil)
	h.view.Replace(toToggleText(lines))
	h.view.ScrollToTop()
	return true
}

// Head shows the first page of the log.
func (h *logHistory) Head() {
	h.loadMu.Lock()
	defer h.loadMu.Unlock()

	if h.atStart() && !h.view.Detached() {
		// the whole log is loaded already
		h.view.ScrollToTop()
		return
	}
	// keep new lines aside from now on, so none is lost while loading
	wasDetached := h.view.Detached()
	h.view.Detach()
	h.busy(true)
	page, more, err := h.source.Newer(collector.LogCursor{})
	h.busy(false)
	if err != nil {
		log.Errorf("loading the start of the log: %s", err)
		if !wasDetached {
			h.view.Attach(nil, func(widgets.ToggleText) bool { return true })
		}
		return
	}
	if more {
		h.view.Replace(toToggleText(page))
	} else {
		// the log fits in one page: show it all and follow the tail
		h.view.Replace(nil)
		h.attach(page)
	}
	h.markStart()
	h.view.ScrollToTop()
}

// LoadNext appends the page after the newest line shown, while the view is
// detached. On the last page the view joins the tail again.
func (h *logHistory) LoadNext() {
	if !h.loadMu.TryLock() {
		return
	}
	defer h.loadMu.Unlock()
	h.loadNext()
}

func (h *logHistory) loadNext() bool {
	cur, _, ok := h.cursor(true)
	if !ok || !h.view.Detached() {
		return false
	}
	page, more, err := h.source.Newer(cur)
	if err != nil {
		log.Errorf("loading newer logs: %s", err)
		return false
	}
	if more {
		h.view.Append(toToggleText(page))
	} else {
		h.attach(page)
	}
	return true
}

// Tail shows the end of the log again, following new lines.
func (h *logHistory) Tail() {
	if !h.view.Detached() {
		h.view.ScrollToBottom()
		return
	}
	h.loadMu.Lock()
	defer h.loadMu.Unlock()

	cutoff := time.Now()
	h.busy(true)
	page, more, err := h.source.Older(collector.LogCursor{Time: cutoff}, 0)
	h.busy(false)
	if err != nil {
		log.Errorf("loading the end of the log: %s", err)
		return
	}
	h.start.Store(nil)
	h.view.Replace(nil)
	h.view.Attach(toToggleText(page), func(l widgets.ToggleText) bool {
		tl, ok := l.(*toggleLog)
		return ok && tl.timestamp.After(cutoff)
	})
	if !more {
		h.markStart()
	}
	h.view.ScrollToBottom()
}

// attach appends the last page of a detached view, then the live lines
// received meanwhile that are newer than it.
func (h *logHistory) attach(page []models.Log) {
	var after time.Time
	if len(page) > 0 {
		after = page[len(page)-1].Timestamp
	} else if cur, _, ok := h.cursor(true); ok {
		after = cur.Time
	}
	h.view.Attach(toToggleText(page), func(l widgets.ToggleText) bool {
		tl, ok := l.(*toggleLog)
		return ok && tl.timestamp.After(after)
	})
}

func (h *logHistory) busy(on bool) {
	if on {
		h.view.SetLabel(h.label + " (loading...)")
	} else {
		h.view.SetLabel(h.label)
	}
}

// cursor returns the position of the oldest loaded line (the newest one if
// newest) and how many lines are loaded. ok is false if none is.
func (h *logHistory) cursor(newest bool) (cur collector.LogCursor, loaded int, ok bool) {
	line, run, loaded := h.view.Edge(newest, sameTime)
	tl, ok := line.(*toggleLog)
	if !ok {
		return cur, loaded, false
	}
	return collector.LogCursor{Time: tl.timestamp, Same: run}, loaded, true
}

func sameTime(a, b widgets.ToggleText) bool {
	ta, okA := a.(*toggleLog)
	tb, okB := b.(*toggleLog)
	return okA && okB && ta.timestamp.Equal(tb.timestamp)
}

// markStart records that the oldest line in the view is the first one of
// the log.
func (h *logHistory) markStart() {
	line, _, _ := h.view.Edge(false, sameTime)
	tl, _ := line.(*toggleLog)
	h.start.Store(tl)
}

// atStart reports whether the first line of the log is loaded: it is still
// the oldest line in the view (not dropped to save memory).
func (h *logHistory) atStart() bool {
	start := h.start.Load()
	if start == nil {
		return false
	}
	line, _, _ := h.view.Edge(false, sameTime)
	tl, _ := line.(*toggleLog)
	return tl == start
}

func toToggleText(logs []models.Log) []widgets.ToggleText {
	lines := make([]widgets.ToggleText, 0, len(logs))
	for _, l := range logs {
		lines = append(lines, &toggleLog{timestamp: l.Timestamp, message: l.Message})
	}
	return lines
}
