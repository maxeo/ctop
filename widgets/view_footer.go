package widgets

import (
	ui "github.com/gizak/termui"
)

const (
	titleRows  = 1               // title bar
	footerRows = 1 + KeyHelpRows // status row and key help
	// rows of text kept on screen when scrolling past the newest line
	minVisibleRows = 20
)

// drawFooter draws the status row (in reverse video when not empty) and the
// key help below it, like nano.
func (t *TextView) drawFooter(buf ui.Buffer) {
	y := t.Block.Y + t.Height - footerRows
	if t.status != "" {
		t.bar(buf, y, t.status)
	}
	drawKeyHelp(buf, t.Block.X, y+1, t.Width, KeyHelpRows, t.keys, t.TextFgColor, t.TextBgColor)
}
