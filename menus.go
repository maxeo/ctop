package main

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/eqms/ctop/config"
	"github.com/eqms/ctop/container"
	"github.com/eqms/ctop/cwidgets"
	"github.com/eqms/ctop/widgets"
	"github.com/eqms/ctop/widgets/menu"
	ui "github.com/gizak/termui"
	tm "github.com/nsf/termbox-go"
)

// MenuFn executes a menu window, returning the next menu or nil
type MenuFn func() MenuFn

// ctop's input mode. In termbox's Alt mode a lone Esc is never reported
// (termbox waits for the next key and sends Alt+key), so Esc could close
// nothing; ctop uses no Alt+key shortcut.
const inputMode = tm.InputEsc

// inputEscMode makes sure termbox is in Esc input mode while a text input or
// panel is open, and returns the function that restores ctop's mode.
func inputEscMode() (restore func()) {
	tm.SetInputMode(tm.InputEsc)
	return func() { tm.SetInputMode(inputMode) }
}

// inputY places an input box at the bottom, above the key help and the
// status line.
func inputY(i *widgets.Input) int {
	return cwidgets.TermHeight() - i.Height - cGrid.Bottom
}

func FilterMenu() MenuFn {
	ui.DefaultEvtStream.ResetHandlers()
	defer ui.DefaultEvtStream.ResetHandlers()

	defer inputEscMode()()

	i := widgets.NewInput()
	i.BorderLabel = "Filter"
	i.SetY(inputY(i))
	i.Data = config.GetVal("filterStr")
	ui.Render(i)

	// refresh container rows on input
	i.OnChange(func(s string) {
		config.Update("filterStr", s)
		_ = RefreshDisplay()
	})

	i.InputHandlers()
	cancel := func(ui.Event) {
		config.Update("filterStr", "")
		ui.StopLoop()
	}
	ui.Handle("/sys/kbd/<escape>", cancel)
	ui.Handle("/sys/kbd/C-c", cancel)
	ui.Handle("/sys/kbd/<enter>", func(ui.Event) {
		config.Update("filterStr", i.Data)
		ui.StopLoop()
	})
	ui.Loop()
	return nil
}

// panelLoop runs the event loop of a panel below the container list: the
// list keeps updating, and Esc, q and Ctrl+C call close.
func panelLoop(close func()) {
	defer inputEscMode()()
	ui.Handle("/sys/kbd/q", func(ui.Event) { close() })
	ui.Handle("/sys/kbd/C-c", func(ui.Event) { close() })
	ui.Handle("/sys/kbd/<escape>", func(ui.Event) { close() })
	ui.Handle("/timer/1s", func(ui.Event) { _ = RefreshDisplay() })
	ui.Handle("/sys/wnd/resize", func(ui.Event) {
		tm.Clear(tm.ColorDefault, tm.ColorDefault) // also refreshes termbox's cached size
		header.Align()
		cGrid.SetWidth(cwidgets.TermWidth())
		RedrawRows(true)
	})
	ui.Loop()
}

var sortKeys = []widgets.KeyHelp{
	{Key: "↑ ↓", Label: "Sort field"},
	{Key: "r", Label: "Reverse"},
	{Key: "Enter", Label: "Keep"},
	{Key: "Esc", Label: "Cancel"},
}

