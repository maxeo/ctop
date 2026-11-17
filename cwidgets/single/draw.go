package single

import (
	"image"
	"strings"

	ui "github.com/gizak/termui"
	"github.com/mattn/go-runewidth"
)

// box is a bordered panel: the border with its label, and the area inside.
type box struct {
	*ui.Block
	fg, labelFg ui.Attribute // the border colors when not selected
}

func newBox(label string) box {
	b := ui.NewBlock()
	b.BorderLabel = label
	return box{b, b.BorderFg, b.BorderLabelFg}
}

// selectedFg is the border color of the selected panel.
const selectedFg = ui.ColorCyan | ui.AttrBold

// highlight shows whether the box is selected, with the border color.
func (b box) highlight(on bool) {
	if on {
		b.BorderFg, b.BorderLabelFg = selectedFg, selectedFg
	} else {
		b.BorderFg, b.BorderLabelFg = b.fg, b.labelFg
	}
}

// place moves the box to r.
func (b box) place(r image.Rectangle) {
	b.X, b.Y = r.Min.X, r.Min.Y
	b.Width, b.Height = r.Dx(), r.Dy()
}

// inner returns the area inside the border.
func (b box) inner() image.Rectangle {
	return b.InnerBounds()
}

// textFg is the color of plain text.
func textFg() ui.Attribute { return ui.ThemeAttr("par.text.fg") }

// dimFg is the color of field names and secondary text.
func dimFg() ui.Attribute { return ui.ThemeAttr("label.fg") }

// drawText draws s from x on row y, cut with "…" so that it ends before
// end, and returns the x after the text.
func drawText(buf ui.Buffer, x, y, end int, s string, fg, bg ui.Attribute) int {
	s = strings.ReplaceAll(s, "\n", " ")
	if runewidth.StringWidth(s) > end-x {
		s = runewidth.Truncate(s, end-x, "…")
	}
	for _, ch := range s {
		w := runewidth.RuneWidth(ch)
		if x+w > end {
			break
		}
		buf.Set(x, y, ui.Cell{Ch: ch, Fg: fg, Bg: bg})
		x += w
	}
	return x
}

// drawRight draws s so that it ends just before end.
func drawRight(buf ui.Buffer, y, start, end int, s string, fg, bg ui.Attribute) {
	x := end - runewidth.StringWidth(s)
	if x < start {
		x = start
	}
	drawText(buf, x, y, end, s, fg, bg)
}

// spark levels, from 1/8 to a full cell
var sparks = []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

// drawSpark draws data as vertical bars in r, one column per value with the
// newest on the right. A full bar is top; top <= 0 means the highest value
// shown.
func drawSpark(buf ui.Buffer, r image.Rectangle, data []int, top int, color ui.Attribute) {
	if r.Dx() <= 0 || r.Dy() <= 0 {
		return
	}
	if len(data) > r.Dx() {
		data = data[len(data)-r.Dx():]
	}
	peak := top
	if peak <= 0 {
		for _, v := range data {
			if v > peak {
				peak = v
			}
		}
	}
	if peak <= 0 {
		return
	}
	levels := r.Dy() * 8
	x := r.Max.X - len(data)
	for i, v := range data {
		if v <= 0 {
			continue
		}
		if v > peak {
			v = peak
		}
		h := (v*levels + peak - 1) / peak // at least 1/8 for any value above 0
		for row := 0; h > 0; row++ {
			n := h
			if n > 8 {
				n = 8
			}
			buf.Set(x+i, r.Max.Y-1-row, ui.Cell{Ch: sparks[n-1], Fg: color, Bg: ui.ColorDefault})
			h -= n
		}
	}
}

// clear fills r with blanks.
func clear(buf ui.Buffer, r image.Rectangle) {
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			buf.Set(x, y, ui.Cell{Ch: ' ', Fg: ui.ColorDefault, Bg: ui.ColorDefault})
		}
	}
}
