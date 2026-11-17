package widgets

import (
	"image"

	"github.com/eqms/ctop/cwidgets"
	ui "github.com/gizak/termui"
	"github.com/mattn/go-runewidth"
)

// KeyHelp is one entry of a key help, shown like nano's "^X Exit": Key in
// reverse video, then Label.
type KeyHelp struct {
	Key   string
	Label string
}

// KeyHelpRows is the height of a key help.
const KeyHelpRows = 2

// drawKeyHelp draws keys in rows of equal columns, filled top to bottom like
// nano, from (x, y) over width cells.
func drawKeyHelp(buf ui.Buffer, x, y, width, rows int, keys []KeyHelp, fg, bg ui.Attribute) {
	if len(keys) == 0 || width <= 0 || rows <= 0 {
		return
	}
	cols := (len(keys) + rows - 1) / rows
	colWidth := width / cols
	if colWidth < 4 {
		colWidth = 4
	}
	reverse := ui.ColorDefault | ui.AttrReverse
	for i, k := range keys {
		cx := x + (i/rows)*colWidth
		row := y + i%rows
		end := cx + colWidth - 1
		if end > x+width {
			end = x + width
		}
		cx = drawText(buf, cx, row, end, k.Key, reverse, bg)
		drawText(buf, cx, row, end, " "+k.Label, fg, bg)
	}
}

// keyHelpFits reports whether drawKeyHelp shows every key and label whole
// in the given rows.
func keyHelpFits(keys []KeyHelp, width, rows int) bool {
	if len(keys) == 0 {
		return true
	}
	cols := (len(keys) + rows - 1) / rows
	colWidth := width / cols
	for _, k := range keys {
		if runewidth.StringWidth(k.Key)+1+runewidth.StringWidth(k.Label) > colWidth-1 {
			return false
		}
	}
	return true
}

// drawText draws s from x on row y, stopping before end, and returns the next x.
func drawText(buf ui.Buffer, x, y, end int, s string, fg, bg ui.Attribute) int {
	for _, ch := range s {
		w := runewidth.RuneWidth(ch)
		if x+w > end {
			break
		}
		buf.Set(x, y, ui.Cell{Ch: ch, Fg: fg, Bg: bg})
		x += w
	}
	return x
}

// width of a column of the full key list
const allKeysColWidth = 28

// KeyBar shows a key help on the last rows of the terminal: two rows of
// keys, or every key (All) when ShowAll is set.
type KeyBar struct {
	Keys    []KeyHelp
	All     []KeyHelp
	ShowAll bool
}

func NewKeyBar(keys, all []KeyHelp) *KeyBar {
	return &KeyBar{Keys: keys, All: all}
}

// keys returns the keys shown and the rows they take.
func (k *KeyBar) keys() ([]KeyHelp, int) {
	if !k.ShowAll || len(k.All) == 0 {
		return k.Keys, KeyHelpRows
	}
	cols := (cwidgets.TermWidth() - 1) / allKeysColWidth
	if cols < 1 {
		cols = 1
	}
	rows := (len(k.All) + cols - 1) / cols
	if rows < KeyHelpRows {
		rows = KeyHelpRows
	}
	return k.All, rows
}

// Height returns the rows taken by the key bar.
func (k *KeyBar) Height() int {
	_, rows := k.keys()
	return rows
}

func (k *KeyBar) Buffer() ui.Buffer {
	keys, rows := k.keys()
	w, h := cwidgets.TermWidth(), cwidgets.TermHeight()
	y := h - rows
	buf := ui.NewBuffer()
	// termui only draws the cells inside the buffer area
	buf.SetArea(image.Rect(0, y, w, h))
	for row := y; row < h; row++ {
		for x := 0; x < w; x++ {
			buf.Set(x, row, ui.Cell{Ch: ' ', Fg: ui.ColorDefault, Bg: ui.ColorDefault})
		}
	}
	drawKeyHelp(buf, 1, y, w-1, rows, keys, ui.ThemeAttr("par.text.fg"), ui.ColorDefault)
	return buf
}
