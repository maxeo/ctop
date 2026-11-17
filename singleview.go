package main

import (
	"strings"
	"time"
	"unicode/utf8"

	ui "github.com/gizak/termui"
	"github.com/maxeo/ctop/cwidgets/single"
	"github.com/maxeo/ctop/widgets"
	tm "github.com/nsf/termbox-go"
)

// custom event sent escWait after <escape> in the single view
const singleEscEvent = "/single/esc"

var (
	// nothing selected
	singleKeys = []widgets.KeyHelp{
		{Key: "Enter", Label: "Select panel"},
		{Key: "PgUp/Dn", Label: "Container"},
		{Key: "↑ ↓", Label: "Scroll env"},
		{Key: "/", Label: "Filter env"},
		{Key: "l", Label: "Logs"},
		{Key: "m", Label: "Menu"},
		{Key: "e", Label: "Shell"},
		{Key: "Esc ← →", Label: "List"},
	}
	// a panel selected
	singleSelectKeys = []widgets.KeyHelp{
		{Key: "Enter", Label: "Details"},
		{Key: "← → ↑ ↓", Label: "Move"},
		{Key: "PgUp/Dn", Label: "Container"},
		{Key: "Esc", Label: "Unselect"},
		{Key: "l", Label: "Logs"},
		{Key: "q", Label: "List"},
	}
	// a detail view open
	singleDetailKeys = []widgets.KeyHelp{
		{Key: "↑ ↓ PgUp/Dn", Label: "Scroll"},
		{Key: "← →", Label: "Other panel"},
		{Key: "/", Label: "Filter"},
		{Key: "Home End", Label: "Top/end"},
		{Key: "Esc", Label: "Back"},
		{Key: "q", Label: "List"},
	}
	// typing a filter
	singleFilterKeys = []widgets.KeyHelp{
		{Key: "Enter", Label: "Keep filter"},
		{Key: "Esc", Label: "Clear filter"},
		{Key: "^U", Label: "Clear text"},
		{Key: "↑ ↓", Label: "Scroll"},
	}
)

// singleView handles the keys of the single view. It has three modes (see
// single.Mode): nothing selected (the arrows scroll the environment or go
// back to the list), a panel selected (the arrows move the selection) and a
// detail view (the arrows scroll it or show the next panel's).
type singleView struct {
	ex        *single.Single
	next      MenuFn // what to show after the view, nil for the list
	csi       csiKeys
	escSeq    int
	escID     int
	filtering bool
	filter    string
	done      bool
}

func SingleView() MenuFn {
	c := cursor.Selected()
	if c == nil {
		return nil
	}

	_ = tm.Clear(tm.ColorDefault, tm.ColorDefault)
	ui.DefaultEvtStream.ResetHandlers()
	defer ui.DefaultEvtStream.ResetHandlers()

	ex := single.NewSingle()
	ex.SetKeys(singleKeys)
	c.SetUpdater(ex)
	defer c.SetUpdater(c.Widgets)

	ex.Align()
	ui.Render(ex)

	sv := &singleView{ex: ex}
	ui.Handle("/sys/kbd/", func(e ui.Event) {
		sv.key(strings.TrimPrefix(e.Path, "/sys/kbd/"))
	})
	ui.Handle(singleEscEvent, func(e ui.Event) {
		if id, ok := e.Data.(int); ok && id == sv.escID {
			sv.escID = 0
			sv.escape()
		}
	})
	// the wheel works like the up and down arrows (Windows only, see
	// HandleWheel)
	HandleWheel(func() { sv.key("<up>") }, func() { sv.key("<down>") })
	ui.Handle("/timer/1s", func(ui.Event) { ui.Render(ex) })
	ui.Handle("/sys/wnd/resize", func(e ui.Event) {
		_ = tm.Clear(tm.ColorDefault, tm.ColorDefault) // also refreshes termbox's cached size
		ex.Align()
		ui.Render(ex)
	})

	ui.Loop()
	return sv.next
}

// close leaves the view and shows next (nil: the list), once.
func (v *singleView) close(next MenuFn) {
	if v.done {
		return
	}
	v.done = true
	v.next = next
	ui.StopLoop()
}

