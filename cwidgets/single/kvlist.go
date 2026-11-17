package single

import (
	"fmt"
	"strings"

	ui "github.com/gizak/termui"
	"github.com/mattn/go-runewidth"
	"github.com/maxeo/ctop/models"
)

// widest key column of a list
const keyMax = 32

// KVList shows sections of name / value rows in a box: it scrolls, can be
// filtered, and with Wrap set long values go on in the lines below instead
// of being cut.
type KVList struct {
	box
	Title     string // the label of the box, before the counts
	Wrap      bool
	Empty     string // shown when there are no rows
	sections  []models.DetailSection
	lines     []kvLine // the rows matching the filter, as drawn
	width     int      // width the lines were made for
	rows      int      // rows matching the filter
	total     int      // all rows
	offset    int      // first line shown
	filter    string
	filtering bool // the filter is being typed
}

// kvLine is a line of the list: a section title, a row (or the rest of a
// wrapped value, with an empty key) or a blank line between sections.
type kvLine struct {
	title string
	key   string
	value string
	cont  bool
}

func NewKVList(title string) *KVList {
	return &KVList{box: newBox(title), Title: title, Empty: "nothing to show"}
}

// SetSections sets the content; the scroll position is kept where possible.
func (l *KVList) SetSections(s []models.DetailSection) {
	l.sections = s
	l.total = 0
	for _, sec := range s {
		l.total += len(sec.Rows)
	}
	l.width = -1 // rebuild the lines
}

// SetFilter shows only the rows whose name or value contains f, ignoring
// case, and the sections whose title contains it; typing tells whether the
// filter is still being typed.
func (l *KVList) SetFilter(f string, typing bool) {
	if f != l.filter {
		l.offset = 0
		l.width = -1
	}
	l.filter, l.filtering = f, typing
}

// Len returns the number of rows matching the filter.
func (l *KVList) Len() int {
	l.build()
	return l.rows
}

// visible returns the number of lines that fit in the box.
func (l *KVList) visible() int {
	if r := l.Height - 2; r > 0 {
		return r
	}
	return 0
}

// Scroll moves the list by n lines (negative: up).
func (l *KVList) Scroll(n int) {
	l.offset += n
	l.clamp()
}

// Page moves the list by a page (dir -1: up, 1: down).
func (l *KVList) Page(dir int) {
	n := l.visible() - 1
	if n < 1 {
		n = 1
	}
	l.Scroll(dir * n)
}

// Top and Bottom go to the first and last lines.
func (l *KVList) Top()    { l.offset = 0 }
func (l *KVList) Bottom() { l.build(); l.offset = len(l.lines); l.clamp() }

func (l *KVList) clamp() {
	l.build()
	if max := len(l.lines) - l.visible(); l.offset > max {
		l.offset = max
	}
	if l.offset < 0 {
		l.offset = 0
	}
}

// keyWidth returns the width of the name column.
func (l *KVList) keyWidth(width int) int {
	w := 0
	for _, sec := range l.sections {
		for _, r := range sec.Rows {
			w = max(w, runewidth.StringWidth(r.Key))
		}
	}
	return min(w, keyMax, width/3)
}

// build makes the lines for the current width and filter.
func (l *KVList) build() {
	width := l.Width - 4 // border and a space on each side
	if width == l.width {
		return
	}
	l.width = width
	l.lines = l.lines[:0]
	l.rows = 0
	f := strings.ToLower(l.filter)
	valueWidth := width - l.keyWidth(width) - 3
	titles := 0
	for _, sec := range l.sections {
		var lines []kvLine
		// a filter matching the title shows the whole section
		all := f == "" || strings.Contains(strings.ToLower(sec.Title), f)
		for _, r := range sec.Rows {
			if !all && !strings.Contains(strings.ToLower(r.Key+"="+r.Value), f) {
				continue
			}
			l.rows++
			parts := []string{strings.ReplaceAll(r.Value, "\n", " ")}
			if l.Wrap && valueWidth > 0 {
				parts = wrapText(r.Value, valueWidth)
			}
			for i, p := range parts {
				line := kvLine{value: p}
				if i == 0 {
					line.key = r.Key
				} else {
					line.cont = true
				}
				lines = append(lines, line)
			}
		}
		if len(lines) == 0 {
			continue
		}
		if sec.Title != "" {
			if titles > 0 {
				l.lines = append(l.lines, kvLine{})
			}
			l.lines = append(l.lines, kvLine{title: sec.Title})
			titles++
		}
		l.lines = append(l.lines, lines...)
	}
	l.clamp()
}

// wrapText splits s (and its own lines) into pieces of at most width cells.
func wrapText(s string, width int) []string {
	var out []string
	for _, line := range strings.Split(s, "\n") {
		for runewidth.StringWidth(line) > width {
			cut := runewidth.Truncate(line, width, "")
			if cut == "" { // a character wider than width
				break
			}
			out = append(out, cut)
			line = line[len(cut):]
		}
		out = append(out, line)
	}
	return out
}

func (l *KVList) label() string {
	label := l.Title
	switch {
	case l.filter != "" || l.filtering:
		label += fmt.Sprintf(" (%d of %d)", l.rows, l.total)
	default:
		label += fmt.Sprintf(" (%d)", l.total)
	}
	if rows := l.visible(); len(l.lines) > rows && rows > 0 {
		label += fmt.Sprintf(" · lines %d-%d of %d", l.offset+1, min(l.offset+rows, len(l.lines)), len(l.lines))
	}
	if l.filtering {
		label += " · filter: " + l.filter + "█"
	} else if l.filter != "" {
		label += " · filter: " + l.filter
	}
	return label
}

func (l *KVList) Buffer() ui.Buffer {
	l.build()
	l.clamp()
	l.BorderLabel = l.label()
	buf := l.Block.Buffer()
	in := l.inner()
	x0, end := in.Min.X+1, in.Max.X-1
	if len(l.lines) == 0 {
		msg := l.Empty
		if l.filter != "" {
			msg = "nothing matches the filter"
		}
		if in.Dy() > 0 {
			drawText(buf, x0, in.Min.Y, end, msg, dimFg(), ui.ColorDefault)
		}
		return buf
	}
	keyWidth := l.keyWidth(end - x0)
	for i, line := range l.lines[l.offset:] {
		y := in.Min.Y + i
		if y >= in.Max.Y {
			break
		}
		switch {
		case line.title != "":
			drawText(buf, x0, y, end, line.title, ui.ColorYellow|ui.AttrBold, ui.ColorDefault)
		case line.cont:
			drawText(buf, x0+keyWidth+3, y, end, line.value, textFg(), ui.ColorDefault)
		case line.key != "":
			drawText(buf, x0, y, x0+keyWidth, line.key, ui.ColorCyan, ui.ColorDefault)
			drawText(buf, x0+keyWidth+1, y, x0+keyWidth+3, "=", dimFg(), ui.ColorDefault)
			drawText(buf, x0+keyWidth+3, y, end, line.value, textFg(), ui.ColorDefault)
		}
	}
	// the list goes on: arrows on the right border
	if l.offset > 0 {
		buf.Set(in.Max.X, in.Min.Y, ui.Cell{Ch: '▲', Fg: dimFg(), Bg: ui.ColorDefault})
	}
	if l.offset+l.visible() < len(l.lines) {
		buf.Set(in.Max.X, in.Max.Y-1, ui.Cell{Ch: '▼', Fg: dimFg(), Bg: ui.ColorDefault})
	}
	return buf
}
