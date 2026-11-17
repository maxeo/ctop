package widgets

import (
	"math"
	"sync"

	"github.com/eqms/ctop/cwidgets"
	ui "github.com/gizak/termui"
	"github.com/mattn/go-runewidth"
)

type ToggleText interface {
	// returns text for toggle on/off
	Toggle(on bool) string
}

// most live lines kept aside while the view is detached from the tail
const maxPending = 10000

// TextView shows a scrollable window of text lines.
//
// Normally the loaded lines end at the live tail and new lines from the input
// stream are appended. When detached (e.g. showing the start of a log), the
// loaded lines end somewhere in the middle: new input lines are kept aside in
// pending until Attach joins the view to the tail again.
type TextView struct {
	ui.Block
	inputStream  <-chan ToggleText
	render       chan struct{} // buffered(1): pending render requests are coalesced
	done         chan struct{} // closed when the input stream ends
	stopped      chan struct{} // closed when the render goroutine returns
	mu           sync.Mutex    // guards the fields below, shared by input, render and event goroutines
	toggleState  bool
	scroll       int          // lines scrolled back from the bottom (0 = follow tail, < 0 = blank rows below the newest line)
	onTop        func()       // called when the user scrolls past the oldest loaded line
	onBottom     func()       // called when the user scrolls past the newest line while detached
	detached     bool         // the loaded lines don't reach the live tail
	pending      []ToggleText // input lines received while detached
	status       string       // status row above the key help, like nano's
	keys         []KeyHelp    // key help in the two bottom rows
	beforeRender func()       // runs once on the render goroutine, see BeforeRender
	search       viewSearch
	Text         []ToggleText // all the text
	TextOut      []string     // text to be displayed
	outSrc       []int        // index in Text of each TextOut line
	outFirst     []bool       // whether each TextOut line starts its Text line (not a wrapped part)
	TextFgColor  ui.Attribute
	TextBgColor  ui.Attribute
}

func NewTextView(lines <-chan ToggleText) *TextView {
	t := &TextView{
		Block:       *ui.NewBlock(),
		inputStream: lines,
		render:      make(chan struct{}, 1),
		done:        make(chan struct{}),
		stopped:     make(chan struct{}),
		Text:        []ToggleText{},
		TextOut:     []string{},
		TextFgColor: ui.ThemeAttr("menu.text.fg"),
		TextBgColor: ui.ThemeAttr("menu.text.bg"),
		search:      viewSearch{match: -1},
	}

	// like tail -f: lines use the whole width, with no border, so text
	// selected in the terminal contains only log lines; BorderLabel is shown
	// in a title bar, and a status row and two rows of key help are at the
	// bottom, like nano
	t.Border = false
	t.Height = cwidgets.TermHeight()
	t.Width = cwidgets.TermWidth()

	t.readInputLoop()
	t.renderLoop()
	return t
}

// Adjusts text inside this view according to the window size. No need to call ui.Render(...)
// after calling this method, it is called automatically
func (t *TextView) Resize() {
	ui.Clear()
	t.mu.Lock()
	t.Height = cwidgets.TermHeight()
	t.Width = cwidgets.TermWidth()
	t.mu.Unlock()
	t.requestRender()
}

// Toggles text inside this view. No need to call ui.Render(...) after calling this method,
// it is called automatically
func (t *TextView) Toggle() {
	t.mu.Lock()
	t.toggleState = !t.toggleState
	t.mu.Unlock()
	t.requestRender()
}

// BeforeRender runs f on the render goroutine before the next redraw. Use it
// for termbox calls that write to the output buffer, such as SetInputMode:
// called from another goroutine they race with the Flush of a redraw, and
// their escape sequence can be lost.
func (t *TextView) BeforeRender(f func()) {
	t.mu.Lock()
	prev := t.beforeRender
	t.beforeRender = func() {
		if prev != nil {
			prev()
		}
		f()
	}
	t.mu.Unlock()
	t.requestRender()
}

