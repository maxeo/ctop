package main

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/eqms/ctop/connector/collector"
	"github.com/eqms/ctop/models"
	"github.com/eqms/ctop/widgets"
)

// logHistory loads log lines that are not in the log view yet.
//
// The view normally holds the end of the log (following new lines) and
// LoadPage adds older pages when the user scrolls past the oldest line.
// Head replaces it with the start of the log, detached from the tail;
// LoadNext then adds newer pages when the user scrolls down, until the view
// reaches the tail and follows new lines again. Tail goes back to the end.
// Only one load runs at a time.
type logHistory struct {
	view      *widgets.TextView
	source    collector.LogHistory
	label     string
	loadMu    sync.Mutex  // held while a load runs
	exhausted atomic.Bool // the start of the log is loaded
}

func newLogHistory(view *widgets.TextView, source collector.LogHistory, label string) *logHistory {
	return &logHistory{view: view, source: source, label: label}
}

// LoadPage prepends the page of lines before the oldest one shown.
func (h *logHistory) LoadPage() {
	// skip if another load is running: it will move the view anyway
	if h.exhausted.Load() || !h.loadMu.TryLock() {
		return
	}
	defer h.loadMu.Unlock()
	h.loadPage()
}

func (h *logHistory) loadPage() bool {
	first, loaded := h.oldest()
	if first == nil {
		return false
	}
	older, more, err := h.source.Older(first.timestamp, loaded)
	if err != nil {
		log.Errorf("loading older logs: %s", err)
		return false
	}
	if !more {
		h.exhausted.Store(true)
	}
	h.view.Prepend(toToggleText(older))
	return len(older) > 0
}

// LoadAll prepends every line before the oldest one shown. It reports
// whether lines were added.
func (h *logHistory) LoadAll() bool {
	h.loadMu.Lock()
	defer h.loadMu.Unlock()

	first, _ := h.oldest()
	if first == nil || h.exhausted.Load() {
		return false
	}
	h.busy(true)
	older, err := h.source.AllBefore(first.timestamp)
	h.busy(false)
	if err != nil {
		log.Errorf("loading all logs: %s", err)
		return false
	}
	h.exhausted.Store(true)
	h.view.Prepend(toToggleText(older))
	return len(older) > 0
}

// Head shows the first page of the log.
func (h *logHistory) Head() {
	h.loadMu.Lock()
	defer h.loadMu.Unlock()

	if h.exhausted.Load() && !h.view.Detached() {
		// the whole log is loaded already
		h.view.ScrollToTop()
		return
	}
	// keep new lines aside from now on, so none is lost while loading
	wasDetached := h.view.Detached()
	h.view.Detach()
	h.busy(true)
	page, more, err := h.source.Newer(time.Time{})
	h.busy(false)
	if err != nil {
		log.Errorf("loading the start of the log: %s", err)
		if !wasDetached {
			h.view.Attach(nil, func(widgets.ToggleText) bool { return true })
		}
		return
	}
	h.exhausted.Store(true)
	if more {
		h.view.Replace(toToggleText(page))
	} else {
		// the log fits in one page: show it all and follow the tail
		h.view.Replace(nil)
		h.attach(page)
	}
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
	last := h.newest()
	if last == nil || !h.view.Detached() {
		return false
	}
	page, more, err := h.source.Newer(last.timestamp)
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

// LoadRest appends pages until the view joins the tail again. It reports
// whether lines were added.
func (h *logHistory) LoadRest() bool {
	h.loadMu.Lock()
	defer h.loadMu.Unlock()
	if !h.view.Detached() {
		return false
	}
	h.busy(true)
	defer h.busy(false)
	for h.view.Detached() {
		if !h.loadNext() {
			return false
		}
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
	page, _, err := h.source.Older(cutoff, 0)
	h.busy(false)
	if err != nil {
		log.Errorf("loading the end of the log: %s", err)
		return
	}
	h.exhausted.Store(false)
	h.view.Replace(nil)
	h.view.Attach(toToggleText(page), func(l widgets.ToggleText) bool {
		tl, ok := l.(*toggleLog)
		return ok && !tl.timestamp.Before(cutoff)
	})
	h.view.ScrollToBottom()
}

// attach appends the last page of a detached view, then the live lines
// received meanwhile that are newer than it.
func (h *logHistory) attach(page []models.Log) {
	var after time.Time
	if len(page) > 0 {
		after = page[len(page)-1].Timestamp
	} else if last := h.newest(); last != nil {
		after = last.timestamp
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

func (h *logHistory) oldest() (*toggleLog, int) {
	oldest, loaded := h.view.Oldest()
	first, _ := oldest.(*toggleLog)
	return first, loaded
}

func (h *logHistory) newest() *toggleLog {
	last, _ := h.view.Newest().(*toggleLog)
	return last
}

func toToggleText(logs []models.Log) []widgets.ToggleText {
	lines := make([]widgets.ToggleText, 0, len(logs))
	for _, l := range logs {
		lines = append(lines, &toggleLog{timestamp: l.Timestamp, message: l.Message})
	}
	return lines
}
