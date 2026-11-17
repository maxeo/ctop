package compact

import (
	ui "github.com/gizak/termui"
	"github.com/maxeo/ctop/cwidgets"
)

type CompactGrid struct {
	ui.GridBufferer
	header *CompactHeader
	cols   []CompactCol // reference columns
	Rows   []RowBufferer
	X, Y   int
	Width  int
	Height int
	Offset int // starting row offset
	Bottom int // rows kept free at the bottom of the terminal (key help)
}

func NewCompactGrid() *CompactGrid {
	cg := &CompactGrid{header: NewCompactHeader()}
	cg.rebuildHeader()
	return cg
}

func (cg *CompactGrid) Align() {
	y := cg.Y

	if cg.Offset >= len(cg.Rows) || cg.Offset < 0 {
		cg.Offset = 0
	}

	// update row ypos, width recursively
	colWidths := cg.calcWidths()
	for _, r := range cg.pageRows() {
		r.SetY(y)
		y += r.GetHeight()
		r.SetWidths(cg.Width, colWidths)
	}
}

func (cg *CompactGrid) Clear() {
	cg.Rows = []RowBufferer{}
	cg.rebuildHeader()
}

func (cg *CompactGrid) GetHeight() int { return len(cg.Rows) + cg.header.Height }
func (cg *CompactGrid) SetX(x int)     { cg.X = x }
func (cg *CompactGrid) SetY(y int)     { cg.Y = y }
func (cg *CompactGrid) SetWidth(w int) { cg.Width = w }
func (cg *CompactGrid) MaxRows() int {
	return cwidgets.TermHeight() - cg.header.Height - cg.Y - cg.Bottom
}

// calculate and return per-column width
func (cg *CompactGrid) calcWidths() []int {
	var autoCols int
	width := cg.Width
	colWidths := make([]int, len(cg.cols))

	for n, w := range cg.cols {
		colWidths[n] = w.FixedWidth()
		width -= w.FixedWidth()
		if w.FixedWidth() == 0 {
			autoCols++
		}
	}

	spacing := colSpacing * len(cg.cols)
	autoWidth := (width - spacing) / autoCols
	for n, val := range colWidths {
		if val == 0 {
			colWidths[n] = autoWidth
		}
	}
	return colWidths
}

func (cg *CompactGrid) pageRows() (rows []RowBufferer) {
	rows = append(rows, cg.header)
	end := len(cg.Rows)
	if max := cg.Offset + cg.MaxRows(); max >= cg.Offset && max < end {
		end = max // don't draw over the rows kept free at the bottom
	}
	rows = append(rows, cg.Rows[cg.Offset:end]...)
	return rows
}

func (cg *CompactGrid) Buffer() ui.Buffer {
	buf := ui.NewBuffer()
	for _, r := range cg.pageRows() {
		buf.Merge(r.Buffer())
	}
	return buf
}

func (cg *CompactGrid) AddRows(rows ...RowBufferer) {
	cg.Rows = append(cg.Rows, rows...)
}

func (cg *CompactGrid) rebuildHeader() {
	cg.cols = newRowWidgets()
	cg.header.clearFieldPars()
	for _, col := range cg.cols {
		cg.header.addFieldPar(col.Header())
	}
}
