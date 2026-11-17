package single

import (
	"fmt"
	"image"

	ui "github.com/gizak/termui"
	"github.com/maxeo/ctop/cwidgets"
)

// Mem shows the memory used against the limit, as a bar with the
// percentage, and the usage history below when there is room.
type Mem struct {
	box
	hist  *IntHist
	limit int
}

func NewMem() *Mem {
	return &Mem{box: newBox("MEM"), hist: NewIntHist(histLen)}
}

func (w *Mem) Update(val int, limit int) {
	if val < 0 {
		return
	}
	w.hist.Append(val)
	w.limit = limit
}

// pctColor returns the color of a usage percentage.
func pctColor(pct float64) ui.Attribute {
	switch {
	case pct >= 90:
		return ui.ColorRed
	case pct >= 70:
		return ui.ColorYellow
	}
	return ui.ColorGreen
}

func (w *Mem) Buffer() ui.Buffer {
	in := w.inner()
	_, peak := w.hist.Stats(in.Dx())
	w.BorderLabel = "MEM"
	if w.hist.Count > 0 {
		w.BorderLabel = "MEM · peak " + cwidgets.ByteFormatShort(peak)
	}
	buf := w.Block.Buffer()
	if in.Dy() <= 0 || w.hist.Count == 0 {
		return buf
	}
	x0, end := in.Min.X+1, in.Max.X-1
	val := w.hist.Val
	pct := 0.0
	if w.limit > 0 {
		pct = float64(val) * 100 / float64(w.limit)
	}
	color := pctColor(pct)

	// the values, and the percentage on the right
	drawText(buf, x0, in.Min.Y, end, fmt.Sprintf("%s / %s", cwidgets.ByteFormatShort(val), cwidgets.ByteFormatShort(w.limit)), textFg(), ui.ColorDefault)
	drawRight(buf, in.Min.Y, x0, end, fmt.Sprintf("%.1f%%", pct), color|ui.AttrBold, ui.ColorDefault)

	// the bar
	if in.Dy() >= 2 {
		y := in.Min.Y + 1
		width := end - x0
		filled := int(pct*float64(width)/100 + 0.5)
		if filled > width {
			filled = width
		}
		if filled == 0 && val > 0 {
			filled = 1
		}
		for x := 0; x < width; x++ {
			if x < filled {
				buf.Set(x0+x, y, ui.Cell{Ch: '█', Fg: color, Bg: ui.ColorDefault})
			} else {
				buf.Set(x0+x, y, ui.Cell{Ch: '░', Fg: dimFg(), Bg: ui.ColorDefault})
			}
		}
	}

	// the history, from a floor below the lowest value (see history)
	if in.Dy() >= 4 {
		r := image.Rect(in.Min.X, in.Min.Y+3, in.Max.X, in.Max.Y)
		data, top := w.history(r.Dx())
		drawSpark(buf, r, data, top, ui.ColorBlue)
	}
	return buf
}

// history returns the newest n values for the chart and the value of a
// full bar. The values are measured from a floor below the lowest one, so
// that the lowest is a quarter of the highest: the memory used rarely drops
// near zero, and bars from zero would all look full. A flat history is a
// quarter of the height.
func (w *Mem) history(n int) (data []int, top int) {
	data = w.hist.Last(n)
	real := w.hist.Count
	if real > n {
		real = n
	}
	if real == 0 {
		return data, 0
	}
	low, high := data[n-real], data[n-real]
	for _, v := range data[n-real:] {
		low = min(low, v)
		high = max(high, v)
	}
	if high == low {
		for i := n - real; i < n; i++ {
			data[i] = 1
		}
		return data, 4
	}
	floor := low - (high-low)/3
	for i := n - real; i < n; i++ {
		data[i] -= floor
	}
	return data, high - floor
}
