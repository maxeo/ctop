package main

import (
	ui "github.com/gizak/termui"
)

// Common action keybindings
var keyMap = map[string][]string{
	"up": []string{
		"/sys/kbd/<up>",
		"/sys/kbd/k",
	},
	"down": []string{
		"/sys/kbd/<down>",
		"/sys/kbd/j",
	},
	"pgup": []string{
		"/sys/kbd/<previous>",
		"/sys/kbd/C-<up>",
	},
	"pgdown": []string{
		"/sys/kbd/<next>",
		"/sys/kbd/C-<down>",
	},
	"exit": []string{
		"/sys/kbd/q",
		"/sys/kbd/C-c",
		"/sys/kbd/<escape>",
	},
	"help": []string{
		"/sys/kbd/h",
		"/sys/kbd/?",
	},
}

// Apply a common handler function to all given keys
func HandleKeys(i string, f func()) {
	for _, k := range keyMap[i] {
		ui.Handle(k, func(ui.Event) { f() })
	}
}

// HandleWheel calls up and down for the mouse wheel. The mouse is reported
// only on Windows (see mouseInput): elsewhere the terminal sends the wheel as
// arrow keys.
func HandleWheel(up, down func()) {
	ui.Handle("/sys/mouse", func(e ui.Event) {
		if me, ok := e.Data.(ui.EvtMouse); ok {
			switch me.Press {
			case "MouseWheelUp":
				up()
			case "MouseWheelDown":
				down()
			}
		}
	})
}
