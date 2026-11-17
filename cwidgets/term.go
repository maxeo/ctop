package cwidgets

import (
	tm "github.com/nsf/termbox-go"
)

// TermWidth returns the current terminal width.
//
// Use this instead of ui.TermWidth(): termui's version calls tm.Sync(), which
// clears the physical screen and repaints every cell. Called on each refresh
// tick (and on every cursor move) that caused visible flicker and lag.
// tm.Size() reports the same value without touching the screen; termbox
// refreshes it on every Clear/Flush.
func TermWidth() int {
	w, _ := tm.Size()
	return w
}

// TermHeight returns the current terminal height. See TermWidth.
func TermHeight() int {
	_, h := tm.Size()
	return h
}
