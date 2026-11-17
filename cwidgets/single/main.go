package single

import (
	"image"
	"strconv"
	"sync"

	ui "github.com/gizak/termui"
	"github.com/maxeo/ctop/cwidgets"
	"github.com/maxeo/ctop/logging"
	"github.com/maxeo/ctop/models"
	"github.com/maxeo/ctop/widgets"
)

var log = logging.Init()

// the smallest terminal the view is drawn in
const (
	minWidth  = 40
	minHeight = 16
)

// from this width CPU and MEM, then NET and IO, are side by side
const wideWidth = 110

// Single is the view of one container: a header with its details, the CPU,
// MEM, NET and IO panels and the environment, laid out for the terminal
// size, with a key help at the bottom.
//
// Enter selects a panel (the arrows move the selection), Enter again opens
// its detail view: the panel in large and everything known about it, in a
// list that scrolls and can be filtered.
//
// Metrics and meta come from the collector goroutine, drawing and keys from
// the UI loop: every method takes the lock.
type Single struct {
	mu     sync.Mutex
	Header *Header
	Cpu    *Cpu
	Mem    *Mem
	Net    *Rates
	IO     *Rates
	Env    *Env
	keys   *widgets.KeyBar

	details  models.Details
	selected int     // panel selected (see panelNames), -1 if none
	detail   bool    // the detail view of the selected panel is open
	list     *KVList // the list of the detail view
	p        panels  // where the panels are

	width, height int
	tooSmall      bool
}

func NewSingle() *Single {
	list := NewKVList("")
	list.Wrap = true
	return &Single{
		Header:   NewHeader(),
		Cpu:      NewCpu(),
		Mem:      NewMem(),
		Net:      NewNet(),
		IO:       NewIO(),
		Env:      NewEnv(),
		keys:     widgets.NewKeyBar(nil, nil),
		selected: -1,
		list:     list,
	}
}

// SetKeys sets the key help shown at the bottom.
func (e *Single) SetKeys(keys []widgets.KeyHelp) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.keys.Keys = keys
	e.align() // the key help may take another number of rows
}

// SetWidth is part of cwidgets.WidgetUpdater; the view always uses the
// terminal size (see Align).
func (e *Single) SetWidth(int) {}

func (e *Single) SetMeta(m models.Meta) {
	e.mu.Lock()
	defer e.mu.Unlock()
	for k, v := range m {
		if k == "[ENV-VAR]" {
			e.Env.Set(v)
		} else {
			e.Header.Set(k, v)
		}
	}
}

// SetDetails takes what the connector knows about the container beyond its
// metrics (see container.DetailsUpdater).
func (e *Single) SetDetails(d models.Details) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.details = d
	e.Header.startedAt, e.Header.running = d.StartedAt, d.Running
	e.Net.note = ""
	if d.NetworkMode == "host" {
		e.Net.note = "host network: the traffic is not counted per container"
	}
}

func (e *Single) SetMetrics(m models.Metrics) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.Cpu.Update(m.CPUUtil, int(m.NCpus))
	e.Mem.Update(int(m.MemUsage), int(m.MemLimit))
	e.Net.Update(m.NetRx, m.NetTx)
	e.IO.Update(m.IOBytesRead, m.IOBytesWrite)
	if m.Pids >= 0 {
		e.Header.data["pids"] = strconv.Itoa(m.Pids)
	}
}

// panels are the areas of the view.
type panels struct {
	header, cpu, mem, net, io, env image.Rectangle
}

// layout places the panels in a w x h terminal, above keyRows rows of key
// help. The charts get their preferred height when there is room; the
// environment takes the rest, at least envMin rows when it can.
func layout(w, h, headerRows, keyRows, envVars int) panels {
	var p panels
	p.header = image.Rect(0, 0, w, headerRows)
	top := headerRows
	body := h - keyRows - top

	// the environment wants a row per variable (at least one, for a message)
	envWant := envVars
	if envWant < 1 {
		envWant = 1
	}
	envWant += 2
	envMin := 5
	if envWant < envMin {
		envMin = envWant
	}

	if w >= wideWidth {
		// two columns: CPU | MEM, then NET | IO
		const r1Min, r1Pref, r1Max, r2Min, r2Pref, r2Max = 5, 10, 14, 6, 6, 10
		// the preferred height first, leaving at least envMin rows to the
		// environment; more if the environment does not need the rows
		charts := clamp(body-envMin, r1Min+r2Min, r1Pref+r2Pref)
		charts = clamp(body-envWant, charts, r1Max+r2Max)
		r1 := clamp(charts*r1Pref/(r1Pref+r2Pref), r1Min, r1Max)
		r2 := clamp(charts-r1, r2Min, r2Max)
		r1 = clamp(charts-r2, r1Min, r1Max)
		left := w / 2
		p.cpu = image.Rect(0, top, left, top+r1)
		p.mem = image.Rect(left, top, w, top+r1)
		p.net = image.Rect(0, top+r1, left, top+r1+r2)
		p.io = image.Rect(left, top+r1, w, top+r1+r2)
		top += r1 + r2
	} else {
		// one column: CPU, MEM, NET, IO
		mins := [4]int{5, 4, 6, 6}
		prefs := [4]int{8, 5, 6, 6}
		maxs := [4]int{12, 9, 10, 10}
		sizes := mins
		total := mins[0] + mins[1] + mins[2] + mins[3]
		// the preferred heights first, leaving at least envMin rows to the
		// environment; then up to the maximum if the environment does not
		// need the rows
		grow(sizes[:], prefs[:], body-envMin-total)
		total = sizes[0] + sizes[1] + sizes[2] + sizes[3]
		grow(sizes[:], maxs[:], body-envWant-total)
		rects := []*image.Rectangle{&p.cpu, &p.mem, &p.net, &p.io}
		for i, r := range rects {
			*r = image.Rect(0, top, w, top+sizes[i])
			top += sizes[i]
		}
	}
	if envH := h - keyRows - top; envH >= 3 {
		p.env = image.Rect(0, top, w, top+envH)
	}
	return p
}

