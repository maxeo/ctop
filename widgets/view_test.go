package widgets

import (
	"fmt"
	"testing"

	"github.com/maxeo/ctop/search"
)

func base(q string) *search.Matcher {
	m, _ := search.New(search.Base, q)
	return m
}

func TestSplitEmptyLine(t *testing.T) {

	result := splitLine("", 5)
	if len(result) != 0 {
		t.Errorf("expected: 0 lines, got: %d", len(result))
	}
}

func TestSplitLineShorterThanLimit(t *testing.T) {

	result := splitLine("hello", 7)
	if len(result) != 1 {
		t.Errorf("expected: 0 lines, got: %d", len(result))
	}
}

func TestSplitLineLongerThanLimit(t *testing.T) {

	result := splitLine("hello", 3)
	if len(result) != 2 {
		t.Errorf("expected: 0 lines, got: %d", len(result))
	}
}

func TestSplitLineSameAsLimit(t *testing.T) {

	result := splitLine("hello", 5)
	if len(result) != 1 {
		t.Errorf("expected: 0 lines, got: %d", len(result))
	}
}

type testLine string

func (l testLine) Toggle(bool) string { return string(l) }

func newTestView(lines ...string) *TextView {
	t := &TextView{Text: []ToggleText{}, render: make(chan struct{}, 1), search: viewSearch{match: -1}}
	t.Width, t.Height = 40, 3+titleRows+footerRows // 3 text lines
	for _, l := range lines {
		t.Text = append(t.Text, testLine(l))
	}
	t.layout()
	return t
}

func TestFindNextOlderAndNewer(t *testing.T) {
	v := newTestView("Error one", "ok", "error two", "ok", "ok", "ok", "ok")
	v.SetSearch(base("ERROR"))
	if !v.FindNext(true) || v.search.match != 2 {
		t.Fatalf("expected match 2, got %d", v.search.match)
	}
	if !v.FindNext(true) || v.search.match != 0 {
		t.Fatalf("expected match 0, got %d", v.search.match)
	}
	if v.FindNext(true) || v.search.match != 0 {
		t.Fatalf("expected no older match, still at 0")
	}
	if !v.FindNext(false) || v.search.match != 2 {
		t.Fatalf("expected newer match 2, got %d", v.search.match)
	}
}

func TestPrependShiftsMatch(t *testing.T) {
	v := newTestView("a", "match", "b")
	v.SetSearch(base("match"))
	v.FindNext(true)
	v.Prepend([]ToggleText{testLine("x"), testLine("y")})
	if v.search.match != 3 {
		t.Fatalf("expected match 3, got %d", v.search.match)
	}
}

func TestMarks(t *testing.T) {
	s := viewSearch{m: base("ab"), match: 1}
	got := s.marks("xAbyab", 0)
	want := []int{0, markMatch, markMatch, 0, markMatch, markMatch}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("marks = %v, want %v", got, want)
		}
	}
	if m := s.marks("ab", 1); m[0] != markCurrent {
		t.Fatalf("current match line not marked: %v", m)
	}
}

func TestAttachKeepsNewerPending(t *testing.T) {
	v := newTestView("head1", "head2")
	v.Detach()
	v.pending = []ToggleText{testLine("old"), testLine("new")}
	v.Attach([]ToggleText{testLine("head3")}, func(l ToggleText) bool { return l.Toggle(false) == "new" })
	if v.Detached() || len(v.Text) != 4 || v.Text[3].Toggle(false) != "new" || v.pending != nil {
		t.Fatalf("unexpected state: detached=%v text=%v", v.Detached(), v.Text)
	}
}

func TestSearchTyping(t *testing.T) {
	v := newTestView("alpha", "beta", "gamma", "x", "y", "z")
	v.BeginTyping() // bottom line: "z" (5)
	if !v.SearchTyping(base("a")) || v.search.match != 2 {
		t.Fatalf("expected closest older match 2, got %d", v.search.match)
	}
	if !v.SearchTyping(base("alp")) || v.search.match != 0 {
		t.Fatalf("expected match 0, got %d", v.search.match)
	}
	if v.SearchTyping(base("nothing")) || v.search.match != -1 || v.scroll != 0 {
		t.Fatalf("expected no match and the starting position")
	}
	v.SearchTyping(base("beta"))
	v.CancelTyping()
	if v.search.m != nil || v.scroll != 0 {
		t.Fatalf("cancel did not reset the search")
	}
}

func TestScrollPastEnd(t *testing.T) {
	var lines []string
	for i := 0; i < 50; i++ {
		lines = append(lines, "line")
	}
	v := newTestView(lines...)
	v.Height = 40 + titleRows + footerRows // 40 text rows
	v.ScrollDown(100)
	v.layout()
	if len(v.TextOut) != minVisibleRows || v.scroll != minVisibleRows-40 {
		t.Fatalf("expected %d visible rows, got %d (scroll %d)", minVisibleRows, len(v.TextOut), v.scroll)
	}
	// new lines fill the blank rows, the old ones stay where they are
	v.Append([]ToggleText{testLine("a"), testLine("b")})
	if v.scroll != minVisibleRows-40+2 {
		t.Fatalf("expected scroll %d, got %d", minVisibleRows-40+2, v.scroll)
	}
	// short log: all its lines stay visible
	v = newTestView("a", "b", "c")
	v.Height = 40 + titleRows + footerRows
	v.ScrollDown(100)
	v.layout()
	if len(v.TextOut) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(v.TextOut))
	}
}

