package widgets

import "strings"

// how a character of a visible line is highlighted
const (
	markNone = iota
	markMatch
	markCurrent
)

// viewSearch is the text searched in a TextView and the line found last.
type viewSearch struct {
	query  string // lower case; "" = no search
	match  int    // index in Text of the current match, -1 if none
	anchor int    // while typing: index in Text of the bottom line when typing started
	scroll int    // while typing: scroll offset when typing started
}

// marks returns the highlight of each rune of line (a visible line of Text[src]).
// Matches split by line wrapping are not highlighted.
func (s *viewSearch) marks(line string, src int) []int {
	runes := []rune(line)
	marks := make([]int, len(runes))
	if s.query == "" {
		return marks
	}
	mark := markMatch
	if src == s.match {
		mark = markCurrent
	}
	lower := []rune(strings.ToLower(line))
	query := []rune(s.query)
	if len(lower) != len(runes) {
		return marks // case folding changed the length: skip highlighting
	}
	for i := 0; i+len(query) <= len(lower); {
		if string(lower[i:i+len(query)]) == s.query {
			for j := i; j < i+len(query); j++ {
				marks[j] = mark
			}
			i += len(query)
			continue
		}
		i++
	}
	return marks
}

// SetSearch highlights query (case insensitive) and forgets the current
// match. An empty query ends the search.
func (t *TextView) SetSearch(query string) {
	t.mu.Lock()
	t.search.query = strings.ToLower(query)
	t.search.match = -1
	t.mu.Unlock()
	t.requestRender()
}

// BeginTyping remembers the position on screen: SearchTyping looks for the
// query from there, CancelTyping goes back to it.
func (t *TextView) BeginTyping() {
	t.mu.Lock()
	t.search.anchor = len(t.Text) - 1
	if len(t.outSrc) > 0 {
		t.search.anchor = t.outSrc[len(t.outSrc)-1]
	}
	t.search.scroll = t.scroll
	t.mu.Unlock()
}

// SearchTyping highlights query and shows the closest loaded line that
// contains it: the first one above the starting position, else the first one
// below. Without a match the view goes back to the starting position. It
// reports whether a line matches.
func (t *TextView) SearchTyping(query string) bool {
	t.mu.Lock()
	defer t.requestRender()
	defer t.mu.Unlock()

	t.search.query = strings.ToLower(query)
	t.search.match = -1
	if t.search.query != "" {
		if t.findFromLocked(t.search.anchor, true) || t.findFromLocked(t.search.anchor+1, false) {
			return true
		}
	}
	t.scroll = t.search.scroll
	return t.search.query == ""
}

// CancelTyping ends the search and goes back to the starting position.
func (t *TextView) CancelTyping() {
	t.mu.Lock()
	t.search.query = ""
	t.search.match = -1
	t.scroll = t.search.scroll
	t.mu.Unlock()
	t.requestRender()
}

// SearchQuery returns the text being searched ("" if none).
func (t *TextView) SearchQuery() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.search.query
}

// FindNext moves to the next loaded line containing the search text: older
// (towards the start) or newer. The first search starts from the lines on
// screen. It reports false, without moving, if no other loaded line matches.
func (t *TextView) FindNext(older bool) bool {
	t.mu.Lock()
	found := t.findLocked(older)
	t.mu.Unlock()
	if found {
		t.requestRender()
	}
	return found
}

func (t *TextView) findLocked(older bool) bool {
	if t.search.query == "" || len(t.Text) == 0 {
		return false
	}
	step := 1
	if older {
		step = -1
	}

	var i int
	switch {
	case t.search.match >= 0:
		i = t.search.match + step
	case len(t.outSrc) == 0:
		i = len(t.Text) - 1
	case older:
		i = t.outSrc[len(t.outSrc)-1] // from the bottom of the screen
	default:
		i = t.outSrc[0] // from the top of the screen
	}

	return t.findFromLocked(i, older)
}

// findFromLocked looks for the query from Text[i] (included), towards older
// or newer lines.
func (t *TextView) findFromLocked(i int, older bool) bool {
	step := 1
	if older {
		step = -1
	}
	for ; i >= 0 && i < len(t.Text); i += step {
		if strings.Contains(strings.ToLower(t.Text[i].Toggle(t.toggleState)), t.search.query) {
			t.search.match = i
			t.scrollToLocked(i)
			return true
		}
	}
	return false
}

// scrollToLocked puts Text[i] in the middle of the screen.
func (t *TextView) scrollToLocked(i int) {
	below := 0
	for _, l := range t.Text[i+1:] {
		below += len(splitLine(l.Toggle(t.toggleState), t.textWidth()))
	}
	scroll := below - t.textHeight()/2
	if scroll < 0 {
		scroll = 0
	}
	t.scroll = scroll
}
