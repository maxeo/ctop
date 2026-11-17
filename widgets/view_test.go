package widgets

import "testing"

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
	v.SetSearch("ERROR")
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
	v.SetSearch("match")
	v.FindNext(true)
	v.Prepend([]ToggleText{testLine("x"), testLine("y")})
	if v.search.match != 3 {
		t.Fatalf("expected match 3, got %d", v.search.match)
	}
}

func TestMarks(t *testing.T) {
	s := viewSearch{query: "ab", match: 1}
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
	if !v.SearchTyping("a") || v.search.match != 2 {
		t.Fatalf("expected closest older match 2, got %d", v.search.match)
	}
	if !v.SearchTyping("alp") || v.search.match != 0 {
		t.Fatalf("expected match 0, got %d", v.search.match)
	}
	if v.SearchTyping("nothing") || v.search.match != -1 || v.scroll != 0 {
		t.Fatalf("expected no match and the starting position")
	}
	v.SearchTyping("beta")
	v.CancelTyping()
	if v.search.query != "" || v.scroll != 0 {
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