func (v *singleView) key(k string) {
	if v.done {
		return
	}
	if v.escID != 0 {
		// <escape> just before: the start of a sequence, or Esc then a key
		v.escID = 0
		if k == "[" || k == "O" {
			v.csi.Start()
			return
		}
		v.escape()
		if v.done {
			return
		}
	}
	if k == "<escape>" {
		v.escSeq++
		v.escID = v.escSeq
		id := v.escID
		time.AfterFunc(escWait, func() { ui.SendCustomEvt(singleEscEvent, id) })
		return
	}
	if k == "M-[" || k == "M-O" {
		v.csi.Start()
		return
	}
	if k == "C-c" {
		// Ctrl+C works like Esc: back one level
		v.escape()
		return
	}
	if action, consumed := v.csi.Feed(k); consumed {
		switch action {
		case "home":
			k = "<home>"
		case "end":
			k = "<end>"
		default:
			return
		}
	}

	mode := v.ex.Mode()
	if v.filtering {
		switch k {
		case "<up>":
			v.ex.Scroll(-1)
		case "<down>":
			v.ex.Scroll(1)
		case "<previous>":
			v.ex.Page(-1)
		case "<next>":
			v.ex.Page(1)
		default:
			v.filterKey(k)
		}
		return
	}

	// keys that work the same in every mode
	switch k {
	case "q":
		v.close(nil)
		return
	case "l":
		v.close(LogMenu)
		return
	case "e":
		v.close(ExecShell)
		return
	case "m":
		v.close(ContainerMenu)
		return
	}

	switch mode {
	case single.ModeView:
		v.viewKey(k)
	case single.ModeSelect:
		v.selectKey(k)
	case single.ModeDetail:
		v.detailKey(k)
	}
}

// viewKey handles a key with nothing selected.
func (v *singleView) viewKey(k string) {
	switch k {
	case "<up>", "k":
		v.ex.Scroll(-1)
	case "<down>", "j":
		v.ex.Scroll(1)
	case "<home>":
		v.ex.Top()
	case "<end>":
		v.ex.Bottom()
	case "<previous>":
		v.step(-1)
	case "<next>":
		v.step(1)
	case "<left>", "<right>":
		v.close(nil)
	case "<enter>":
		v.ex.Select()
		v.ex.SetKeys(singleSelectKeys)
	case "/", "C-f":
		v.startFilter()
	}
}

// selectKey handles a key with a panel selected.
func (v *singleView) selectKey(k string) {
	switch k {
	case "<up>", "k":
		v.ex.MoveSelection(0, -1)
	case "<down>", "j":
		v.ex.MoveSelection(0, 1)
	case "<left>", "h":
		v.ex.MoveSelection(-1, 0)
	case "<right>":
		v.ex.MoveSelection(1, 0)
	case "<previous>":
		v.step(-1)
	case "<next>":
		v.step(1)
	case "<enter>":
		v.ex.OpenDetail()
		v.resetFilter()
		v.ex.SetKeys(singleDetailKeys)
	}
}

// detailKey handles a key in a detail view.
func (v *singleView) detailKey(k string) {
	switch k {
	case "<up>", "k":
		v.ex.Scroll(-1)
	case "<down>", "j":
		v.ex.Scroll(1)
	case "<previous>":
		v.ex.Page(-1)
	case "<next>", "<space>":
		v.ex.Page(1)
	case "<home>":
		v.ex.Top()
	case "<end>":
		v.ex.Bottom()
	case "<left>":
		v.resetFilter()
		v.ex.StepDetail(-1)
	case "<right>":
		v.resetFilter()
		v.ex.StepDetail(1)
	case "/", "C-f":
		v.startFilter()
	}
}

// step shows the previous (-1) or next (1) container of the list.
func (v *singleView) step(delta int) {
	if cursor.Step(delta) {
		v.close(SingleView)
	}
}

func (v *singleView) startFilter() {
	v.filtering = true
	v.showFilter()
}

func (v *singleView) resetFilter() {
	v.filtering = false
	v.filter = ""
}

// filterKey handles a key typed in the filter.
func (v *singleView) filterKey(k string) {
	switch {
	case k == "<enter>":
		v.filtering = false
	case backspaceKeys[k]:
		if v.filter == "" {
			return
		}
		_, size := utf8.DecodeLastRuneInString(v.filter)
		v.filter = v.filter[:len(v.filter)-size]
	case k == "C-u":
		v.filter = ""
	case k == "<space>":
		v.filter += " "
	case utf8.RuneCountInString(k) == 1:
		v.filter += k
	default:
		return
	}
	v.showFilter()
}

// modeKeys returns the key help of the current mode.
func (v *singleView) modeKeys() []widgets.KeyHelp {
	switch v.ex.Mode() {
	case single.ModeSelect:
		return singleSelectKeys
	case single.ModeDetail:
		return singleDetailKeys
	}
	return singleKeys
}

func (v *singleView) showFilter() {
	if v.filtering {
		v.ex.SetKeys(singleFilterKeys)
	} else {
		v.ex.SetKeys(v.modeKeys())
	}
	v.ex.Filter(v.filter, v.filtering)
}

// escape goes back one level: it clears the filter, closes the detail
// view, leaves the selection, otherwise goes back to the list.
func (v *singleView) escape() {
	switch {
	case v.filtering || v.filter != "":
		v.resetFilter()
		v.showFilter()
	case v.ex.Mode() == single.ModeDetail:
		v.ex.CloseDetail()
		v.ex.SetKeys(singleSelectKeys)
	case v.ex.Mode() == single.ModeSelect:
		v.ex.Unselect()
		v.ex.SetKeys(singleKeys)
	default:
		v.close(nil)
	}
}
