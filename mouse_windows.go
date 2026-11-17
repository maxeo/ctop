package main

import tm "github.com/nsf/termbox-go"

// On Windows the console does not turn the mouse wheel into arrow keys (as
// xterm-like terminals do in full screen), so ctop asks for mouse events and
// handles the wheel itself (see HandleWheel). Text is then selected with
// Shift + drag.
const mouseInput = tm.InputMouse
