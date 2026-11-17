package main

import (
	"sync"
	"sync/atomic"

	"github.com/eqms/ctop/connector/collector"
	"github.com/eqms/ctop/models"
	"github.com/eqms/ctop/widgets"
)

// logHistory loads older log lines into the log view: one page when the user
// scrolls past the oldest line, or everything when jumping to the start.
// Only one load runs at a time; once the start of the log is loaded, further
// requests do nothing.
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

	first, loaded := h.oldest()
	if first == nil {
		return
	}
	older, more, err := h.source.Older(first.timestamp, loaded)
	if err != nil {
		log.Errorf("loading older logs: %s", err)
		return
	}
	if !more {
		h.markExhausted()
	}
	h.view.Prepend(toToggleText(older))
}

// LoadAll prepends every line before the oldest one shown, then shows the
// start of the log.
func (h *logHistory) LoadAll() {
	// wait for a running page load, then load the rest
	h.loadMu.Lock()
	defer h.loadMu.Unlock()

	if first, _ := h.oldest(); first != nil && !h.exhausted.Load() {
		h.view.SetLabel(h.label + " (loading...)")
		older, err := h.source.AllBefore(first.timestamp)
		h.view.SetLabel(h.label)
		if err != nil {
			log.Errorf("loading all logs: %s", err)
			return
		}
		h.markExhausted()
		h.view.Prepend(toToggleText(older))
	}
	h.view.ScrollToTop()
}

func (h *logHistory) markExhausted() {
	h.exhausted.Store(true)
}

func (h *logHistory) oldest() (*toggleLog, int) {
	oldest, loaded := h.view.Oldest()
	first, _ := oldest.(*toggleLog)
	return first, loaded
}

func toToggleText(logs []models.Log) []widgets.ToggleText {
	lines := make([]widgets.ToggleText, 0, len(logs))
	for _, l := range logs {
		lines = append(lines, &toggleLog{timestamp: l.Timestamp, message: l.Message})
	}
	return lines
}
