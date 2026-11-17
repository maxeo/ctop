//go:build !windows

package main

import tm "github.com/nsf/termbox-go"

// Elsewhere the mouse is left to the terminal, which turns the wheel into
// arrow keys in full screen and keeps text selection as usual (see
// mouse_windows.go).
const mouseInput tm.InputMode = 0
