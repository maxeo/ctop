package main

import (
	"time"

	"github.com/eqms/ctop/config"
	"github.com/eqms/ctop/cwidgets"
	"github.com/eqms/ctop/cwidgets/single"
	"github.com/eqms/ctop/widgets"
	"github.com/eqms/ctop/widgets/menu"
	ui "github.com/gizak/termui"
	tm "github.com/nsf/termbox-go"
)

func ShowConnError(err error) (exit bool) {
	tm.Clear(tm.ColorDefault, tm.ColorDefault)
	ui.DefaultEvtStream.ResetHandlers()
	defer ui.DefaultEvtStream.ResetHandlers()

	setErr := func(err error) {
		errView.Append(err.Error())
		errView.Append("attempting to reconnect...")
		ui.Render(errView)
	}

	HandleKeys("exit", func() {
		exit = true
		ui.StopLoop()
	})

	ui.Handle("/timer/1s", func(ui.Event) {
		_, err := cursor.RefreshContainers()
		if err == nil {
			ui.StopLoop()
			return
		}
		setErr(err)
	})

	ui.Handle("/sys/wnd/resize", func(e ui.Event) {
		tm.Clear(tm.ColorDefault, tm.ColorDefault) // also refreshes termbox's cached size
		errView.Resize()
		ui.Render(errView)
		log.Infof("RESIZE")
	})

	errView.Resize()
	setErr(err)
	ui.Loop()
	return exit
}

func RedrawRows(clr bool) {
	// reinit body rows
	cGrid.Clear()

	// build layout
	y := 1
	if config.GetSwitchVal("enableHeader") {
		header.SetCount(cursor.Len())
		header.SetFilter(config.GetVal("filterStr"))
		header.SetVersion(version)
		y += header.Height()
	}

	cGrid.SetY(y)
	placePanel()

	for _, c := range cursor.filtered {
		cGrid.AddRows(c.Widgets)
	}

	// the status line appearing or going away moves the bottom rows
	if shown := status.Active(); shown != statusShown {
		statusShown = shown
		clr = true
	}

	if clr {
		tm.Clear(tm.ColorDefault, tm.ColorDefault)
		log.Debugf("screen cleared")
	}

	cGrid.Align()

	// Single render call to prevent screen flicker between header and grid draw
	items := []ui.Bufferer{cGrid, gridKeys}
	if config.GetSwitchVal("enableHeader") {
		items = append(items, header)
	}
	if gridPanel != nil {
		items = append(items, gridPanel)
	}
	if statusShown {
		items = append(items, status)
	}
	ui.Render(items...)
}

// how long a status message stays below the container list
const statusTime = 10 * time.Second

// whether the status line was drawn by the last RedrawRows
var statusShown bool

// showStatus moves the queued status messages to the status line below the
// container list. Unlike a dialog it does not take the next key.
func showStatus() {
	if !log.StatusQueued() {
		return
	}
	for sm := range log.FlushStatus() {
		status.Set(sm.Text, sm.IsError, statusTime)
	}
	RedrawRows(false)
}

// gridPanel is a menu shown below the container list (sort field, columns),
// so the list stays on screen and keeps updating while it is open.
var gridPanel *menu.Menu

// placePanel keeps the rows below the list free for the open panel and the
// key help, and the selected container visible above them.
func placePanel() {
	w, h := cwidgets.TermWidth(), cwidgets.TermHeight()
	bottom := gridKeys.Height()
	if status.Active() {
		bottom += status.Height()
		status.Align()
		status.SetY(h - bottom)
	}
	if gridPanel != nil {
		bottom += gridPanel.Height
		gridPanel.X = 0
		gridPanel.Width = w
		gridPanel.Y = h - bottom
	}
	cGrid.Bottom = bottom

	max := cGrid.MaxRows()
	if max < 1 {
		return
	}
	if idx := cursor.Idx(); idx >= cGrid.Offset+max {
		cGrid.Offset = idx - max + 1
	} else if idx >= 0 && idx < cGrid.Offset {
		cGrid.Offset = idx
	}
}

// showPanel opens m below the container list with its own key help, and
// returns the function that closes it.
func showPanel(m *menu.Menu, keys []widgets.KeyHelp) (hide func()) {
	m.SetPadding(2, 1)
	saveKeys, saveAll := gridKeys.Keys, gridKeys.ShowAll
	gridKeys.Keys, gridKeys.ShowAll = keys, false
	gridPanel = m
	RedrawRows(true)
	return func() {
		gridPanel = nil
		gridKeys.Keys, gridKeys.ShowAll = saveKeys, saveAll
		RedrawRows(true)
	}
}

// key help at the bottom of the container list, like nano; h shows every key
var gridKeys = widgets.NewKeyBar([]widgets.KeyHelp{
	{Key: "Enter", Label: "Menu"},
	{Key: "l ←", Label: "Logs"},
	{Key: "o →", Label: "Details"},
	{Key: "e", Label: "Shell"},
	{Key: "f", Label: "Filter"},
	{Key: "s", Label: "Sort"},
	{Key: "a", Label: "All containers"},
	{Key: "c", Label: "Columns"},
	{Key: "h", Label: "Help"},
	{Key: "q", Label: "Quit"},
}, []widgets.KeyHelp{
	{Key: "Enter", Label: "Container menu"},
	{Key: "↑ ↓", Label: "Select container"},
	{Key: "PgUp PgDn", Label: "Page up / down"},
	{Key: "l ←", Label: "Logs"},
	{Key: "o →", Label: "Single view"},
	{Key: "e", Label: "Shell"},
	{Key: "w", Label: "Open in browser"},
	{Key: "f", Label: "Filter"},
	{Key: "s", Label: "Sort field"},
	{Key: "r", Label: "Reverse sort"},
	{Key: "a", Label: "All containers"},
	{Key: "c", Label: "Columns"},
	{Key: "H", Label: "Header"},
	{Key: "S", Label: "Save config"},
	{Key: "h", Label: "Hide help"},
	{Key: "q", Label: "Quit"},
})

