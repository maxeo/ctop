package single

import (
	"fmt"
	"time"

	"github.com/maxeo/ctop/cwidgets"
	"github.com/maxeo/ctop/models"
)

// periods over which the detail views show the average, lowest and highest
// value (seconds, one point per second)
var detailPeriods = []struct {
	name string
	n    int
}{
	{"last minute", 60},
	{"last 5 minutes", 300},
	{"last 15 minutes", 900},
}

// usageRows returns, for each period with points, "avg / low / peak".
func usageRows(h *IntHist, format func(int) string) []models.DetailRow {
	var r []models.DetailRow
	for _, p := range detailPeriods {
		avg, low, peak, count := h.Range(p.n)
		if count == 0 {
			break
		}
		name := p.name
		if count < p.n {
			name = "last " + (time.Duration(count) * time.Second).String()
		}
		r = append(r, models.DetailRow{
			Key:   name,
			Value: fmt.Sprintf("avg %s · low %s · peak %s", format(avg), format(low), format(peak)),
		})
		if count < p.n {
			break // the longer periods would show the same points
		}
	}
	return r
}

func pct(n int) string { return fmt.Sprintf("%d%%", n) }

// Details returns the CPU usage now and over the last periods.
func (w *Cpu) Details() []models.DetailSection {
	r := []models.DetailRow{{Key: "now", Value: pct(w.hist.Val)}}
	if w.ncpus > 0 {
		r = append(r, models.DetailRow{Key: "host CPUs", Value: fmt.Sprint(w.ncpus)})
	}
	r = append(r, usageRows(w.hist, pct)...)
	return []models.DetailSection{{Title: "Usage", Rows: r}}
}

// Details returns the memory used now and over the last periods.
func (w *Mem) Details() []models.DetailSection {
	r := []models.DetailRow{
		{Key: "used", Value: cwidgets.ByteFormatShort(w.hist.Val)},
		{Key: "limit", Value: cwidgets.ByteFormatShort(w.limit)},
	}
	if w.limit > 0 {
		r = append(r, models.DetailRow{Key: "used of limit", Value: fmt.Sprintf("%.1f%%", float64(w.hist.Val)*100/float64(w.limit))})
	}
	r = append(r, usageRows(w.hist, cwidgets.ByteFormatShort)...)
	return []models.DetailSection{{Title: "Usage", Rows: r}}
}

// Details returns, for each direction, the rate now and over the last
// periods, and the total since the container started.
func (r *Rates) Details() []models.DetailSection {
	var s []models.DetailSection
	for i, h := range r.hists {
		rows := []models.DetailRow{{Key: "now", Value: rate(h.Val)}}
		if t := h.Total(); t >= 0 {
			rows = append(rows, models.DetailRow{Key: "total", Value: cwidgets.ByteFormatShort(t)})
		}
		rows = append(rows, usageRows(h.IntHist, rate)...)
		s = append(s, models.DetailSection{Title: r.names[i], Rows: rows})
	}
	return s
}
