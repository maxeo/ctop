package single

// histLen is how many points a chart keeps: enough to fill the width of any
// usual terminal (one point per column).
const histLen = 1024

// IntHist keeps the last values of a metric.
type IntHist struct {
	Val   int   // most current data point
	Data  []int // historical data points, padded with zeros on the left
	Count int   // points added, at most len(Data)
}

func NewIntHist(max int) *IntHist {
	return &IntHist{Data: make([]int, max)}
}

func (h *IntHist) Append(val int) {
	copy(h.Data, h.Data[1:])
	h.Data[len(h.Data)-1] = val
	h.Val = val
	if h.Count < len(h.Data) {
		h.Count++
	}
}

// Last returns a copy of the newest n points, padded with zeros on the left
// when there are fewer, so a chart drawn from the left ends on its right edge.
func (h *IntHist) Last(n int) []int {
	return lastN(h.Data, n)
}

// Stats returns the average and the highest of the newest n points added
// (the zeros padding a new history do not count).
func (h *IntHist) Stats(n int) (avg, peak int) {
	if n > h.Count {
		n = h.Count
	}
	if n <= 0 {
		return 0, 0
	}
	sum := 0
	for _, v := range h.Data[len(h.Data)-n:] {
		sum += v
		if v > peak {
			peak = v
		}
	}
	return (sum + n/2) / n, peak
}

// Range returns the average, lowest and highest of the newest n points
// added, and how many there were.
func (h *IntHist) Range(n int) (avg, low, peak, count int) {
	if n > h.Count {
		n = h.Count
	}
	if n <= 0 {
		return 0, 0, 0, 0
	}
	data := h.Data[len(h.Data)-n:]
	low, peak = data[0], data[0]
	sum := 0
	for _, v := range data {
		sum += v
		low = min(low, v)
		peak = max(peak, v)
	}
	return (sum + n/2) / n, low, peak, n
}

// DiffHist keeps the differences between the values added (e.g. bytes per
// second from a byte counter).
type DiffHist struct {
	*IntHist
	lastVal int
}

// Total returns the last value added (the counter), -1 if none.
func (h *DiffHist) Total() int { return h.lastVal }

func NewDiffHist(max int) *DiffHist {
	return &DiffHist{NewIntHist(max), -1}
}

func (h *DiffHist) Append(val int) {
	if h.lastVal >= 0 && val >= h.lastVal { // skip the first update, and counter resets
		h.IntHist.Append(val - h.lastVal)
	}
	h.lastVal = val
}

func lastN(data []int, n int) []int {
	if n < 0 {
		n = 0
	}
	out := make([]int, n)
	if len(data) > n {
		data = data[len(data)-n:]
	}
	copy(out[n-len(data):], data)
	return out
}