func SingleView() MenuFn {
	c := cursor.Selected()
	if c == nil {
		return nil
	}

	tm.Clear(tm.ColorDefault, tm.ColorDefault)
	ui.DefaultEvtStream.ResetHandlers()
	defer ui.DefaultEvtStream.ResetHandlers()

	ex := single.NewSingle()
	c.SetUpdater(ex)

	ex.Align()
	ui.Render(ex)

	HandleKeys("up", ex.Up)
	HandleKeys("down", ex.Down)
	ui.Handle("/sys/kbd/", func(ui.Event) { ui.StopLoop() })

	ui.Handle("/timer/1s", func(ui.Event) { ui.Render(ex) })
	ui.Handle("/sys/wnd/resize", func(e ui.Event) {
		tm.Clear(tm.ColorDefault, tm.ColorDefault) // also refreshes termbox's cached size
		ex.SetWidth(cwidgets.TermWidth())
		ex.Align()
		log.Infof("resize: width=%v max-rows=%v", ex.Width, cGrid.MaxRows())
	})

	ui.Loop()
	c.SetUpdater(c.Widgets)
	return nil
}

func RefreshDisplay() error {
	// skip display refresh during scroll
	if !cursor.isScrolling {
		needsClear, err := cursor.RefreshContainers()
		if err != nil {
			return err
		}
		RedrawRows(needsClear)
	}
	return nil
}

func Display() bool {
	var menu MenuFn
	var connErr error

	cGrid.SetWidth(cwidgets.TermWidth())
	ui.DefaultEvtStream.Hook(logEvent)

	// initial draw
	header.Align()
	status.Align()
	cursor.RefreshContainers()
	RedrawRows(true)
	showStatus() // messages queued by the menus

	HandleKeys("up", cursor.Up)
	HandleKeys("down", cursor.Down)

	HandleKeys("pgup", cursor.PgUp)
	HandleKeys("pgdown", cursor.PgDown)

	HandleKeys("exit", ui.StopLoop)
	HandleKeys("help", func() {
		gridKeys.ShowAll = !gridKeys.ShowAll
		RedrawRows(true)
	})

	ui.Handle("/sys/kbd/<enter>", func(ui.Event) {
		menu = ContainerMenu
		ui.StopLoop()
	})
	ui.Handle("/sys/kbd/<left>", func(ui.Event) {
		menu = LogMenu
		ui.StopLoop()
	})
	ui.Handle("/sys/kbd/<right>", func(ui.Event) {
		menu = SingleView
		ui.StopLoop()
	})
	ui.Handle("/sys/kbd/l", func(ui.Event) {
		menu = LogMenu
		ui.StopLoop()
	})
	ui.Handle("/sys/kbd/e", func(ui.Event) {
		menu = ExecShell
		ui.StopLoop()
	})
	ui.Handle("/sys/kbd/w", func(ui.Event) {
		OpenInBrowser()
		showStatus()
	})
	ui.Handle("/sys/kbd/o", func(ui.Event) {
		menu = SingleView
		ui.StopLoop()
	})
	ui.Handle("/sys/kbd/a", func(ui.Event) {
		config.Toggle("allContainers")
		connErr = RefreshDisplay()
		if connErr != nil {
			ui.StopLoop()
		}
	})
	ui.Handle("/sys/kbd/D", func(ui.Event) {
		dumpContainer(cursor.Selected())
	})
	ui.Handle("/sys/kbd/f", func(ui.Event) {
		menu = FilterMenu
		ui.StopLoop()
	})
	ui.Handle("/sys/kbd/H", func(ui.Event) {
		config.Toggle("enableHeader")
		RedrawRows(true)
	})
	ui.Handle("/sys/kbd/r", func(e ui.Event) {
		config.Toggle("sortReversed")
	})
	ui.Handle("/sys/kbd/s", func(ui.Event) {
		menu = SortMenu
		ui.StopLoop()
	})
	ui.Handle("/sys/kbd/c", func(ui.Event) {
		menu = ColumnsMenu
		ui.StopLoop()
	})
	ui.Handle("/sys/kbd/S", func(ui.Event) {
		path, err := config.Write()
		if err == nil {
			log.Statusf("wrote config to %s", path)
		} else {
			log.StatusErr(err)
		}
		showStatus()
	})

	ui.Handle("/timer/1s", func(e ui.Event) {
		showStatus()
		connErr = RefreshDisplay()
		if connErr != nil {
			ui.StopLoop()
		}
	})

	ui.Handle("/sys/wnd/resize", func(e ui.Event) {
		tm.Clear(tm.ColorDefault, tm.ColorDefault) // also refreshes termbox's cached size
		header.Align()
		status.Align()
		cursor.ScrollPage()
		cGrid.SetWidth(cwidgets.TermWidth())
		log.Infof("resize: width=%v max-rows=%v", cGrid.Width, cGrid.MaxRows())
		RedrawRows(true)
	})

	ui.Loop()

	if connErr != nil {
		return ShowConnError(connErr)
	}

	if menu != nil {
		for menu != nil {
			menu = menu()
		}
		return false
	}

	return true
}
