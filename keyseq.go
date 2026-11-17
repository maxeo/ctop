package main

import "strings"

// csiKeys recognizes Home/End sequences that termbox does not know.
//
// termbox only decodes the sequences in its terminfo table (for xterm: Home
// \eOH, End \eOF). Terminals such as PuTTY or MobaXterm send \e[1~ / \e[4~,
// rxvt \e[7~ / \e[8~, others \e[H / \e[F, and \eOH / \eOF when TERM is
// not xterm (e.g. tmux). In Alt input mode termbox turns those into <M-[> or
// <M-O> followed by plain keys ("1", "~"), which the log view used to read
// as "any key: close". Feed the keys after <M-[> / <M-O> here instead.
type csiKeys struct {
	active bool
	buf    string
}

// Start begins a sequence (call on <M-[> or <M-O>).
func (k *csiKeys) Start() {
	k.active = true
	k.buf = ""
}

// Feed takes the next key name. consumed reports whether the key belonged to
// a sequence (and must not be handled as a normal key); action is "home" or
// "end" when a sequence is complete, "" otherwise.
func (k *csiKeys) Feed(key string) (action string, consumed bool) {
	if !k.active {
		return "", false
	}
	if len(key) != 1 {
		// not part of an escape sequence, e.g. <escape> or <up>
		k.active = false
		return "", false
	}
	k.buf += key
	switch k.buf {
	case "H", "1~", "7~":
		k.active = false
		return "home", true
	case "F", "4~", "8~":
		k.active = false
		return "end", true
	}
	if len(k.buf) < 3 && strings.ContainsAny(key, "0123456789;") {
		return "", true // wait for the rest
	}
	// unknown sequence: swallow it rather than closing the view
	k.active = false
	return "", true
}
