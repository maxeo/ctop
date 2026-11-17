package widgets

import (
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

// PageSize returns the number of text lines visible at once.
func (t *TextView) PageSize() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.textHeight()
}

func (t *TextView) Buffer() ui.Buffer {
	var cell ui.Cell
	buf := t.Block.Buffer()

	x := t.Block.X + t.padding[0]
	y := t.Block.Y + t.padding[1]

	t.mu.Lock()
	defer t.mu.Unlock()
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
func (t *TextView) layout() {
	maxWidth := t.Width - (t.padding[0] * 2)
	height := t.textHeight()
	need := height + t.scroll

	var lines []string
	for i := len(t.Text) - 1; i >= 0 && len(lines) < need; i-- {
		lines = append(splitLine(t.Text[i].Toggle(t.toggleState), maxWidth), lines...)
	}

	// clamp scroll to the available history
	if max := len(lines) - height; t.scroll > max {
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
			t.layout()
			t.mu.Unlock()
			ui.Render(t)
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