// SetFooter shows status in the status row ("" leaves it empty) and keys in
// the key help rows.
func (t *TextView) SetFooter(status string, keys []KeyHelp) {
	t.mu.Lock()
	t.status = status
	t.keys = keys
	t.mu.Unlock()
	t.requestRender()
}

// ScrollUp moves the view n lines back in history.
func (t *TextView) ScrollUp(n int) {
	t.mu.Lock()
	t.scroll += n
	t.mu.Unlock()
	t.requestRender()
}

// ScrollDown moves the view n lines towards the newest output. At the bottom
// the view follows new lines again, and can go on to leave blank rows below
// the newest line (see layout); when detached it asks for newer lines.
func (t *TextView) ScrollDown(n int) {
	t.mu.Lock()
	atBottom := t.scroll <= 0
	t.scroll -= n
	if t.detached && t.scroll < 0 {
		t.scroll = 0
	}
	onBottom := t.onBottom
	if !t.detached || !atBottom {
		onBottom = nil
	}
	t.mu.Unlock()
	t.requestRender()
	if onBottom != nil {
		go onBottom()
	}
}

// SetOnTop registers f to be called (in its own goroutine) when the user tries
// to scroll above the oldest loaded line, e.g. to load older history.
func (t *TextView) SetOnTop(f func()) {
	t.mu.Lock()
	t.onTop = f
	t.mu.Unlock()
}

// SetOnBottom registers f to be called (in its own goroutine) when the user
// tries to scroll below the newest loaded line while detached.
func (t *TextView) SetOnBottom(f func()) {
	t.mu.Lock()
	t.onBottom = f
	t.mu.Unlock()
}

// Prepend inserts older lines before the current ones. The visible lines
// stay where they are; the user scrolls up to reach the new ones.
func (t *TextView) Prepend(lines []ToggleText) {
	if len(lines) == 0 {
		return
	}
	t.mu.Lock()
	t.Text = append(append([]ToggleText{}, lines...), t.Text...)
	if t.search.match >= 0 {
		t.search.match += len(lines)
	}
	t.search.anchor += len(lines)
	t.mu.Unlock()
	t.requestRender()
}

// Append adds newer lines after the current ones while detached. The visible
// lines stay where they are.
func (t *TextView) Append(lines []ToggleText) {
	t.mu.Lock()
	t.appendLocked(lines)
	t.mu.Unlock()
	t.requestRender()
}

// Detach stops appending input lines: they are kept aside until Attach.
func (t *TextView) Detach() {
	t.mu.Lock()
	t.detached = true
	t.mu.Unlock()
}

// Replace drops the loaded lines and shows lines instead, detached from the
// live tail (call Attach to join it again).
func (t *TextView) Replace(lines []ToggleText) {
	t.mu.Lock()
	t.Text = append([]ToggleText{}, lines...)
	t.detached = true
	t.scroll = 0
	t.search.match = -1
	t.mu.Unlock()
	t.requestRender()
}

// Attach joins a detached view to the live tail: lines are appended, then
// the input lines received meanwhile that keep returns true for.
func (t *TextView) Attach(lines []ToggleText, keep func(ToggleText) bool) {
	t.mu.Lock()
	t.appendLocked(lines)
	var rest []ToggleText
	for _, l := range t.pending {
		if keep(l) {
			rest = append(rest, l)
		}
	}
	t.appendLocked(rest)
	t.detached = false
	t.pending = nil
	t.mu.Unlock()
	t.requestRender()
}

func (t *TextView) appendLocked(lines []ToggleText) {
	t.Text = append(t.Text, lines...)
	switch {
	case t.scroll == 0 && !t.detached:
		// following: the new lines push the old ones up
	case t.scroll < 0 && !t.detached:
		// the new lines fill the blank rows, then the view follows them
		t.scroll += t.added(lines)
		if t.scroll > 0 {
			t.scroll = 0
		}
	default:
		// keep the view still: the new lines are below it
		t.scroll += t.added(lines)
	}
}

