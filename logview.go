package main

import (
	"sync"
	"time"
	"unicode/utf8"

	"github.com/eqms/ctop/widgets"
	ui "github.com/gizak/termui"
	tm "github.com/nsf/termbox-go"
)

// Input modes of the log view. termbox's Alt mode (used by the rest of ctop)
// never reports a lone Esc: it waits for the next key and sends both as
// Alt+key. The log view uses Esc mode instead, and tells Esc apart from the
// start of an escape sequence termbox doesn't decode with escWait.
//
// Normally mouse reporting is on, so the wheel scrolls the log in any
// terminal. A click (or [s]) switches to selection mode: mouse reporting off,
// so the terminal selects text with click and drag.
const (
	logInputNormal = tm.InputEsc | tm.InputMouse
	logInputSelect = tm.InputEsc
)

// a terminal sends the bytes of an escape sequence together; a key typed by
// hand after Esc takes much longer
const escWait = 50 * time.Millisecond

// custom event sent escWait after <escape>
const escEvent = "/logview/esc"

// key names sent by termui for Backspace: termbox reports 0x08 as
// <backspace> and 0x7F (what most terminals send) as Ctrl+8
var backspaceKeys = map[string]bool{
	"<backspace>": true,
	"C-8":         true,
}

// logView handles the keys of the log view. It has three modes: normal
// (scroll, search keys, anything else closes), the search prompt (keys are
// typed into the query) and text selection (mouse reporting off).
type logView struct {
	view    *widgets.TextView
	history *logHistory // nil if the collector has no history
	close   func()

	csi       csiKeys
	escSeq    int // counts <escape> keys
	escID     int // escSeq of the <escape> waiting for escWait, 0 if none
	prompting bool
	query     string // being typed in the prompt
	matched   bool   // a loaded line contains query
	selecting bool
	searchMu  sync.Mutex // one search at a time
}

func (v *logView) key(k string) {
	if v.escID != 0 {
		// <escape> just before: the start of a sequence, or Esc then a key
		v.escID = 0
		if k == "[" || k == "O" {
			v.csi.Start()
			return
		}
		v.escape()
	}
	if k == "<escape>" {
		v.escSeq++
		v.escID = v.escSeq
		id := v.escID
		time.AfterFunc(escWait, func() { ui.SendCustomEvt(escEvent, id) })
		return
	}
	// the rest of a Home/End escape sequence, in any mode
	if k == "M-[" || k == "M-O" {
		v.csi.Start()
		return
	}
	if action, consumed := v.csi.Feed(k); consumed {
		switch action {
		case "home":
			v.home()
		case "end":
			v.end()
		}
		return
	}

	switch {
	case v.prompting:
		v.promptKey(k)
	case v.navKey(k):
	case v.selecting:
		v.selectKey(k)
	default:
		v.normalKey(k)
	}
}

// escTimeout handles the event sent escWait after <escape>: no key followed,
// so it was the Esc key.
func (v *logView) escTimeout(id int) {
	if id == v.escID {
		v.escID = 0
		v.escape()
	}
}

// escape handles the Esc key: it leaves the prompt, the selection mode or the
// search, in this order, otherwise it closes the view.
func (v *logView) escape() {
	switch {
	case v.prompting:
		v.prompting = false
		v.view.CancelTyping()
		v.showFooter("")
	case v.selecting:
		v.setSelecting(false)
	case v.view.SearchQuery() != "":
		v.view.SetSearch("")
		v.showFooter("")
	default:
		v.close()
	}
}

// mouse handles a mouse event (only reported outside selection mode).
func (v *logView) mouse(press string) {
	switch press {
	case "MouseWheelUp":
		v.view.ScrollUp(logWheelStep)
	case "MouseWheelDown":
		v.view.ScrollDown(logWheelStep)
	case "MouseLeft", "MouseRight", "MouseMiddle":
		// the user wants to select text: give the mouse to the terminal
		if !v.selecting && !v.prompting {
			v.setSelecting(true)
		}
	}
}

// navKey handles the keys that move the view, in normal and select mode.
func (v *logView) navKey(k string) bool {
	switch k {
	case "<up>", "k":
		v.view.ScrollUp(1)
	case "<down>", "j":
		v.view.ScrollDown(1)
	case "<previous>", "C-<up>":
		v.view.ScrollUp(v.view.PageSize())
	case "<next>", "C-<down>":
		v.view.ScrollDown(v.view.PageSize())
	case "<home>":
		v.home()
	case "<end>":
		v.end()
	case "t":
		v.view.Toggle()
	default:
		return false
	}
	return true
}

func (v *logView) normalKey(k string) {
	searching := v.view.SearchQuery() != ""
	switch {
	case k == "C-f" || k == "/":
		v.prompting = true
		v.query = ""
		v.matched = true
		v.view.BeginTyping()
		v.view.SearchTyping("")
		v.showFooter("")
	case k == "s":
		v.setSelecting(true)
	case searching && k == "n":
		go v.find(true)
	case searching && k == "N":
		go v.find(false)
	default:
		v.close()
	}
}

func (v *logView) selectKey(k string) {
	switch k {
	case "s":
		v.setSelecting(false)
	case "q", "<left>", "<right>":
		v.close()
	}
	// other keys do nothing, so copy shortcuts (Ctrl+C, Ctrl+Insert) that
	// reach the application don't close the view
}

