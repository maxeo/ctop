package widgets

import (
	"strings"

	ui "github.com/gizak/termui"
)

var (
	input_chars = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ-_./:@"
)

type Padding [2]int // x,y padding

type Input struct {
	ui.Block
	Label       string
	Data        string
	MaxLen      int
	TextFgColor ui.Attribute
	TextBgColor ui.Attribute
	onChange    func(string) // called with the text after each change
	padding     Padding
}

func NewInput() *Input {
	i := &Input{
		Block:       *ui.NewBlock(),
		Label:       "input",
		MaxLen:      20,
		TextFgColor: ui.ThemeAttr("menu.text.fg"),
		TextBgColor: ui.ThemeAttr("menu.text.bg"),
		padding:     Padding{4, 2},
	}
	i.BorderFg = ui.ThemeAttr("menu.border.fg")
	i.BorderLabelFg = ui.ThemeAttr("menu.label.fg")
	i.calcSize()
	return i
}

func (i *Input) calcSize() {
	i.Height = 3 // minimum height
	i.Width = i.MaxLen + (i.padding[0] * 2)
}

// SetMaxLen updates MaxLen and recalculates widget dimensions
func (i *Input) SetMaxLen(n int) {
	i.MaxLen = n
	i.calcSize()
}

func (i *Input) Buffer() ui.Buffer {
	var cell ui.Cell
	buf := i.Block.Buffer()

	x := i.Block.X + i.padding[0]
	y := i.Block.Y + 1
	for _, ch := range i.Data {
		cell = ui.Cell{Ch: ch, Fg: i.TextFgColor, Bg: i.TextBgColor}
		buf.Set(x, y, cell)
		x++
	}

	return buf
}

// OnChange registers f, called on the event loop with the text after each
// change. (It replaces a channel that KeyPress fed synchronously: a slow
// reader blocked the whole event loop.)
func (i *Input) OnChange(f func(string)) {
	i.onChange = f
}

func (i *Input) KeyPress(e ui.Event) {
	ch := strings.ReplaceAll(e.Path, "/sys/kbd/", "")
	switch {
	case ch == "C-8" || ch == "<backspace>": // 0x7F or 0x08, depending on the terminal
		if len(i.Data) == 0 {
			return
		}
		i.Data = i.Data[:len(i.Data)-1]
	case ch == "C-u":
		if i.Data == "" {
			return
		}
		i.Data = ""
	case len(ch) == 1 && strings.Contains(input_chars, ch) && len(i.Data) < i.MaxLen:
		i.Data += ch
	default:
		return
	}
	if i.onChange != nil {
		i.onChange(i.Data)
	}
	ui.Render(i)
}

// Setup some default handlers for menu navigation
func (i *Input) InputHandlers() {
	ui.Handle("/sys/kbd/", i.KeyPress)
}