// added returns how many screen rows lines take.
func (t *TextView) added(lines []ToggleText) int {
	n := 0
	for _, l := range lines {
		n += len(splitLine(l.Toggle(t.toggleState), t.textWidth()))
	}
	return n
}

// Detached reports whether the loaded lines stop before the live tail.
func (t *TextView) Detached() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.detached
}

// Oldest returns the oldest loaded line (nil if none) and how many lines are loaded.
func (t *TextView) Oldest() (ToggleText, int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if len(t.Text) == 0 {
		return nil, 0
	}
	return t.Text[0], len(t.Text)
}

// Newest returns the newest loaded line (nil if none).
func (t *TextView) Newest() ToggleText {
	t.mu.Lock()
	defer t.mu.Unlock()
	if len(t.Text) == 0 {
		return nil
	}
	return t.Text[len(t.Text)-1]
}

// ScrollToTop shows the oldest loaded line.
func (t *TextView) ScrollToTop() {
	t.mu.Lock()
	t.scroll = math.MaxInt32 // clamped by layout
	t.mu.Unlock()
	t.requestRender()
}

// ScrollToBottom shows the newest lines and follows new output again. Like
// scrolling down as far as possible, the newest lines go to the top
// (minVisibleRows of them) with blank rows below for the lines to come.
func (t *TextView) ScrollToBottom() {
	t.mu.Lock()
	if t.detached {
		t.scroll = 0
	} else {
		t.scroll = math.MinInt32 // clamped by layout
	}
	t.mu.Unlock()
	t.requestRender()
}

// SetLabel changes the border label and redraws.
func (t *TextView) SetLabel(label string) {
	t.mu.Lock()
	t.BorderLabel = label
	t.mu.Unlock()
	t.requestRender()
}

// PageSize returns the number of text lines visible at once.
func (t *TextView) PageSize() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.textHeight()
}

func (t *TextView) Buffer() ui.Buffer {
	t.mu.Lock()
	defer t.mu.Unlock()
	buf := t.Block.Buffer()

	x0 := t.Block.X
	y := t.Block.Y + titleRows

	for row, line := range t.TextOut {
		marks := t.search.marks(line, t.outSrc[row])
		stamp := 0
		if t.outFirst[row] {
			stamp = timestampRunes(line)
		}
		x := x0
		for i, ch := range []rune(line) {
			cell := ui.Cell{Ch: ch, Fg: t.TextFgColor, Bg: t.TextBgColor}
			if i < stamp {
				cell.Fg = timestampColor
			}
			switch marks[i] {
			case markMatch:
				cell.Fg, cell.Bg = ui.ColorBlack, ui.ColorYellow
			case markCurrent:
				cell.Fg, cell.Bg = ui.ColorBlack, ui.ColorGreen
			}
			buf.Set(x, y, cell)
			x += runewidth.RuneWidth(ch)
		}
		y++
	}

	t.bar(buf, t.Block.Y, t.BorderLabel)
	t.drawFooter(buf)
	return buf
}

// bar draws s in reverse video across the whole width of row y.
func (t *TextView) bar(buf ui.Buffer, y int, s string) {
	attr := ui.ColorDefault | ui.AttrReverse
	x := t.Block.X
	end := t.Block.X + t.Width
	for _, ch := range " " + s {
		if x >= end {
			return
		}
		buf.Set(x, y, ui.Cell{Ch: ch, Fg: attr, Bg: ui.ColorDefault})
		x += runewidth.RuneWidth(ch)
	}
	for ; x < end; x++ {
		buf.Set(x, y, ui.Cell{Ch: ' ', Fg: attr, Bg: ui.ColorDefault})
	}
}

// requestRender never blocks: if a render is already pending it is reused.
// Callers run on the termui event loop, so a blocking send here would freeze
// the whole UI (including Ctrl+C).
func (t *TextView) requestRender() {
	select {
	case t.render <- struct{}{}:
	default:
	}
}

