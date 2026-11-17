package main

import (
	"sync"
	"time"
	"unicode/utf8"

	ui "github.com/gizak/termui"
	"github.com/maxeo/ctop/search"
	"github.com/maxeo/ctop/widgets"
	tm "github.com/nsf/termbox-go"
)

// Input modes of the log view. termbox's Alt mode never reports a lone Esc:
// it waits for the next key and sends both as Alt+key. Like the rest of ctop
// (see inputMode), the log view uses Esc mode instead, and tells Esc apart
// from the start of an escape sequence termbox doesn't decode with escWait.
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
	query     string      // being typed in the prompt
	mode      search.Mode // how the query is matched, changed with Tab
	matched   bool        // a loaded line matches query
	badQuery  string      // why query cannot be searched (invalid regex), "" if it can
	moved     bool        // the view was scrolled while typing: search from there
	older     bool        // the user last moved towards older lines: Enter finds the previous match
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
			v.moved = v.prompting
			v.older = false
		case "end":
			v.end()
			v.moved = v.prompting
			v.older = true
		}
		return
	}

	if k == "C-c" {
		// Ctrl+C works like Esc: back one level
		v.escape()
		return
	}

	switch {
	case v.prompting && k == "C-s":
		// Ctrl+S cannot be typed: it switches the selection mode, the
		// prompt stays open
		v.setSelecting(!v.selecting)
	case v.prompting && isMoveKey(k) && v.navKey(k):
		// scrolling while typing: the next key searches from the new position
		v.moved = true
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
		if v.moved {
			// stay where the user scrolled to
			v.view.SetSearch(nil)
		} else {
			v.view.CancelTyping()
		}
		v.showFooter("")
	case v.selecting:
		v.setSelecting(false)
	case v.view.Searching() != nil:
		v.view.SetSearch(nil)
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
		v.moved = v.prompting
		v.older = true
	case "MouseWheelDown":
		v.view.ScrollDown(logWheelStep)
		v.moved = v.prompting
		v.older = false
	case "MouseLeft", "MouseRight", "MouseMiddle":
		// the user wants to select text: give the mouse to the terminal,
		// also while typing a search (the prompt stays open)
		if !v.selecting {
			v.setSelecting(true)
		}
	}
}

// isMoveKey reports whether k is a key that moves the view and cannot be
// typed into the search prompt (unlike "k", "j" or "t").
func isMoveKey(k string) bool {
	switch k {
	case "<up>", "<down>", "<previous>", "<next>", "C-<up>", "C-<down>", "<home>", "<end>":
		return true
	}
	return false
}

// navKey handles the keys that move the view, in normal and select mode, and
// the move keys (see isMoveKey) while typing a search.
func (v *logView) navKey(k string) bool {
	switch k {
	case "<up>", "k":
		v.view.ScrollUp(1)
		v.older = true
	case "<down>", "j":
		v.view.ScrollDown(1)
		v.older = false
	case "<previous>", "C-<up>":
		v.view.ScrollUp(v.view.PageSize())
		v.older = true
	case "<next>", "C-<down>":
		v.view.ScrollDown(v.view.PageSize())
		v.older = false
	case "<home>":
		v.home()
		v.older = false
	case "<end>":
		v.end()
		v.older = true
	case "t":
		v.view.Toggle()
	default:
		return false
	}
	return true
}

func (v *logView) normalKey(k string) {
	switch {
	case k == "C-f" || k == "/":
		v.startPrompt()
	case k == "s" || k == "C-s":
		v.setSelecting(true)
	case v.searchKey(k):
	default:
		v.close()
	}
}

// searchKey handles the keys of a search shown: n older match, N newer
// match, Enter the next match in the direction the user last scrolled, Tab
// the next search mode.
func (v *logView) searchKey(k string) bool {
	m := v.view.Searching()
	if m == nil {
		return false
	}
	switch k {
	case "n":
		v.older = true
	case "N":
		v.older = false
	case "<enter>":
	case "<tab>":
		v.nextMode(m.Query)
		return true
	default:
		return false
	}
	go v.find(v.older)
	return true
}

// nextMode searches query with the next mode. If query is not a valid
// regular expression, it says so and the mode after Regex is tried.
func (v *logView) nextMode(query string) {
	msg := ""
	v.mode = v.mode.Next()
	m, err := search.New(v.mode, query)
	if err != nil {
		msg = "not a valid regex: " + err.Error()
		v.mode = v.mode.Next()
		m, _ = search.New(v.mode, query)
	}
	v.view.SetSearch(m)
	v.showFooter(msg)
}

// startPrompt opens the search prompt. In selection mode the mouse stays
// with the terminal, so text can still be selected while typing.
func (v *logView) startPrompt() {
	v.prompting = true
	v.moved = false
	v.query = ""
	v.matched = true
	v.badQuery = ""
	v.view.BeginTyping()
	v.view.SearchTyping(nil)
	v.showFooter("")
}

func (v *logView) selectKey(k string) {
	switch {
	case k == "s" || k == "C-s":
		v.setSelecting(false)
	case k == "C-f" || k == "/":
		v.startPrompt()
	case v.searchKey(k):
	case k == "q" || k == "<left>" || k == "<right>":
		v.close()
	}
	// other keys do nothing, so copy shortcuts (Ctrl+Insert) that reach the
	// application don't close the view
}

