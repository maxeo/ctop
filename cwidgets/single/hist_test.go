package single

import (
	"reflect"
	"testing"
)

func TestIntHistLast(t *testing.T) {
	h := NewIntHist(4)
	h.Append(1)
	h.Append(2)
	if got := h.Last(6); !reflect.DeepEqual(got, []int{0, 0, 0, 0, 1, 2}) {
		t.Fatalf("padded: %v", got)
	}
	if got := h.Last(2); !reflect.DeepEqual(got, []int{1, 2}) {
		t.Fatalf("newest: %v", got)
	}
	got := h.Last(2)
	got[0] = 9
	if h.Data[2] != 1 {
		t.Fatal("Last must return a copy")
	}
	if got := h.Last(-1); len(got) != 0 {
		t.Fatalf("negative width: %v", got)
	}
	if avg, peak := h.Stats(10); avg != 2 || peak != 2 || h.Count != 2 {
		t.Fatalf("stats %d %d count %d (the padding must not count)", avg, peak, h.Count)
	}
	if avg, _ := h.Stats(1); avg != 2 {
		t.Fatalf("last one: %d", avg)
	}
	if avg, peak := NewIntHist(3).Stats(5); avg != 0 || peak != 0 {
		t.Fatal("empty stats")
	}
}

func TestDiffHist(t *testing.T) {
	h := NewDiffHist(3)
	for _, v := range []int{100, 150, 160, 10, 30} { // 10: the counter was reset
		h.Append(v)
	}
	if !reflect.DeepEqual(h.Data, []int{50, 10, 20}) {
		t.Fatalf("diffs: %v", h.Data)
	}
}

func TestMemHistory(t *testing.T) {
	m := NewMem()
	if got, _ := m.history(3); !reflect.DeepEqual(got, []int{0, 0, 0}) {
		t.Fatalf("empty: %v", got)
	}
	for _, v := range []int{1000, 1300, 1000} {
		m.Update(v, 4000)
	}
	// floor 900: the lowest is a quarter of the highest
	if got, top := m.history(4); !reflect.DeepEqual(got, []int{0, 100, 400, 100}) || top != 400 {
		t.Fatalf("floor: %v top %d", got, top)
	}
	m = NewMem()
	m.Update(500, 4000)
	m.Update(500, 4000)
	if got, top := m.history(2); !reflect.DeepEqual(got, []int{1, 1}) || top != 4 {
		t.Fatalf("flat: %v top %d", got, top)
	}
}