func TestScrollToBottomLeavesRoom(t *testing.T) {
	var lines []string
	for i := 0; i < 50; i++ {
		lines = append(lines, "line")
	}
	v := newTestView(lines...)
	v.Height = 40 + titleRows + footerRows
	v.ScrollUp(10)
	v.ScrollToBottom()
	v.layout()
	if len(v.TextOut) != minVisibleRows || v.outSrc[len(v.outSrc)-1] != 49 {
		t.Fatalf("expected the last %d lines on top, got %d rows", minVisibleRows, len(v.TextOut))
	}
}

func TestTimestampRunes(t *testing.T) {
	cases := map[string]int{
		"2026-11-17 11:39:46.814 WARNING x":                  23,
		"2026-11-17T11:39:46.814Z 2026-11-17 11:39:46.814 W": 48,
		"[2026-11-17 11:39:46] info":                         21,
		"2026-11-17T11:39:46+01:00 msg":                      25,
		"s6-rc: info: service starting":                      0,
		"2026-11-17T11:39:46.814Z plain message":             24,
	}
	for line, want := range cases {
		if got := timestampRunes(line); got != want {
			t.Errorf("timestampRunes(%q) = %d, want %d", line, got, want)
		}
	}
}

func TestSplitLineRunes(t *testing.T) {
	cases := []struct {
		line string
		size int
		want []string
	}{
		{"perché così", 6, []string{"perché", " così"}},
		{"àèìòù", 2, []string{"àè", "ìò", "ù"}},
		{"日本語", 4, []string{"日本", "語"}},     // wide runes: 2 cells each
		{"a日本", 2, []string{"a", "日", "本"}}, // a wide rune never straddles two rows
		{"日", 1, []string{"日"}},             // too wide for a row: alone on it
	}
	for _, c := range cases {
		got := splitLine(c.line, c.size)
		if len(got) != len(c.want) {
			t.Fatalf("splitLine(%q, %d) = %q, want %q", c.line, c.size, got, c.want)
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Fatalf("splitLine(%q, %d) = %q, want %q", c.line, c.size, got, c.want)
			}
		}
	}
}

func manyLines(n int, prefix string) []ToggleText {
	lines := make([]ToggleText, n)
	for i := range lines {
		lines[i] = testLine(fmt.Sprintf("%s%d", prefix, i))
	}
	return lines
}

func TestAppendDropsOldest(t *testing.T) {
	v := newTestView()
	v.Append(manyLines(maxLines+trimSlack+1, "l"))
	if len(v.Text) != maxLines {
		t.Fatalf("%d lines kept, want %d", len(v.Text), maxLines)
	}
	if got := v.Text[len(v.Text)-1].Toggle(false); got != fmt.Sprintf("l%d", maxLines+trimSlack) {
		t.Fatalf("newest line %q dropped", got)
	}
}

func TestPrependDropsNewestAndDetaches(t *testing.T) {
	v := newTestView("tail")
	v.SetSearch(base("tail"))
	v.FindNext(true)
	v.Prepend(manyLines(maxLines+trimSlack, "old"))
	if len(v.Text) != maxLines || !v.Detached() {
		t.Fatalf("%d lines, detached=%v; want %d lines, detached", len(v.Text), v.Detached(), maxLines)
	}
	if v.search.match != -1 {
		t.Fatalf("match %d on a dropped line", v.search.match)
	}
	if line, run, loaded := v.Edge(false, func(a, b ToggleText) bool { return a == b }); line.Toggle(false) != "old0" || run != 1 || loaded != maxLines {
		t.Fatalf("Edge = %v %d %d", line, run, loaded)
	}
}

func TestEdgeRun(t *testing.T) {
	v := newTestView("a1", "a2", "b", "c1", "c2", "c3")
	sameFirst := func(a, b ToggleText) bool { return a.Toggle(false)[0] == b.Toggle(false)[0] }
	if _, run, _ := v.Edge(false, sameFirst); run != 2 {
		t.Fatalf("oldest run %d, want 2", run)
	}
	if line, run, _ := v.Edge(true, sameFirst); run != 3 || line.Toggle(false) != "c3" {
		t.Fatalf("newest %v run %d, want c3 3", line, run)
	}
}

func TestFindNewerAfterReplace(t *testing.T) {
	v := newTestView("x")
	v.SetSearch(base("hit"))
	v.Replace([]ToggleText{testLine("a"), testLine("hit 1"), testLine("b"), testLine("c"), testLine("d"), testLine("hit 2")})
	if !v.FindNext(false) || v.search.match != 1 {
		t.Fatalf("newer search after Replace: match %d, want 1", v.search.match)
	}
}

func TestKeyRows(t *testing.T) {
	v := newTestView()
	v.Width = 40
	v.keys = []KeyHelp{{Key: "a", Label: "one"}, {Key: "b", Label: "two"}}
	if r := v.keyRowsLocked(); r != KeyHelpRows {
		t.Fatalf("%d rows for two short keys, want %d", r, KeyHelpRows)
	}
	for i := 0; i < 10; i++ {
		v.keys = append(v.keys, KeyHelp{Key: "Key", Label: "a long label"})
	}
	if r := v.keyRowsLocked(); r != KeyHelpRows+1 {
		t.Fatalf("%d rows for keys that do not fit in two, want %d", r, KeyHelpRows+1)
	}
}

func TestMarksAllWordsAccents(t *testing.T) {
	m, _ := search.New(search.AllWords, "è perch")
	s := viewSearch{m: m, match: -1}
	got := s.marks("è perché", 0)
	// runes: è, space, p e r c h é: è and perch (é is not è)
	want := []int{markMatch, 0, markMatch, markMatch, markMatch, markMatch, markMatch, 0}
	for i := range want {
		if (got[i] != 0) != (want[i] != 0) {
			t.Fatalf("marks = %v, want matches at %v", got, want)
		}
	}
}