// promptKey handles a key typed in the search prompt. The loaded lines are
// searched at every key; Enter keeps the search (loading more of the log if
// no loaded line matches), Esc drops it and goes back where the view was.
func (v *logView) promptKey(k string) {
	switch {
	case k == "<enter>":
		v.prompting = false
		if v.query == "" {
			v.showFooter("")
			return
		}
		if v.matched {
			v.showFooter("")
		} else {
			go v.findAnywhere()
		}
		return
	case backspaceKeys[k]:
		if v.query == "" {
			return
		}
		_, size := utf8.DecodeLastRuneInString(v.query)
		v.query = v.query[:len(v.query)-size]
	case k == "C-u":
		v.query = ""
	case k == "<space>":
		v.query += " "
	case utf8.RuneCountInString(k) == 1:
		v.query += k
	default:
		return
	}
	v.matched = v.view.SearchTyping(v.query)
	v.showFooter("")
}

// key help of each mode, shown at the bottom like nano's
var (
	normalKeys = []widgets.KeyHelp{
		{Key: "^F", Label: "Search"},
		{Key: "t", Label: "Timestamps"},
		{Key: "Home", Label: "Start of log"},
		{Key: "End", Label: "Newest lines"},
		{Key: "PgUp", Label: "Page up"},
		{Key: "PgDn", Label: "Page down"},
		{Key: "s", Label: "Select text"},
		{Key: "Click", Label: "Select text"},
		{Key: "q", Label: "Close"},
		{Key: "← →", Label: "Close"},
	}
	searchKeys = []widgets.KeyHelp{
		{Key: "n", Label: "Older match"},
		{Key: "N", Label: "Newer match"},
		{Key: "^F", Label: "New search"},
		{Key: "Esc", Label: "End search"},
		{Key: "Home", Label: "Start of log"},
		{Key: "End", Label: "Newest lines"},
		{Key: "PgUp", Label: "Page up"},
		{Key: "PgDn", Label: "Page down"},
		{Key: "q", Label: "Close"},
		{Key: "← →", Label: "Close"},
	}
	promptKeys = []widgets.KeyHelp{
		{Key: "Enter", Label: "Search"},
		{Key: "Esc", Label: "Cancel"},
		{Key: "^U", Label: "Clear"},
	}
	selectKeys = []widgets.KeyHelp{
		{Key: "Esc", Label: "Back"},
		{Key: "s", Label: "Back"},
		{Key: "PgUp", Label: "Page up"},
		{Key: "PgDn", Label: "Page down"},
		{Key: "Home", Label: "Start of log"},
		{Key: "End", Label: "Newest lines"},
		{Key: "q", Label: "Close"},
		{Key: "← →", Label: "Close"},
	}
)

// showFooter shows the status and key help of the current mode; msg (a
// search result, or "") goes first in the status row.
func (v *logView) showFooter(msg string) {
	var status string
	var keys []widgets.KeyHelp
	switch {
	case v.prompting:
		status = "Search: " + v.query + "_"
		if v.query != "" && !v.matched {
			status += "   (no match in the loaded lines, Enter searches the whole log)"
		}
		keys = promptKeys
	case v.selecting:
		status = "SELECTION: drag with the mouse to select, the terminal copies"
		keys = selectKeys
	case v.view.SearchQuery() != "":
		status = "Search: \"" + v.view.SearchQuery() + "\""
		keys = searchKeys
	default:
		keys = normalKeys
	}
	if msg != "" {
		if status != "" {
			status = msg + "   " + status
		} else {
			status = msg
		}
	}
	v.view.SetFooter(status, keys)
}

// find moves to the next match, loading more of the log if no loaded line
// matches: everything before the loaded lines when searching older ones, the
// rest of the log when searching newer ones from the start of the log.
func (v *logView) find(older bool) {
	if !v.searchMu.TryLock() {
		return
	}
	defer v.searchMu.Unlock()

	msg := ""
	if !v.findLoading(older) {
		if older {
			msg = "no older match"
		} else {
			msg = "no newer match"
		}
	}
	v.showFooter(msg)
}

// findAnywhere looks in the whole log: older lines first, then newer ones.
func (v *logView) findAnywhere() {
	if !v.searchMu.TryLock() {
		return
	}
	defer v.searchMu.Unlock()

	msg := ""
	if !v.findLoading(true) && !v.findLoading(false) {
		msg = "no match"
	}
	v.showFooter(msg)
}

func (v *logView) findLoading(older bool) bool {
	if v.view.FindNext(older) {
		return true
	}
	if v.history == nil {
		return false
	}
	var loaded bool
	if older {
		loaded = v.history.LoadAll()
	} else {
		loaded = v.history.LoadRest()
	}
	return loaded && v.view.FindNext(older)
}

func (v *logView) setSelecting(on bool) {
	v.selecting = on
	mode := logInputNormal
	if on {
		mode = logInputSelect
	}
	v.view.BeforeRender(func() { tm.SetInputMode(mode) })
	v.showFooter("")
}

func (v *logView) home() {
	if v.history != nil {
		go v.history.Head()
	} else {
		v.view.ScrollToTop()
	}
}

func (v *logView) end() {
	if v.history != nil {
		go v.history.Tail()
	} else {
		v.view.ScrollToBottom()
	}
}