// promptKey handles a key typed in the search prompt. The loaded lines are
// searched at every key and when Tab changes the mode; a regular expression
// that is not valid yet (e.g. half typed) shows its error instead. Enter keeps
// the search (loading more of the log if no loaded line matches), Esc drops
// it and goes back where the view was.
func (v *logView) promptKey(k string) {
	switch {
	case k == "<enter>":
		m, err := search.New(v.mode, v.query)
		if err != nil {
			v.badQuery = err.Error()
			v.showFooter("")
			return
		}
		v.prompting = false
		if m == nil {
			v.view.SetSearch(nil)
			v.showFooter("")
			return
		}
		if v.matched {
			v.showFooter("")
		} else {
			go v.findAnywhere()
		}
		return
	case k == "<tab>":
		v.mode = v.mode.Next()
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
	v.badQuery = ""
	m, err := search.New(v.mode, v.query)
	if err != nil {
		// a regex not valid yet: say why, and highlight nothing
		v.badQuery = err.Error()
		v.matched = true
		v.view.SearchTyping(nil)
	} else {
		v.matched = v.searchTyping(m)
	}
	v.showFooter("")
}

// searchTyping shows the closest loaded match of m while the prompt is open.
func (v *logView) searchTyping(m *search.Matcher) bool {
	if v.moved {
		// look for the query from where the user scrolled to
		v.moved = false
		v.view.BeginTyping()
	}
	return v.view.SearchTyping(m)
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
		{Key: "s ^S", Label: "Select text"},
		{Key: "Click", Label: "Select text"},
		{Key: "q", Label: "Close"},
		{Key: "← →", Label: "Close"},
	}
	searchKeys = []widgets.KeyHelp{
		{Key: "n", Label: "Older match"},
		{Key: "N", Label: "Newer match"},
		{Key: "Enter", Label: "Next, same way"},
		{Key: "Tab", Label: "Mode"}, // the label shows the mode, see withMode
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
		{Key: "Tab", Label: "Mode"}, // the label shows the mode, see withMode
		{Key: "n N", Label: "after Enter"},
		{Key: "Esc", Label: "Cancel"},
		{Key: "^U", Label: "Clear"},
		{Key: "↑ ↓", Label: "Scroll"},
		{Key: "PgUp", Label: "Page up"},
		{Key: "PgDn", Label: "Page down"},
		{Key: "Home", Label: "Start of log"},
		{Key: "End", Label: "Newest lines"},
		{Key: "^S Click", Label: "Select text"},
	}
	selectKeys = []widgets.KeyHelp{
		{Key: "Esc", Label: "Back"},
		{Key: "s ^S", Label: "Back"},
		{Key: "^F", Label: "Search"},
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
		status = "Search " + modeChoice(v.mode) + ": " + v.query + "_"
		switch {
		case v.badQuery != "":
			status += "   (" + v.badQuery + ")"
		case v.query != "" && !v.matched:
			status += "   (no match in the loaded lines, Enter searches the whole log)"
		}
		keys = promptKeys
	case v.selecting:
		status = "SELECTION: drag with the mouse to select, the terminal copies"
		keys = selectKeys
		if m := v.view.Searching(); m != nil {
			status += "   " + searchStatus(m)
			keys = append([]widgets.KeyHelp{
				{Key: "n", Label: "Older match"},
				{Key: "N", Label: "Newer match"},
				{Key: "Enter", Label: "Next, same way"},
				{Key: "Tab", Label: "Mode"},
			}, selectKeys...)
		}
	case v.view.Searching() != nil:
		status = searchStatus(v.view.Searching())
		keys = searchKeys
	default:
		keys = normalKeys
	}
	keys = withMode(keys, v.mode)
	if msg != "" {
		if status != "" {
			status = msg + "   " + status
		} else {
			status = msg
		}
	}
	v.view.SetFooter(status, keys)
}

// withMode returns keys with the Tab entry labelled with the current mode,
// e.g. "Tab Mode: Base".
func withMode(keys []widgets.KeyHelp, mode search.Mode) []widgets.KeyHelp {
	out := make([]widgets.KeyHelp, len(keys))
	copy(out, keys)
	for i := range out {
		if out[i].Key == "Tab" {
			out[i].Label = "Mode: " + mode.String()
		}
	}
	return out
}

// modeChoice lists the search modes with the current one in brackets, so
// that the prompt shows what Tab switches to: (Tab) [Base] All words Regex
func modeChoice(current search.Mode) string {
	s := "(Tab)"
	for m := search.Base; ; m = m.Next() {
		if m == current {
			s += " [" + m.String() + "]"
		} else {
			s += " " + m.String()
		}
		if m.Next() == search.Base {
			return s
		}
	}
}

// searchStatus describes the search shown, e.g. Search [Regex]: "err.*"
func searchStatus(m *search.Matcher) string {
	return "Search [" + m.Mode.String() + "]: \"" + m.Query + "\""
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
	m := v.view.Searching()
	if m == nil {
		return false
	}
	// the lines around the next match replace the loaded ones: only a page
	// of the log is read into memory, whatever its size
	var loaded bool
	if older {
		loaded = v.history.FindOlder(m.Match)
	} else {
		loaded = v.history.FindNewer(m.Match)
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
