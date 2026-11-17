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

type TextView struct {
	ui.Block
	inputStream <-chan ToggleText
	render      chan struct{} // buffered(1): pending render requests are coalesced
	done        chan struct{} // closed when the input stream ends
	mu          sync.Mutex    // guards the fields below, shared by input, render and event goroutines
	toggleState bool
	scroll      int          // lines scrolled back from the bottom (0 = follow tail)
	onTop       func()       // called when the user scrolls past the oldest loaded line
	Text        []ToggleText // all the text
	TextOut     []string     // text to be displayed
	TextFgColor ui.Attribute
	TextBgColor ui.Attribute
	padding     Padding
}

func NewTextView(lines <-chan ToggleText) *TextView {
	t := &TextView{
		Block:       *ui.NewBlock(),
		inputStream: lines,
		render:      make(chan struct{}, 1),
		done:        make(chan struct{}),
		Text:        []ToggleText{},
		TextOut:     []string{},
		TextFgColor: ui.ThemeAttr("menu.text.fg"),
		TextBgColor: ui.ThemeAttr("menu.text.bg"),
		padding:     Padding{4, 2},
	}

	t.BorderFg = ui.ThemeAttr("menu.border.fg")
	t.BorderLabelFg = ui.ThemeAttr("menu.label.fg")
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

// ScrollUp moves the view n lines back in history.
func (t *TextView) ScrollUp(n int) {
	t.mu.Lock()
	t.scroll += n
	t.mu.Unlock()
	t.requestRender()
}

// ScrollDown moves the view n lines towards the newest output. At the bottom
// the view follows new lines again.
func (t *TextView) ScrollDown(n int) {
	t.mu.Lock()
	t.scroll -= n
	if t.scroll < 0 {
		t.scroll = 0
	}
	t.mu.Unlock()
	t.requestRender()
}

// SetOnTop registers f to be called (in its own goroutine) when the user tries
// to scroll above the oldest loaded line, e.g. to load older history.
func (t *TextView) SetOnTop(f func()) {
	t.mu.Lock()
	t.onTop = f
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
	t.mu.Unlock()
	t.requestRender()
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

// ScrollToTop shows the oldest loaded line.
func (t *TextView) ScrollToTop() {
	t.mu.Lock()
	t.scroll = math.MaxInt32 // clamped by layout
	t.mu.Unlock()
	t.requestRender()
}

// ScrollToBottom shows the newest line and follows new output again.
func (t *TextView) ScrollToBottom() {
	t.mu.Lock()
	t.scroll = 0
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
	var cell ui.Cell
	t.mu.Lock()
	defer t.mu.Unlock()
	buf := t.Block.Buffer()

	x := t.Block.X + t.padding[0]
	y := t.Block.Y + t.padding[1]

	for _, line := range t.TextOut {
		for _, ch := range line {
			cell = ui.Cell{Ch: ch, Fg: t.TextFgColor, Bg: t.TextBgColor}
			buf.Set(x, y, cell)
			x = x + runewidth.RuneWidth(ch)
		}
		x = t.Block.X + t.padding[0]
		y++
	}
	return buf
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
	h := t.Height - (t.padding[1] * 2)
	if h < 1 {
		h = 1
	}
	return h
}

// layout rebuilds TextOut for the current size, toggle state and scroll offset.
// It reports whether the user scrolled up past the oldest loaded line.
func (t *TextView) layout() (atTop bool) {
	maxWidth := t.Width - (t.padding[0] * 2)
	height := t.textHeight()
	need := height + t.scroll

	// collect wrapped lines newest first, then reverse once: prepending in
	// the loop is quadratic and froze the view on logs with thousands of lines
	var lines []string
	i := len(t.Text) - 1
	for ; i >= 0 && len(lines) < need; i-- {
		wrapped := splitLine(t.Text[i].Toggle(t.toggleState), maxWidth)
		for j := len(wrapped) - 1; j >= 0; j-- {
			lines = append(lines, wrapped[j])
		}
	}
	for l, r := 0, len(lines)-1; l < r; l, r = l+1, r-1 {
		lines[l], lines[r] = lines[r], lines[l]
	}

	// clamp scroll to the available history
	if max := len(lines) - height; t.scroll > 0 && t.scroll >= max && i < 0 {
		atTop = true
		if max < 0 {
			max = 0
		}
		t.scroll = max
	}

	end := len(lines) - t.scroll
	start := end - height
	if start < 0 {
		start = 0
	}
	t.TextOut = lines[start:end]
	return atTop
}

func (t *TextView) renderLoop() {
	go func() {
		for {
			select {
			case <-t.render:
			case <-t.done:
				return
			}
			t.mu.Lock()
			atTop := t.layout()
			onTop := t.onTop
			t.mu.Unlock()
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
			t.Text = append(t.Text, line)
			if t.scroll > 0 {
				// keep the scrolled-back view still while new lines arrive
				t.scroll += len(splitLine(line.Toggle(t.toggleState), t.Width-(t.padding[0]*2)))
			}
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
