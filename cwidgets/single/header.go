package single

import (
	"strings"
	"time"

	ui "github.com/gizak/termui"
	"github.com/hako/durafmt"
	"github.com/mattn/go-runewidth"
)

// fields shown in the header, after the image and the state
var headerFields = []string{"id", "uptime", "created", "health", "restarts", "pids", "IPs", "ports"}

// Header shows the container name, image, state and the other fields in
// columns.
type Header struct {
	box
	data      map[string]string
	cols      int
	startedAt time.Time // to show the uptime live; zero if unknown
	running   bool
}

func NewHeader() *Header {
	return &Header{box: newBox(""), data: map[string]string{}, cols: 1}
}

func (h *Header) Set(k, v string) {
	h.data[k] = v
	if k == "name" {
		h.BorderLabel = " " + v + " "
	}
}

// setWidth chooses the number of columns for the width.
func (h *Header) setWidth(w int) {
	switch {
	case w >= 135:
		h.cols = 3
	case w >= 64:
		h.cols = 2
	default:
		h.cols = 1
	}
}

// rows returns the height of the header for its columns.
func (h *Header) rows() int {
	return 2 + 1 + (len(headerFields)+h.cols-1)/h.cols
}

// value returns the text shown for field k.
func (h *Header) value(k string) string {
	if k == "uptime" && h.running && !h.startedAt.IsZero() {
		// the uptime from the connector is from when it last looked
		return durafmt.Parse(time.Since(h.startedAt)).LimitFirstN(1).String()
	}
	v := strings.TrimSpace(h.data[k])
	if v == "" {
		switch k {
		case "health":
			return "N/A" // no healthcheck
		case "ports":
			return "none"
		}
		return "-"
	}
	if k == "id" && len(v) > 12 {
		v = v[:12]
	}
	// several lines (one per network or port): one line, comma separated
	var parts []string
	for _, p := range strings.Split(v, "\n") {
		if p = strings.TrimSpace(p); p != "" {
			parts = append(parts, p)
		}
	}
	return strings.Join(parts, ", ")
}

// stateColor returns the color of a container or health state.
func stateColor(s string) ui.Attribute {
	switch s {
	case "running", "healthy":
		return ui.ColorGreen
	case "paused", "starting", "created", "restarting":
		return ui.ColorYellow
	case "exited", "dead", "unhealthy", "destroyed":
		return ui.ColorRed
	}
	return textFg()
}

func (h *Header) Buffer() ui.Buffer {
	buf := h.Block.Buffer()
	in := h.inner()
	x0, end := in.Min.X+1, in.Max.X-1

	// first row: the image, and the state on the right
	state := h.data["state"]
	if state == "" {
		state = "unknown"
	}
	badge := "● " + state
	drawRight(buf, in.Min.Y, x0, end, badge, stateColor(state)|ui.AttrBold, ui.ColorDefault)
	drawText(buf, x0, in.Min.Y, end-runewidth.StringWidth(badge)-2, h.data["image"], textFg()|ui.AttrBold, ui.ColorDefault)

	// the other fields in columns, filled left to right
	keyWidth := 0
	for _, k := range headerFields {
		if w := runewidth.StringWidth(k); w > keyWidth {
			keyWidth = w
		}
	}
	colWidth := (end - x0) / h.cols
	for i, k := range headerFields {
		x := x0 + (i%h.cols)*colWidth
		y := in.Min.Y + 1 + i/h.cols
		if y >= in.Max.Y {
			break
		}
		colEnd := x + colWidth - 2
		if i%h.cols == h.cols-1 {
			colEnd = end
		}
		drawText(buf, x, y, colEnd, k, dimFg(), ui.ColorDefault)
		v := h.value(k)
		fg := textFg()
		if k == "health" {
			fg = stateColor(v)
		}
		drawText(buf, x+keyWidth+2, y, colEnd, v, fg, ui.ColorDefault)
	}
	return buf
}
