package search

import (
	"fmt"
	"testing"
)

func ranges(m *Matcher, s string) string {
	out := ""
	for _, r := range m.Find(s) {
		out += fmt.Sprintf("[%s]", s[r[0]:r[1]])
	}
	return out
}

func TestModes(t *testing.T) {
	cases := []struct {
		mode  Mode
		query string
		line  string
		want  string
	}{
		{Base, "log", "LOG: login catalog", "[LOG][log][log]"},
		{Base, "a.b", "axb a.b", "[a.b]"}, // not a regexp
		{AllWords, "log", "LOG: login catalog", "[LOG][log][log]"},
		{AllWords, "é", "è perché é", "[é][é]"}, // è and é are different letters
		{Regex, `err(or)?\b`, "Error: err, errors", "[Error][err]"},
		{Regex, `\d+ ms`, "took 12 ms", "[12 ms]"},
		{Regex, `x*`, "abc", ""}, // empty matches only
		// Base: the text as typed, spaces included
		{Base, "warn power main", "WARNING (MainThread) power state", ""},
		{Base, "reset the", "Could not RESET THE power", "[RESET THE]"},
		{Base, " blue", "bluetooth a blue", "[ blue]"},
		// All words: every word, in any order, also inside longer words
		{AllWords, "warn power main", "2026-11-17 WARNING (MainThread) [x] Could not reset the power state", "[WARN][Main][power]"},
		{AllWords, "blue auto", "[bluetooth_auto_recovery.recover] Bluetooth", "[blue][auto][Blue]"},
		{AllWords, "power missing", "power", ""},
		{AllWords, "  power   ", "power", "[power]"},
		{Regex, "a b", "a b ab", "[a b]"},     // a regex is one pattern
		{AllWords, "a a", "a b", "[a]"},       // repeated words are merged
		{AllWords, "ab abc", "abcd", "[abc]"}, // overlapping words are merged
	}
	for _, c := range cases {
		m, err := New(c.mode, c.query)
		if err != nil {
			t.Fatalf("New(%v, %q): %v", c.mode, c.query, err)
		}
		if got := ranges(m, c.line); got != c.want {
			t.Errorf("%v %q in %q = %s, want %s", c.mode, c.query, c.line, got, c.want)
		}
		if want := c.want != ""; m.Match(c.line) != want {
			t.Errorf("%v %q Match(%q) = %v, want %v", c.mode, c.query, c.line, !want, want)
		}
	}
}

func TestNew(t *testing.T) {
	if m, err := New(Base, ""); m != nil || err != nil {
		t.Fatalf("empty query: %v %v", m, err)
	}
	if m, err := New(AllWords, "   "); m != nil || err != nil {
		t.Fatalf("only spaces: %v %v", m, err)
	}
	if _, err := New(Regex, "a("); err == nil {
		t.Fatal("invalid regexp accepted")
	}
	if _, err := New(Base, "a("); err != nil {
		t.Fatalf("plain text rejected: %v", err)
	}
}

func TestModeCycle(t *testing.T) {
	if Base.Next() != AllWords || AllWords.Next() != Regex || Regex.Next() != Base {
		t.Fatal("modes do not cycle Base, All words, Regex")
	}
}