// SortMenu picks the sort field below the container list: moving the cursor
// sorts the list at once, Enter keeps the field, Esc goes back to the old one.
func SortMenu() MenuFn {
	ui.DefaultEvtStream.ResetHandlers()
	defer ui.DefaultEvtStream.ResetHandlers()

	m := menu.NewMenu()
	m.Selectable = true
	m.SortItems = true
	m.BorderLabel = "Sort Field"
	for _, field := range container.SortFields() {
		m.AddItems(menu.Item{Val: field})
	}

	oldField := config.GetVal("sortField")
	oldReversed := config.GetSwitchVal("sortReversed")
	m.SetCursor(oldField)

	hide := showPanel(m, sortKeys)
	defer hide()

	apply := func() {
		config.Update("sortField", m.SelectedValue())
		_ = RefreshDisplay()
	}
	HandleKeys("up", func() { m.Up(); apply() })
	HandleKeys("down", func() { m.Down(); apply() })
	ui.Handle("/sys/kbd/r", func(ui.Event) {
		config.Toggle("sortReversed")
		_ = RefreshDisplay()
	})
	ui.Handle("/sys/kbd/<enter>", func(ui.Event) { ui.StopLoop() })
	ui.Handle("/sys/kbd/s", func(ui.Event) { ui.StopLoop() })

	panelLoop(func() {
		config.Update("sortField", oldField)
		if config.GetSwitchVal("sortReversed") != oldReversed {
			config.Toggle("sortReversed")
		}
		ui.StopLoop()
	})
	return nil
}

var columnsKeys = []widgets.KeyHelp{
	{Key: "↑ ↓", Label: "Select column"},
	{Key: "Enter", Label: "Show / hide"},
	{Key: "PgUp", Label: "Move left"},
	{Key: "PgDn", Label: "Move right"},
	{Key: "Esc", Label: "Close"},
}

// ColumnsMenu shows, hides and moves columns below the container list; the
// list shows each change at once.
func ColumnsMenu() MenuFn {
	const (
		enabledStr  = "[X]"
		disabledStr = "[ ]"
		padding     = 2
	)

	ui.DefaultEvtStream.ResetHandlers()
	defer ui.DefaultEvtStream.ResetHandlers()

	m := menu.NewMenu()
	m.Selectable = true
	m.SortItems = false
	m.BorderLabel = "Columns"

	rebuild := func() {
		// get padding for right alignment of enabled status
		var maxLen int
		for _, col := range config.GlobalColumns {
			if len(col.Label) > maxLen {
				maxLen = len(col.Label)
			}
		}
		maxLen += padding

		// rebuild menu items
		m.ClearItems()
		for _, col := range config.GlobalColumns {
			txt := col.Label + strings.Repeat(" ", maxLen-len(col.Label))
			if col.Enabled {
				txt += enabledStr
			} else {
				txt += disabledStr
			}
			m.AddItems(menu.Item{Val: col.Name, Label: txt})
		}
	}

	// recreate the container rows with the new columns
	update := func() {
		rebuild()
		if cSource, err := cursor.cSuper.Get(); err == nil {
			for _, c := range cSource.All() {
				c.RecreateWidgets()
			}
		}
		RedrawRows(true)
	}

	rebuild()
	hide := showPanel(m, columnsKeys)
	defer hide()

	toggle := func() {
		config.ColumnToggle(m.SelectedValue())
		update()
	}
	HandleKeys("up", m.Up)
	HandleKeys("down", m.Down)
	HandleKeys("pgup", func() {
		config.ColumnLeft(m.SelectedValue())
		m.Up()
		update()
	})
	HandleKeys("pgdown", func() {
		config.ColumnRight(m.SelectedValue())
		m.Down()
		update()
	})
	ui.Handle("/sys/kbd/x", func(ui.Event) { toggle() })
	ui.Handle("/sys/kbd/<space>", func(ui.Event) { toggle() })
	ui.Handle("/sys/kbd/<enter>", func(ui.Event) { toggle() })
	ui.Handle("/sys/kbd/c", func(ui.Event) { ui.StopLoop() })

	panelLoop(ui.StopLoop)
	return nil
}

