package single

import (
	"image"

	ui "github.com/gizak/termui"
	"github.com/maxeo/ctop/cwidgets"
)

// Rates shows two per-second rates (e.g. RX / TX) with their history,
// filling the whole width with the newest value on the right edge.
type Rates struct {
	box
	hists  [2]*DiffHist
	names  [2]string
	colors [2]ui.Attribute
	note   string // shown when there is no traffic, e.g. on the host network
}

func newRates(label string, names [2]string) *Rates {
	return &Rates{
		box:    newBox(label),
		hists:  [2]*DiffHist{NewDiffHist(histLen), NewDiffHist(histLen)},
		names:  names,
		colors: [2]ui.Attribute{ui.ColorGreen, ui.ColorYellow},
	}
}

// NewNet shows the network traffic received and sent.
func NewNet() *Rates { return newRates("NET", [2]string{"↓ RX", "↑ TX"}) }

// NewIO shows the bytes read from and written to the disks.
func NewIO() *Rates { return newRates("IO", [2]string{"↓ read", "↑ write"}) }

// Update adds the totals read now; the charts show the difference from the
// previous update. Negative values (not available) are skipped.
func (r *Rates) Update(a, b int64) {
	for i, v := range [2]int64{a, b} {
		if v >= 0 {
			r.hists[i].Append(int(v))
		}
	}
}

func rate(n int) string { return cwidgets.ByteFormatShort(n) + "/s" }

func (r *Rates) Buffer() ui.Buffer {
	buf := r.Block.Buffer()
	in := r.inner()
	x0, end := in.Min.X+1, in.Max.X-1
	// each rate: a title row, then its chart
	part := in.Dy() / 2
	for i, h := range r.hists {
		y := in.Min.Y + i*part
		if part <= 0 || y >= in.Max.Y {
			break
		}
		_, peak := h.Stats(in.Dx())
		x := drawText(buf, x0, y, end, r.names[i]+" ", r.colors[i]|ui.AttrBold, ui.ColorDefault)
		x = drawText(buf, x, y, end, rate(h.Val), textFg()|ui.AttrBold, ui.ColorDefault)
		drawText(buf, x, y, end, "   peak "+rate(peak), dimFg(), ui.ColorDefault)
		if part > 1 {
			chart := image.Rect(in.Min.X, y+1, in.Max.X, y+part)
			drawSpark(buf, chart, h.Last(chart.Dx()), 0, r.colors[i])
		}
	}
	if r.note != "" && in.Dy() > 2 && r.hists[0].Val == 0 && r.hists[1].Val == 0 {
		drawText(buf, x0, in.Min.Y+1, end, r.note, dimFg(), ui.ColorDefault)
	}
	return buf
}
