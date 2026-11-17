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
		{WholeWord, "log", "LOG: login catalog log", "[LOG][log]"},
		{WholeWord, "perch", "perché perch", "[perch]"}, // é is a letter
		{WholeWord, "è", "è così", "[è]"},
		{WholeWord, "a", "a_a a", "[a]"},
		{Regex, `err(or)?\b`, "Error: err, errors", "[Error][err]"},
		{Regex, `\d+ ms`, "took 12 ms", "[12 ms]"},
		{Regex, `x*`, "abc", ""}, // empty matches only
		// several words: all of them, in any order, each highlighted
		{Base, "warn power main", "2026-11-17 WARNING (MainThread) [x] Could not reset the power state", "[WARN][Main][power]"},
		{Base, "blue auto", "[bluetooth_auto_recovery.recover] Bluetooth", "[blue][auto][Blue]"},
		{Base, "blue missing", "bluetooth", ""},
		{Base, "  blue   ", "blue", "[blue]"},
		{WholeWord, "could power", "Could not reset the power state", "[Could][power]"},
		{WholeWord, "blue auto", "[bluetooth_auto_recovery.recover]", ""},
		{Regex, "a b", "a b ab", "[a b]"}, // a regex is one pattern
		{Base, "ab abc", "abcd", "[abc]"}, // overlapping words are merged
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
	if m, err := New(WholeWord, "   "); m != nil || err != nil {
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
	if Base.Next() != WholeWord || WholeWord.Next() != Regex || Regex.Next() != Base {
		t.Fatal("modes do not cycle Base, Whole word, Regex")
	}
}