func ContainerMenu() MenuFn {
	c := cursor.Selected()
	if c == nil {
		return nil
	}

	ui.DefaultEvtStream.ResetHandlers()
	defer ui.DefaultEvtStream.ResetHandlers()

	m := menu.NewMenu()
	m.Selectable = true
	m.BorderLabel = "Menu"

	items := []menu.Item{
		menu.Item{Val: "single", Label: "[o] single view"},
		menu.Item{Val: "logs", Label: "[l] log view"},
	}

	if c.Meta["state"] == "running" {
		items = append(items, menu.Item{Val: "stop", Label: "[s] stop"})
		items = append(items, menu.Item{Val: "pause", Label: "[p] pause"})
		items = append(items, menu.Item{Val: "restart", Label: "[r] restart"})
		items = append(items, menu.Item{Val: "stopremove", Label: "[X] stop + remove"})
		items = append(items, menu.Item{Val: "exec", Label: "[e] exec shell"})
		if c.Meta["Web Port"] != "" {
			items = append(items, menu.Item{Val: "browser", Label: "[w] open in browser"})
		}
	}
	if c.Meta["state"] == "exited" || c.Meta["state"] == "created" {
		items = append(items, menu.Item{Val: "start", Label: "[s] start"})
		items = append(items, menu.Item{Val: "remove", Label: "[R] remove"})
	}
	if c.Meta["state"] == "paused" {
		items = append(items, menu.Item{Val: "unpause", Label: "[p] unpause"})
	}
	items = append(items, menu.Item{Val: "commit", Label: "[C] commit to image"})
	items = append(items, menu.Item{Val: "cancel", Label: "[c] cancel"})

	m.AddItems(items...)
	ui.Render(m)

	HandleKeys("up", m.Up)
	HandleKeys("down", m.Down)

	var selected string

	// shortcuts
	ui.Handle("/sys/kbd/o", func(ui.Event) {
		selected = "single"
		ui.StopLoop()
	})
	ui.Handle("/sys/kbd/l", func(ui.Event) {
		selected = "logs"
		ui.StopLoop()
	})
	if c.Meta["state"] != "paused" {
		ui.Handle("/sys/kbd/s", func(ui.Event) {
			if c.Meta["state"] == "running" {
				selected = "stop"
			} else {
				selected = "start"
			}
			ui.StopLoop()
		})
	}
	if c.Meta["state"] != "exited" && c.Meta["state"] != "created" {
		ui.Handle("/sys/kbd/p", func(ui.Event) {
			if c.Meta["state"] == "paused" {
				selected = "unpause"
			} else {
				selected = "pause"
			}
			ui.StopLoop()
		})
	}
	if c.Meta["state"] == "running" {
		ui.Handle("/sys/kbd/e", func(ui.Event) {
			selected = "exec"
			ui.StopLoop()
		})
		ui.Handle("/sys/kbd/r", func(ui.Event) {
			selected = "restart"
			ui.StopLoop()
		})
		if c.Meta["Web Port"] != "" {
			ui.Handle("/sys/kbd/w", func(ui.Event) {
				selected = "browser"
				ui.StopLoop()
			})
		}
	}
	ui.Handle("/sys/kbd/R", func(ui.Event) {
		selected = "remove"
		ui.StopLoop()
	})
	ui.Handle("/sys/kbd/X", func(ui.Event) {
		selected = "stopremove"
		ui.StopLoop()
	})
	ui.Handle("/sys/kbd/C", func(ui.Event) {
		selected = "commit"
		ui.StopLoop()
	})
	ui.Handle("/sys/kbd/c", func(ui.Event) {
		ui.StopLoop()
	})

	ui.Handle("/sys/kbd/<enter>", func(ui.Event) {
		selected = m.SelectedValue()
		ui.StopLoop()
	})
	ui.Handle("/sys/kbd/", func(ui.Event) {
		ui.StopLoop()
	})
	ui.Loop()

	var nextMenu MenuFn
	switch selected {
	case "single":
		nextMenu = SingleView
	case "logs":
		nextMenu = LogMenu
	case "exec":
		nextMenu = ExecShell
	case "browser":
		nextMenu = OpenInBrowser
	case "start":
		nextMenu = Confirm(confirmTxt("start", c.GetMeta("name")), c.Start)
	case "stop":
		nextMenu = Confirm(confirmTxt("stop", c.GetMeta("name")), c.Stop)
	case "remove":
		nextMenu = Confirm(confirmTxt("remove", c.GetMeta("name")), c.Remove)
	case "pause":
		nextMenu = Confirm(confirmTxt("pause", c.GetMeta("name")), c.Pause)
	case "unpause":
		nextMenu = Confirm(confirmTxt("unpause", c.GetMeta("name")), c.Unpause)
	case "restart":
		nextMenu = Confirm(confirmTxt("restart", c.GetMeta("name")), c.Restart)
	case "stopremove":
		nextMenu = Confirm(confirmTxt("stop and remove", c.GetMeta("name")), c.StopAndRemove)
	case "commit":
		nextMenu = CommitMenu
	}

	return nextMenu
}