func (t *TextView) textHeight() int {
	h := t.Height - titleRows - footerRows
	if h < 1 {
		h = 1
	}
	return h
}

func (t *TextView) textWidth() int {
	return t.Width
}

// layout rebuilds TextOut for the current size, toggle state and scroll offset.
// It reports whether the user scrolled up past the oldest loaded line.
func (t *TextView) layout() (atTop bool) {
	maxWidth := t.textWidth()
	height := t.textHeight()
	if t.detached && t.scroll < 0 {
		t.scroll = 0
	}
	need := height + t.scroll
	if need < height {
		need = height
	}

	// collect wrapped lines newest first, then reverse once: prepending in
	// the loop is quadratic and froze the view on logs with thousands of lines
	var lines []string
	var src []int
	var first []bool
	i := len(t.Text) - 1
	for ; i >= 0 && len(lines) < need; i-- {
		wrapped := splitLine(t.Text[i].Toggle(t.toggleState), maxWidth)
		for j := len(wrapped) - 1; j >= 0; j-- {
			lines = append(lines, wrapped[j])
			src = append(src, i)
			first = append(first, j == 0)
		}
	}
	for l, r := 0, len(lines)-1; l < r; l, r = l+1, r-1 {
		lines[l], lines[r] = lines[r], lines[l]
		src[l], src[r] = src[r], src[l]
		first[l], first[r] = first[r], first[l]
	}

	// clamp scroll to the available history
	if max := len(lines) - height; t.scroll > 0 && t.scroll >= max && i < 0 {
		atTop = true
		if max < 0 {
			max = 0
		}
		t.scroll = max
	}

	if t.scroll < 0 {
		// scrolled past the newest line: show it higher up, with blank rows
		// below for the lines to come, but keep at least minVisibleRows
		visible := height + t.scroll
		keep := minVisibleRows
		if keep > len(lines) {
			keep = len(lines)
		}
		if keep > height {
			keep = height
		}
		if visible < keep {
			visible = keep
			t.scroll = visible - height
		}
		start := len(lines) - visible
		if start < 0 {
			start = 0
		}
		t.TextOut = lines[start:]
		t.outSrc = src[start:]
		t.outFirst = first[start:]
		return false
	}

	end := len(lines) - t.scroll
	start := end - height
	if start < 0 {
		start = 0
	}
	t.TextOut = lines[start:end]
	t.outSrc = src[start:end]
	t.outFirst = first[start:end]
	return atTop
}

// Stopped is closed once the view stops drawing (after its input stream ends).
func (t *TextView) Stopped() <-chan struct{} {
	return t.stopped
}

func (t *TextView) renderLoop() {
	go func() {
		defer close(t.stopped)
		for {
			select {
			case <-t.render:
			case <-t.done:
				return
			}
			t.mu.Lock()
			atTop := t.layout()
			onTop := t.onTop
			before := t.beforeRender
			t.beforeRender = nil
			t.mu.Unlock()
			if before != nil {
				before()
			}
			ui.Render(t)
			if atTop && onTop != nil {
				go onTop()
			}
		}
	}()
}

func (t *TextView) readInputLoop() {
	go func() {
		for line := range t.inputStream {
			t.mu.Lock()
			if t.detached {
				// drop the oldest: they are the ones most likely to be
				// loaded again when the view reaches the tail
				if len(t.pending) >= maxPending {
					t.pending = t.pending[1:]
				}
				t.pending = append(t.pending, line)
				t.mu.Unlock()
				continue
			}
			t.appendLocked([]ToggleText{line})
			t.mu.Unlock()
			t.requestRender()
		}
		close(t.done)
	}()
}

func splitLine(line string, lineSize int) []string {
	if line == "" {
		return []string{}
	}
	if lineSize < 1 {
		return []string{line}
	}

	var lines []string
	for {
		if len(line) <= lineSize {
			lines = append(lines, line)
			return lines
		}
		lines = append(lines, line[:lineSize])
		line = line[lineSize:]
	}
}
