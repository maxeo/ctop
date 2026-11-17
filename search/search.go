// Package search matches the text searched in the log view: plain text,
// whole words or a regular expression, always ignoring case.
package search

import (
	"errors"
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Mode is how the query is matched.
type Mode int

const (
	Base      Mode = iota // the text anywhere
	WholeWord             // the text as a whole word
	Regex                 // a regular expression (Go syntax)
)

func (m Mode) String() string {
	switch m {
	case WholeWord:
		return "Whole word"
	case Regex:
		return "Regex"
	}
	return "Base"
}

// Next returns the mode after m, cycling.
func (m Mode) Next() Mode {
	return (m + 1) % 3
}

// Matcher finds a query in lines of text.
//
// In the Base and Whole word modes the query can hold several words
// separated by spaces: a line matches when it contains every word, in any
// order, and each word is highlighted. A regular expression is one pattern,
// spaces included.
type Matcher struct {
	Query string
	Mode  Mode
	res   []*regexp.Regexp // one per word (one for a regex)
}

// New compiles query for mode. It returns nil for an empty query (or only
// spaces), and an error for an invalid regular expression.
func New(mode Mode, query string) (*Matcher, error) {
	var patterns []string
	if mode == Regex {
		if query != "" {
			patterns = []string{query}
		}
	} else {
		for _, w := range strings.Fields(query) {
			patterns = append(patterns, regexp.QuoteMeta(w))
		}
	}
	if len(patterns) == 0 {
		return nil, nil
	}
	m := &Matcher{Query: query, Mode: mode}
	for _, p := range patterns {
		re, err := regexp.Compile("(?i)" + p)
		if err != nil {
			// report the pattern as typed, without the (?i) added above
			return nil, errors.New(strings.Replace(err.Error(), "(?i)", "", 1))
		}
		m.res = append(m.res, re)
	}
	return m, nil
}

// Match reports whether s contains every word (one that Find returns).
func (m *Matcher) Match(s string) bool {
	for _, re := range m.res {
		if len(m.find(s, re, 1)) == 0 {
			return false
		}
	}
	return true
}

// Find returns the byte ranges [start, end) of the matches of every word in
// s, in order and not overlapping, or nil if a word is missing. Empty matches
// of a regular expression are left out.
func (m *Matcher) Find(s string) [][2]int {
	var all [][2]int
	for _, re := range m.res {
		found := m.find(s, re, -1)
		if len(found) == 0 {
			return nil
		}
		all = append(all, found...)
	}
	if len(m.res) == 1 {
		return all
	}
	// merge the ranges of the different words
	sort.Slice(all, func(i, j int) bool { return all[i][0] < all[j][0] })
	merged := all[:1]
	for _, r := range all[1:] {
		last := &merged[len(merged)-1]
		if r[0] <= last[1] {
			if r[1] > last[1] {
				last[1] = r[1]
			}
			continue
		}
		merged = append(merged, r)
	}
	return merged
}

func (m *Matcher) find(s string, re *regexp.Regexp, max int) [][2]int {
	var found [][2]int
	pos := 0
	for pos <= len(s) && (max < 0 || len(found) < max) {
		loc := re.FindStringIndex(s[pos:])
		if loc == nil {
			break
		}
		start, end := pos+loc[0], pos+loc[1]
		if end == start || m.Mode == WholeWord && !wholeWord(s, start, end) {
			// try again one character further
			_, size := utf8.DecodeRuneInString(s[start:])
			if size == 0 {
				break
			}
			pos = start + size
			continue
		}
		found = append(found, [2]int{start, end})
		pos = end
	}
	return found
}

// wholeWord reports whether s[start:end] is not preceded or followed by a
// letter, digit or underscore (in any script, unlike \b in Go regexps).
func wholeWord(s string, start, end int) bool {
	if r, _ := utf8.DecodeLastRuneInString(s[:start]); start > 0 && isWordRune(r) {
		return false
	}
	if r, _ := utf8.DecodeRuneInString(s[end:]); end < len(s) && isWordRune(r) {
		return false
	}
	return true
}

func isWordRune(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}
