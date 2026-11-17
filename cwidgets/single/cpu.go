package single

import (
	"fmt"

	ui "github.com/gizak/termui"
)

// Cpu shows the CPU usage history, with the current, average and highest
// value of the period shown.
type Cpu struct {
	box
	hist  *IntHist
	ncpus int
}

func NewCpu() *Cpu {
	return &Cpu{box: newBox("CPU"), hist: NewIntHist(histLen)}
}

func (w *Cpu) Update(val, ncpus int) {
	w.ncpus = ncpus
	if val >= 0 {
		w.hist.Append(val)
	}
}

// cpuScale returns the top of the CPU chart: a round value above peak, so
// that a low usage does not fill the chart.
func cpuScale(peak int) int {
	for _, top := range []int{5, 10, 25, 50, 100} {
		if peak <= top {
			return top
		}
	}
	return (peak + 99) / 100 * 100 // more than one core
}

func (w *Cpu) Buffer() ui.Buffer {
	in := w.inner()
	avg, peak := w.hist.Stats(in.Dx())
	w.BorderLabel = "CPU"
	if w.hist.Count > 0 {
		w.BorderLabel = fmt.Sprintf("CPU %d%% · avg %d%% · peak %d%%", w.hist.Val, avg, peak)
	}
	buf := w.Block.Buffer()
	if in.Dy() <= 0 {
		return buf
	}
	top := cpuScale(peak)
	drawSpark(buf, in, w.hist.Last(in.Dx()), top, ui.ColorGreen)
	// the scale, in the top right corner
	drawRight(buf, in.Min.Y, in.Min.X, in.Max.X, fmt.Sprintf("%d%%", top), dimFg(), ui.ColorDefault)
	return buf
}
