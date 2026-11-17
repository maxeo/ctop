package widgets

import (
	ui "github.com/gizak/termui"
	"github.com/mattn/go-runewidth"
)

const (
	titleRows  = 1 // title bar
	footerRows = 3 // status row and two rows of key help
	// rows of text kept on screen when scrolling past the newest line
	minVisibleRows = 20
)

// KeyHelp is one entry of the key help at the bottom of a TextView: Key is
// shown in reverse video, like nano's "^X Exit".
type KeyHelp struct {
	Key   string
	Label string
}

// drawFooter draws the status row (in reverse video when not empty) and the
// key help in two rows of equal columns, filled top to bottom, like nano.
func (t *TextView) drawFooter(buf ui.Buffer) {
	y := t.Block.Y + t.Height - footerRows
	if t.status != "" {
		t.bar(buf, y, t.status)
	}

	if len(t.keys) == 0 {
		return
	}
	cols := (len(t.keys) + 1) / 2
	colWidth := t.Width / cols
	if colWidth < 4 {
		colWidth = 4
	}
	reverse := ui.ColorDefault | ui.AttrReverse
	for i, k := range t.keys {
		x := t.Block.X + (i/2)*colWidth
		row := y + 1 + i%2
		end := x + colWidth - 1
		if end > t.Block.X+t.Width {
			end = t.Block.X + t.Width
		}
		x = t.text(buf, x, row, end, k.Key, reverse)
		t.text(buf, x, row, end, " "+k.Label, t.TextFgColor)
	}
}

// text draws s from x on row y, stopping before end, and returns the next x.
func (t *TextView) text(buf ui.Buffer, x, y, end int, s string, fg ui.Attribute) int {
	for _, ch := range s {
		w := runewidth.RuneWidth(ch)
		if x+w > end {
			break
		}
		buf.Set(x, y, ui.Cell{Ch: ch, Fg: fg, Bg: t.TextBgColor})
		x += w
	}
	return x
}
