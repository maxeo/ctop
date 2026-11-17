package collector

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/eqms/ctop/models"
)

var t0 = time.Date(2026, 11, 17, 10, 0, 0, 0, time.UTC)

// testLog builds a log of n lines "line 0" ... "line n-1"; times[i] is the
// second of line i (one second per line if times is nil).
func testLog(n int, times []int) []models.Log {
	logs := make([]models.Log, n)
	for i := range logs {
		sec := i
		if times != nil {
			sec = times[i]
		}
		logs[i] = models.Log{Timestamp: t0.Add(time.Duration(sec) * time.Second), Message: fmt.Sprintf("line %d", i)}
	}
	return logs
}

// feed passes the lines to add, oldest first, until it returns false.
func feed(logs []models.Log, add func(models.Log) bool) {
	for _, l := range logs {
		if !add(l) {
			return
		}
	}
}

func messages(logs []models.Log) string {
	s := ""
	for i, l := range logs {
		if i > 0 {
			s += ","
		}
		s += l.Message[len("line "):]
	}
	return s
}

// contains matches query anywhere, ignoring case.
func contains(query string) func(string) bool {
	query = strings.ToLower(query)
	return func(s string) bool { return strings.Contains(strings.ToLower(s), query) }
}

func cursorAt(logs []models.Log, i, same int) LogCursor {
	return LogCursor{Time: logs[i].Timestamp, Same: same}
}

func TestPageBefore(t *testing.T) {
	logs := testLog(10, nil)
	p := pageBefore{cur: cursorAt(logs, 6, 1), n: 3}
	feed(logs, p.add)
	lines, more := p.result()
	if got := messages(lines); got != "3,4,5" || !more {
		t.Fatalf("page = %s more=%v, want 3,4,5 more=true", got, more)
	}

	p = pageBefore{cur: cursorAt(logs, 2, 1), n: 3}
	feed(logs, p.add)
	lines, more = p.result()
	if got := messages(lines); got != "0,1" || more {
		t.Fatalf("page = %s more=%v, want 0,1 more=false", got, more)
	}
}

func TestPageBeforeSameTime(t *testing.T) {
	// lines 2..5 share a second; lines 4 and 5 are loaded
	logs := testLog(8, []int{0, 1, 2, 2, 2, 2, 3, 4})
	p := pageBefore{cur: cursorAt(logs, 4, 2), n: 3}
	feed(logs, p.add)
	lines, _ := p.result()
	if got := messages(lines); got != "1,2,3" {
		t.Fatalf("page = %s, want 1,2,3 (the older lines with the same time included)", got)
	}
}

func TestPageAfter(t *testing.T) {
	logs := testLog(10, []int{0, 1, 2, 3, 3, 3, 4, 5, 6, 7})
	// lines 3 and 4 (second 3) are loaded, line 5 has the same second
	p := pageAfter{after: afterCursor{cur: cursorAt(logs, 4, 2)}, n: 3}
	feed(logs, p.add)
	if got := messages(p.lines); got != "5,6,7" || !p.more {
		t.Fatalf("page = %s more=%v, want 5,6,7 more=true", got, p.more)
	}

	p = pageAfter{n: 20}
	feed(logs, p.add)
	if len(p.lines) != 10 || p.more {
		t.Fatalf("zero cursor: %d lines more=%v, want 10 more=false", len(p.lines), p.more)
	}
}

func TestFindBefore(t *testing.T) {
	logs := testLog(30, nil)
	logs[5].Message = "line 5 ERROR"
	logs[12].Message = "line 12 error"
	logs[25].Message = "line 25 Error" // loaded: after the cursor

	f := newFindBefore(cursorAt(logs, 20, 1), contains("eRRor"), 2)
	feed(logs, f.add)
	if got := messages(f.win); !f.found || got != "10,11,12 error,13,14" {
		t.Fatalf("window = %s found=%v, want 10,11,12 error,13,14", got, f.found)
	}

	f = newFindBefore(cursorAt(logs, 5, 1), contains("error"), 2)
	feed(logs, f.add)
	if f.found {
		t.Fatalf("found %s, the match is the loaded line itself", messages(f.win))
	}
}

func TestFindBeforeSkipsLoadedSameTime(t *testing.T) {
	// lines 3 and 4 share a second; only line 4 is loaded
	logs := testLog(6, []int{0, 1, 2, 3, 3, 4})
	logs[3].Message = "line 3 match"
	logs[4].Message = "line 4 match"
	f := newFindBefore(cursorAt(logs, 4, 1), contains("match"), 1)
	feed(logs, f.add)
	if got := messages(f.win); !f.found || got != "2,3 match" {
		t.Fatalf("window = %s found=%v, want 2,3 match", got, f.found)
	}
}

func TestFindAfter(t *testing.T) {
	logs := testLog(30, nil)
	logs[3].Message = "line 3 x"
	logs[15].Message = "line 15 x"
	logs[18].Message = "line 18 x"

	f := newFindAfter(cursorAt(logs, 10, 1), contains("X"), 2)
	feed(logs, f.add)
	if got := messages(f.win); !f.found || got != "13,14,15 x,16,17" {
		t.Fatalf("window = %s found=%v, want 13,14,15 x,16,17", got, f.found)
	}

	f = newFindAfter(cursorAt(logs, 20, 1), contains("x"), 2)
	feed(logs, f.add)
	if f.found {
		t.Fatalf("found %s, no match after the cursor", messages(f.win))
	}
}
