package widgets

import (
	"regexp"
	"unicode/utf8"

	ui "github.com/gizak/termui"
)

// a timestamp at the start of a log line: ISO 8601 with a space or "T",
// optional fraction and time zone, optionally in square brackets, e.g.
// "2026-11-17 11:39:46.814", "2026-11-17T11:39:46.814Z", "[2026-11-17 11:39:46]"
var lineTimestamp = regexp.MustCompile(`^\[?\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}(?:[.,]\d+)?(?:Z|[+-]\d{2}:?\d{2})?\]?`)

// color of the timestamps at the start of log lines
const timestampColor = ui.ColorCyan

// timestampRunes returns how many runes at the start of line belong to
// timestamps: one, or two when ctop's own timestamp (key t) precedes the one
// written by the application.
func timestampRunes(line string) int {
	n := 0
	rest := line
	for i := 0; i < 2; i++ {
		loc := lineTimestamp.FindStringIndex(rest)
		if loc == nil {
			break
		}
		n += utf8.RuneCountInString(rest[:loc[1]])
		rest = rest[loc[1]:]
		if len(rest) == 0 || rest[0] != ' ' {
			break
		}
		// include the space only if another timestamp follows
		if !lineTimestamp.MatchString(rest[1:]) {
			break
		}
		n++
		rest = rest[1:]
	}
	return n
}
