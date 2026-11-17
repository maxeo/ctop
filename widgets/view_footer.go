package widgets

import (
	ui "github.com/gizak/termui"
)

const (
	titleRows  = 1               // title bar
	footerRows = 1 + KeyHelpRows // status row and key help, when it fits in two rows
	// rows of text kept on screen when scrolling past the newest line
	minVisibleRows = 20
)

// drawFooter draws the status row (in reverse video when not empty) and the
// key help below it, like nano.
func (t *TextView) drawFooter(buf ui.Buffer) {
	rows := t.keyRowsLocked()
	y := t.Block.Y + t.Height - 1 - rows
	if t.status != "" {
		t.bar(buf, y, t.status)
	}
	drawKeyHelp(buf, t.Block.X, y+1, t.Width, rows, t.keys, t.TextFgColor, t.TextBgColor)
}

// keyRowsLocked returns the rows of the key help: two, or three when the
// keys of the current mode do not fit in two at this width.
func (t *TextView) keyRowsLocked() int {
	if keyHelpFits(t.keys, t.Width, KeyHelpRows) {
		return KeyHelpRows
	}
	return KeyHelpRows + 1
}