func ExecShell() MenuFn {
	c := cursor.Selected()
	if c == nil {
		return nil
	}

	// Suspend TUI — release terminal for interactive shell
	ui.Close()

	// Execute a login shell directly in the container.
	// The shell is configurable via --shell flag, CTOP_SHELL env, or config file.
	shell := config.GetVal("shell")

	// Validate shell against whitelist to prevent arbitrary command execution
	if shell != "" && !isAllowedShell(shell) {
		log.StatusErr(fmt.Errorf("shell not allowed: %s (allowed: sh, bash, zsh, fish, or absolute paths under /bin/ /usr/bin/)", shell))
		return ContainerMenu
	}

	var execCmd []string
	if shell == "/bin/sh" || shell == "sh" || shell == "" {
		execCmd = []string{
			"sh", "-c",
			`if command -v fish >/dev/null 2>&1; then exec fish -l; elif command -v zsh >/dev/null 2>&1; then exec zsh -l; elif command -v bash >/dev/null 2>&1; then exec bash -l; else exec sh -l; fi`,
		}
	} else {
		execCmd = []string{shell, "-l"}
	}

	if err := c.Exec(execCmd); err != nil {
		log.StatusErr(err)
	}

	// Resume TUI
	if err := ui.Init(); err != nil {
		panic(err)
	}
	tm.SetInputMode(inputMode)
	tm.Sync()
	tm.Clear(tm.ColorDefault, tm.ColorDefault)

	// Hintergrund neu zeichnen bevor das Menü wieder gerendert wird
	_ = RefreshDisplay()

	return ContainerMenu
}

func OpenInBrowser() MenuFn {
	c := cursor.Selected()
	if c == nil {
		return nil
	}

	webPort := c.Meta.Get("Web Port")
	if webPort == "" {
		log.Statusf("%s has no published TCP port to open", c.GetMeta("name"))
		return nil
	}
	link := webURL(webPort)
	if !canOpenBrowser() {
		log.Statusf("open %s in your browser", link)
		return nil
	}
	if err := startBrowser(link); err != nil {
		log.Statusf("cannot open a browser here, open %s", link)
	}
	return nil
}

// startBrowser opens url without waiting for the browser. Its output goes to
// /dev/null: the browser (e.g. Chromium) would otherwise write its errors
// over the screen ctop is drawing on.
func startBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	// nil Stdin, Stdout and Stderr are connected to the null device
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }()
	return nil
}

// webURL returns the URL of a published port. Over SSH, "localhost" is the
// remote host, so it is replaced with the address the SSH client connected to.
func webURL(hostPort string) string {
	host, port, err := net.SplitHostPort(hostPort)
	if err == nil && host == "localhost" {
		if f := strings.Fields(os.Getenv("SSH_CONNECTION")); len(f) == 4 {
			hostPort = net.JoinHostPort(f[2], port)
		}
	}
	return "http://" + hostPort + "/"
}

