package single

import (
	"image"
	"sort"
	"strings"
	"testing"

	"github.com/maxeo/ctop/models"
)

// checkLayout verifies that the panels fit in w x h above the key rows,
// do not overlap and are at least as tall as their minimum.
func checkLayout(t *testing.T, w, h, envVars int) panels {
	t.Helper()
	p := layout(w, h, 6, 2, envVars)
	rects := []image.Rectangle{p.header, p.cpu, p.mem, p.net, p.io}
	if !p.env.Empty() {
		rects = append(rects, p.env)
	}
	for i, r := range rects {
		if r.Min.X < 0 || r.Max.X > w || r.Min.Y < 0 {
			t.Fatalf("%dx%d: panel %d %v outside", w, h, i, r)
		}
		for _, o := range rects[i+1:] {
			if r.Overlaps(o) {
				t.Fatalf("%dx%d: %v overlaps %v", w, h, r, o)
			}
		}
	}
	if p.cpu.Dy() < 5 || p.mem.Dy() < 4 || p.net.Dy() < 6 || p.io.Dy() < 6 {
		t.Fatalf("%dx%d: chart below its minimum: %+v", w, h, p)
	}
	return p
}

func TestLayout(t *testing.T) {
	for _, w := range []int{40, 80, 109, 110, 160, 300} {
		for _, h := range []int{16, 24, 40, 60, 100} {
			for _, env := range []int{0, 3, 200} {
				p := checkLayout(t, w, h, env)
				if !p.env.Empty() && p.env.Max.Y > h-2 {
					t.Fatalf("%dx%d: env %v over the key help", w, h, p.env)
				}
			}
		}
	}

	// wide: CPU | MEM side by side
	if p := checkLayout(t, 160, 45, 40); p.cpu.Min.Y != p.mem.Min.Y || p.cpu.Max.X != p.mem.Min.X {
		t.Fatalf("not side by side: %+v", p)
	}
	// narrow: one column
	if p := checkLayout(t, 80, 45, 40); p.mem.Min.Y != p.cpu.Max.Y || p.mem.Dx() != 80 {
		t.Fatalf("not stacked: %+v", p)
	}
	// a long environment does not shrink the charts below their preferred
	// height, and takes the rest
	p := checkLayout(t, 160, 45, 200)
	if p.cpu.Dy() < 10 || p.env.Max.Y != 43 {
		t.Fatalf("long env: %+v", p)
	}
	// a short environment leaves the rows to the charts
	if p := checkLayout(t, 160, 60, 2); p.cpu.Dy() != 14 || p.env.Dy() < 4 {
		t.Fatalf("short env: %+v", p)
	}
}

func TestNaturalLess(t *testing.T) {
	keys := []string{"VAR_10", "path", "VAR_2", "VAR_1", "LANG", "var_02x", "VAR_02"}
	sort.SliceStable(keys, func(i, j int) bool { return naturalLess(keys[i], keys[j]) })
	want := []string{"LANG", "path", "VAR_1", "VAR_2", "VAR_02", "var_02x", "VAR_10"}
	for i := range want {
		if keys[i] != want[i] {
			t.Fatalf("got %v, want %v", keys, want)
		}
	}
}

func TestEnvScrollAndFilter(t *testing.T) {
	e := NewEnv()
	e.Height, e.Width = 7, 40 // 5 lines
	e.Set("A=1;B=2;C=3;D=4;E=5;F=6;G=7;H=8;PASSWORD=x")
	if e.Len() != 9 {
		t.Fatalf("len %d", e.Len())
	}
	e.Bottom()
	if e.offset != 4 {
		t.Fatalf("bottom offset %d", e.offset)
	}
	e.Scroll(10)
	if e.offset != 4 {
		t.Fatalf("scroll past the end: %d", e.offset)
	}
	e.Page(-1)
	if e.offset != 0 {
		t.Fatalf("page up: %d", e.offset)
	}
	e.SetFilter("pass", false)
	if e.Len() != 1 || e.lines[0].value == "x" {
		t.Fatalf("filter or redaction: %+v", e.lines)
	}
	e.SetFilter("", false)
	if e.Len() != 9 {
		t.Fatalf("no filter: %d", e.Len())
	}
}

func TestKVListWrap(t *testing.T) {
	l := NewKVList("x")
	l.Wrap = true
	l.Width, l.Height = 24, 10 // 20 cells: key 3 + " = " + 14
	l.SetSections([]models.DetailSection{
		{Title: "One", Rows: []models.DetailRow{{Key: "abc", Value: "0123456789abcdefghij"}}},
		{Title: "Two", Rows: []models.DetailRow{{Key: "k", Value: "v"}}},
	})
	l.build()
	var got []string
	for _, line := range l.lines {
		got = append(got, line.title+"|"+line.key+"|"+line.value)
	}
	want := []string{"One||", "|abc|0123456789abcd", "||efghij", "||", "Two||", "|k|v"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("lines %q, want %q", got, want)
	}
	// the filter keeps the sections with matching rows only
	l.SetFilter("efg", false)
	if l.Len() != 1 || l.lines[0].title != "One" || len(l.lines) != 3 {
		t.Fatalf("filtered: %+v", l.lines)
	}
	l.SetFilter("tw", false) // a section title
	if l.Len() != 1 || l.lines[0].title != "Two" {
		t.Fatalf("title filter: %+v", l.lines)
	}
}

func TestNearest(t *testing.T) {
	p := layout(160, 45, 6, 2, 40)
	r := [panelCount]image.Rectangle{p.header, p.cpu, p.mem, p.net, p.io, p.env}
	cases := []struct{ from, dx, dy, want int }{
		{panelContainer, 0, 1, panelCPU},
		{panelContainer, 1, 0, -1},
		{panelCPU, 1, 0, panelMem},
		{panelMem, -1, 0, panelCPU},
		{panelMem, 0, 1, panelIO},
		{panelMem, 0, -1, panelContainer},
		{panelIO, 0, 1, panelEnv},
		{panelEnv, 0, -1, panelNet},
		{panelEnv, 0, 1, -1},
	}
	for _, c := range cases {
		if got := nearest(r, c.from, c.dx, c.dy); got != c.want {
			t.Errorf("from %s (%d,%d): %d, want %d", panelNames[c.from], c.dx, c.dy, got, c.want)
		}
	}
	// one column: left and right go nowhere
	p = layout(80, 50, 6, 2, 40)
	r = [panelCount]image.Rectangle{p.header, p.cpu, p.mem, p.net, p.io, p.env}
	if got := nearest(r, panelMem, 0, 1); got != panelNet {
		t.Errorf("stacked down: %d", got)
	}
	if got := nearest(r, panelMem, 1, 0); got != -1 {
		t.Errorf("stacked right: %d", got)
	}
}
