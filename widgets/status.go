package widgets

import (
	"time"

	"github.com/eqms/ctop/cwidgets"
	ui "github.com/gizak/termui"
)

var (
	statusHeight = 1
)

type StatusLine struct {
	Message *ui.Par
	bg      *ui.Par
	until   time.Time // the message is hidden from then on
}

func NewStatusLine() *StatusLine {
	p := ui.NewPar("")
	p.X = 2
	p.Border = false
	p.Height = statusHeight
	p.Bg = ui.ThemeAttr("header.bg")
	p.TextFgColor = ui.ThemeAttr("header.fg")
	p.TextBgColor = ui.ThemeAttr("header.bg")
	return &StatusLine{
		Message: p,
		bg:      statusBg(),
	}
}

// Set shows s for d, without stopping the page it is shown on: the next
// key still goes to the page.
func (sl *StatusLine) Set(s string, isErr bool, d time.Duration) {
	if isErr {
		sl.Message.TextFgColor = ui.ThemeAttr("status.danger")
	} else {
		sl.Message.TextFgColor = ui.ThemeAttr("header.fg")
	}
	sl.Message.Text = s
	sl.until = time.Now().Add(d)
}

// Active tells whether a message is shown.
func (sl *StatusLine) Active() bool {
	return sl.Message.Text != "" && time.Now().Before(sl.until)
}

// SetY places the status line on row y.
func (sl *StatusLine) SetY(y int) {
	sl.bg.Y = y
	sl.Message.Y = y
}

func (sl *StatusLine) Buffer() ui.Buffer {
	buf := ui.NewBuffer()
	buf.Merge(sl.bg.Buffer())
	buf.Merge(sl.Message.Buffer())
	return buf
}

func (sl *StatusLine) Align() {
	sl.bg.SetWidth(cwidgets.TermWidth() - 1)
	sl.Message.SetWidth(cwidgets.TermWidth() - 2)

	sl.bg.Y = cwidgets.TermHeight() - 1
	sl.Message.Y = cwidgets.TermHeight() - 1
}

func (sl *StatusLine) Height() int { return statusHeight }

func statusBg() *ui.Par {
	bg := ui.NewPar("")
	bg.X = 1
	bg.Height = statusHeight
	bg.Border = false
	bg.Bg = ui.ThemeAttr("header.bg")
	return bg
}