// canOpenBrowser tells whether a browser can open on this screen: not over
// SSH, and on Linux only with a graphical session.
func canOpenBrowser() bool {
	if os.Getenv("SSH_CONNECTION") != "" || os.Getenv("SSH_TTY") != "" {
		return false
	}
	if runtime.GOOS == "linux" {
		return os.Getenv("DISPLAY") != "" || os.Getenv("WAYLAND_DISPLAY") != ""
	}
	return true
}

// Create a confirmation dialog with a given description string and
// func to perform if confirmed
func Confirm(txt string, fn func()) MenuFn {
	menu := func() MenuFn {
		ui.DefaultEvtStream.ResetHandlers()
		defer ui.DefaultEvtStream.ResetHandlers()

		m := menu.NewMenu()
		m.Selectable = true
		m.BorderLabel = "Confirm"
		m.SubText = txt

		items := []menu.Item{
			menu.Item{Val: "cancel", Label: "[c]ancel"},
			menu.Item{Val: "yes", Label: "[y]es"},
		}

		var response bool

		m.AddItems(items...)
		ui.Render(m)

		yes := func() {
			response = true
			ui.StopLoop()
		}

		no := func() {
			response = false
			ui.StopLoop()
		}

		HandleKeys("up", m.Up)
		HandleKeys("down", m.Down)
		HandleKeys("exit", no)
		ui.Handle("/sys/kbd/c", func(ui.Event) { no() })
		ui.Handle("/sys/kbd/y", func(ui.Event) { yes() })

		ui.Handle("/sys/kbd/<enter>", func(ui.Event) {
			switch m.SelectedValue() {
			case "cancel":
				no()
			case "yes":
				yes()
			}
		})

		ui.Loop()
		if response {
			fn()
		}
		return nil
	}
	return menu
}

func CommitMenu() MenuFn {
	c := cursor.Selected()
	if c == nil {
		return nil
	}

	ui.DefaultEvtStream.ResetHandlers()
	defer ui.DefaultEvtStream.ResetHandlers()

	// Generate suggested image name from container name
	name := strings.ReplaceAll(c.GetMeta("name"), "/", "-")
	suggested := fmt.Sprintf("%s-snapshot", name)

	defer inputEscMode()()

	i := widgets.NewInput()
	i.BorderLabel = "Commit as Image"
	i.SetMaxLen(60)
	i.SetY(inputY(i))
	i.Data = suggested
	ui.Render(i)

	var imageName string
	i.InputHandlers()
	ui.Handle("/sys/kbd/<escape>", func(ui.Event) { ui.StopLoop() })
	ui.Handle("/sys/kbd/C-c", func(ui.Event) { ui.StopLoop() })
	ui.Handle("/sys/kbd/<enter>", func(ui.Event) {
		imageName = i.Data
		ui.StopLoop()
	})
	ui.Loop()

	if imageName == "" {
		return nil
	}

	// Split repo:tag
	repo := imageName
	tag := "latest"
	if idx := strings.LastIndex(imageName, ":"); idx > 0 {
		repo = imageName[:idx]
		tag = imageName[idx+1:]
	}

	return Confirm(
		fmt.Sprintf("commit %s as %s:%s?", c.GetMeta("name"), repo, tag),
		func() { c.Commit(repo, tag) },
	)
}

func confirmTxt(a, n string) string { return fmt.Sprintf("%s container %s?", a, n) }

// isAllowedShell validates the shell path against a whitelist of known shells.
var allowedShells = map[string]bool{
	"sh": true, "bash": true, "zsh": true, "fish": true,
	"/bin/sh": true, "/bin/bash": true, "/bin/zsh": true, "/bin/fish": true,
	"/usr/bin/sh": true, "/usr/bin/bash": true, "/usr/bin/zsh": true, "/usr/bin/fish": true,
	"/usr/local/bin/bash": true, "/usr/local/bin/zsh": true, "/usr/local/bin/fish": true,
}

func isAllowedShell(shell string) bool {
	return allowedShells[shell]
}
