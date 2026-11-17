package single

import (
	"image"

	"github.com/maxeo/ctop/models"
)

// the panels, in the order of the detail views (← → in a detail view)
const (
	panelContainer = iota
	panelCPU
	panelMem
	panelNet
	panelIO
	panelEnv
	panelCount
)

var panelNames = [panelCount]string{"Container", "CPU", "MEM", "NET", "IO", "Env"}

// boxes returns the box of each panel, by panel number.
func (e *Single) boxes() [panelCount]box {
	return [panelCount]box{e.Header.box, e.Cpu.box, e.Mem.box, e.Net.box, e.IO.box, e.Env.box}
}

// rects returns where each panel is in the main view.
func (e *Single) rects() [panelCount]image.Rectangle {
	return [panelCount]image.Rectangle{e.p.header, e.p.cpu, e.p.mem, e.p.net, e.p.io, e.p.env}
}

// Mode is what the arrows and Enter act on.
type Mode int

const (
	ModeView   Mode = iota // nothing selected
	ModeSelect             // a panel is selected
	ModeDetail             // the detail view of a panel is open
)

// Mode returns the current mode.
func (e *Single) Mode() Mode {
	e.mu.Lock()
	defer e.mu.Unlock()
	switch {
	case e.detail:
		return ModeDetail
	case e.selected >= 0:
		return ModeSelect
	}
	return ModeView
}

// Select selects the header panel (the first one).
func (e *Single) Select() {
	e.do(func() { e.selected = panelContainer })
}

// Unselect leaves the selection.
func (e *Single) Unselect() {
	e.do(func() { e.selected = -1 })
}

// MoveSelection selects the nearest panel in the direction (dx, dy), each
// -1, 0 or 1. It reports whether there was one.
func (e *Single) MoveSelection(dx, dy int) bool {
	moved := false
	e.do(func() {
		if next := nearest(e.rects(), e.selected, dx, dy); next >= 0 {
			e.selected = next
			moved = true
		}
	})
	return moved
}

// nearest returns the panel next to panel from in the direction (dx, dy):
// the one whose edge is closest, preferring those in line with from; -1 if
// none.
func nearest(rects [panelCount]image.Rectangle, from, dx, dy int) int {
	if from < 0 {
		return -1
	}
	cur := rects[from]
	best, bestScore := -1, 0
	for i, r := range rects {
		if i == from || r.Empty() {
			continue
		}
		var gap, off int
		switch {
		case dx > 0 && r.Min.X >= cur.Max.X:
			gap, off = r.Min.X-cur.Max.X, spanOffset(cur.Min.Y, cur.Max.Y, r.Min.Y, r.Max.Y)
		case dx < 0 && r.Max.X <= cur.Min.X:
			gap, off = cur.Min.X-r.Max.X, spanOffset(cur.Min.Y, cur.Max.Y, r.Min.Y, r.Max.Y)
		case dy > 0 && r.Min.Y >= cur.Max.Y:
			gap, off = r.Min.Y-cur.Max.Y, spanOffset(cur.Min.X, cur.Max.X, r.Min.X, r.Max.X)
		case dy < 0 && r.Max.Y <= cur.Min.Y:
			gap, off = cur.Min.Y-r.Max.Y, spanOffset(cur.Min.X, cur.Max.X, r.Min.X, r.Max.X)
		default:
			continue
		}
		// a panel in line counts more than a closer one aside; between equal
		// ones the first (left, top) wins
		score := gap + 4*off
		if best < 0 || score < bestScore {
			best, bestScore = i, score
		}
	}
	return best
}

// spanOffset returns how far the span [b0, b1) is from overlapping
// [a0, a1); 0 if they overlap.
func spanOffset(a0, a1, b0, b1 int) int {
	switch {
	case b1 <= a0:
		return a0 - b1 + 1
	case b0 >= a1:
		return b0 - a1 + 1
	}
	return 0
}

// OpenDetail opens the detail view of the selected panel.
func (e *Single) OpenDetail() {
	e.do(func() {
		if e.selected < 0 {
			return
		}
		e.detail = true
		e.clearFilters()
		e.list.Top()
		e.Env.Top()
		e.align()
	})
}

// CloseDetail goes back from the detail view to the panels, with the same
// panel selected.
func (e *Single) CloseDetail() {
	e.do(func() {
		e.detail = false
		e.clearFilters()
		e.Env.Top()
		e.align()
	})
}

// StepDetail shows the detail view of the previous (-1) or next (1) panel.
func (e *Single) StepDetail(dir int) {
	e.do(func() {
		next := e.selected + dir
		if !e.detail || next < 0 || next >= panelCount {
			return
		}
		e.selected = next
		e.clearFilters()
		e.list.Top()
		e.Env.Top()
		e.align()
	})
}

func (e *Single) clearFilters() {
	e.list.SetFilter("", false)
	e.Env.SetFilter("", false)
}

// current returns the list the scroll and filter keys act on: the detail
// list, or the environment.
func (e *Single) current() *KVList {
	if e.detail && e.selected != panelEnv {
		return e.list
	}
	return e.Env.KVList
}

// Scroll moves the current list by n lines.
func (e *Single) Scroll(n int) { e.do(func() { e.current().Scroll(n) }) }

// Page moves the current list by a page (dir -1: up, 1: down).
func (e *Single) Page(dir int) { e.do(func() { e.current().Page(dir) }) }

// Top and Bottom go to the start and the end of the current list.
func (e *Single) Top()    { e.do(func() { e.current().Top() }) }
func (e *Single) Bottom() { e.do(func() { e.current().Bottom() }) }

// Filter shows the rows of the current list matching f (typing: still
// being typed).
func (e *Single) Filter(f string, typing bool) {
	e.do(func() { e.current().SetFilter(f, typing) })
}

// Details returns the header fields as a detail section, for connectors
// without details.
func (h *Header) Details() []models.DetailSection {
	var r []models.DetailRow
	for _, k := range append([]string{"name", "image", "state"}, headerFields...) {
		r = append(r, models.DetailRow{Key: k, Value: h.value(k)})
	}
	return []models.DetailSection{{Title: "Container", Rows: r}}
}