// grow adds up to room rows to sizes, one at a time to each in turn, up to
// the limits.
func grow(sizes, limits []int, room int) {
	for room > 0 {
		grown := false
		for i := range sizes {
			if room > 0 && sizes[i] < limits[i] {
				sizes[i]++
				room--
				grown = true
			}
		}
		if !grown {
			return
		}
	}
}

func clamp(v, lo, hi int) int {
	if v > hi {
		v = hi
	}
	if v < lo {
		v = lo
	}
	return v
}

// Align lays the panels out for the terminal size.
func (e *Single) Align() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.align()
}

func (e *Single) align() {
	e.width, e.height = cwidgets.TermWidth(), cwidgets.TermHeight()
	e.tooSmall = e.width < minWidth || e.height < minHeight
	if e.tooSmall {
		return
	}
	e.Header.setWidth(e.width)
	keyRows := e.keys.Height()
	e.p = layout(e.width, e.height, e.Header.rows(), keyRows, e.Env.total)
	e.Header.place(e.p.header)
	if e.detail {
		e.alignDetail(keyRows)
		return
	}
	e.Env.Wrap = false
	e.Cpu.place(e.p.cpu)
	e.Mem.place(e.p.mem)
	e.Net.place(e.p.net)
	e.IO.place(e.p.io)
	e.Env.place(e.p.env)
	e.Env.width = -1
	log.Debugf("single view: %dx%d", e.width, e.height)
}

// alignDetail lays out the detail view: the header, the panel in large
// (for the charts) and the list below.
func (e *Single) alignDetail(keyRows int) {
	top := e.p.header.Max.Y
	bottom := e.height - keyRows
	body := image.Rect(0, top, e.width, bottom)
	if e.selected == panelEnv {
		// the variables in the whole body, long values wrapped
		e.Env.Wrap = true
		e.Env.place(body)
		e.Env.width = -1
		return
	}
	if w := e.chart(); w != nil {
		h := clamp((bottom-top)*2/5, 6, 18)
		w.place(image.Rect(0, top, e.width, top+h))
		top += h
	}
	e.list.place(image.Rect(0, top, e.width, bottom))
	e.list.width = -1
}

// chart returns the box of the selected panel when it is a chart.
func (e *Single) chart() *box {
	switch e.selected {
	case panelCPU:
		return &e.Cpu.box
	case panelMem:
		return &e.Mem.box
	case panelNet:
		return &e.Net.box
	case panelIO:
		return &e.IO.box
	}
	return nil
}

func (e *Single) Buffer() ui.Buffer {
	e.mu.Lock()
	defer e.mu.Unlock()
	if w, h := cwidgets.TermWidth(), cwidgets.TermHeight(); w != e.width || h != e.height {
		e.align()
	}
	buf := ui.NewBuffer()
	if e.tooSmall {
		buf.SetArea(image.Rect(0, 0, e.width, e.height))
		clear(buf, buf.Area)
		drawText(buf, 0, 0, e.width, "screen too small!", textFg(), ui.ColorDefault)
		return buf
	}
	for i, b := range e.boxes() {
		b.highlight(i == e.selected && !e.detail)
	}
	buf.Merge(e.Header.Buffer())
	if e.detail {
		buf.Merge(e.detailBuffer())
	} else {
		buf.Merge(e.Cpu.Buffer(), e.Mem.Buffer(), e.Net.Buffer(), e.IO.Buffer())
		if e.Env.Height > 0 {
			buf.Merge(e.Env.Buffer())
		}
	}
	buf.Merge(e.keys.Buffer())
	return buf
}

func (e *Single) detailBuffer() ui.Buffer {
	if e.selected == panelEnv {
		return e.Env.Buffer()
	}
	buf := ui.NewBuffer()
	var sections []models.DetailSection
	switch e.selected {
	case panelContainer:
		sections = e.details.Container
		if len(sections) == 0 {
			sections = e.Header.Details()
		}
	case panelCPU:
		buf.Merge(e.Cpu.Buffer())
		sections = append(e.Cpu.Details(), e.details.CPU...)
	case panelMem:
		buf.Merge(e.Mem.Buffer())
		sections = append(e.Mem.Details(), e.details.Mem...)
	case panelNet:
		buf.Merge(e.Net.Buffer())
		sections = append(e.Net.Details(), e.details.Net...)
	case panelIO:
		buf.Merge(e.IO.Buffer())
		sections = append(e.IO.Details(), e.details.IO...)
	}
	e.list.Title = panelNames[e.selected] + " details"
	e.list.SetSections(sections)
	buf.Merge(e.list.Buffer())
	return buf
}

func (e *Single) do(f func()) {
	e.mu.Lock()
	f()
	e.mu.Unlock()
	ui.Render(e)
}
