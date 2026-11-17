package main

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/eqms/ctop/connector/collector"
	"github.com/eqms/ctop/widgets"
	ui "github.com/gizak/termui"
	tm "github.com/nsf/termbox-go"
)

// lines scrolled per mouse wheel notch in the log view
const logWheelStep = 3

// LogMenu shows the selected container's logs.
//
// Keys: wheel / <up> <down> scroll, <previous> <next> (PgUp/PgDn) move one
// screen, <home> jumps to the start of the log (loading all of it), <end>
// back to the newest line, [t] toggles timestamps, anything else closes.
// Home/End also work in terminals whose sequences termbox doesn't decode.
func LogMenu() MenuFn {

	c := cursor.Selected()
	if c == nil {
		return nil
	}

	ui.DefaultEvtStream.ResetHandlers()
	defer ui.DefaultEvtStream.ResetHandlers()

	logCollector := c.Logs()
	logs, stop := logReader(logCollector)
	defer stop()
	label := fmt.Sprintf("Logs [%s]", c.GetMeta("name"))
	m := widgets.NewTextView(logs)
	m.BorderLabel = label

	var history *logHistory
	if h, ok := logCollector.(collector.LogHistory); ok {
		history = newLogHistory(m, h, label)
		m.SetOnTop(history.LoadPage)
	}
	ui.Render(m)

	ui.Handle("/sys/wnd/resize", func(e ui.Event) {
		m.Resize()
	})
	ui.Handle("/sys/kbd/t", func(ui.Event) {
		m.Toggle()
	})
	// Enable mouse reporting only here, so the wheel scrolls the log in any
	// terminal (without it, many terminals scroll their own scrollback
	// instead). Down stops at the newest line, where the view keeps
	// following new output. Other mouse buttons are ignored.
	tm.SetInputMode(tm.InputAlt | tm.InputMouse)
	defer tm.SetInputMode(tm.InputAlt)
	ui.Handle("/sys/mouse", func(e ui.Event) {
		me, ok := e.Data.(ui.EvtMouse)
		if !ok {
			return
		}
		switch me.Press {
		case "MouseWheelUp":
			m.ScrollUp(logWheelStep)
		case "MouseWheelDown":
			m.ScrollDown(logWheelStep)
		}
	})
	// Terminals that turn the wheel into <up>/<down> (alternate scroll mode)
	// land here too.
	HandleKeys("up", func() { m.ScrollUp(1) })
	HandleKeys("down", func() { m.ScrollDown(1) })
	HandleKeys("pgup", func() { m.ScrollUp(m.PageSize()) })
	HandleKeys("pgdown", func() { m.ScrollDown(m.PageSize()) })
	home := func() {
		if history != nil {
			go history.LoadAll()
		} else {
			m.ScrollToTop()
		}
	}
	ui.Handle("/sys/kbd/<home>", func(ui.Event) { home() })
	ui.Handle("/sys/kbd/<end>", func(ui.Event) { m.ScrollToBottom() })
	var csi csiKeys
	ui.Handle("/sys/kbd/M-[", func(ui.Event) { csi.Start() })
	ui.Handle("/sys/kbd/M-O", func(ui.Event) { csi.Start() })
	// Close at most once: several key events can arrive before the loop
	// exits, and a second StopLoop would reach the container list's loop,
	// which treats it as "quit ctop".
	var closeOnce sync.Once
	closeView := func() {
		closeOnce.Do(func() {
			stop()
			ui.StopLoop()
		})
	}
	// Any other key closes the view, except the rest of a Home/End escape
	// sequence (see csiKeys).
	ui.Handle("/sys/kbd/", func(e ui.Event) {
		action, consumed := csi.Feed(strings.TrimPrefix(e.Path, "/sys/kbd/"))
		switch {
		case action == "home":
			home()
		case action == "end":
			m.ScrollToBottom()
		case !consumed:
			closeView()
		}
	})
	ui.Loop()
	return nil
}

type toggleLog struct {
	timestamp time.Time
	message   string
}

func (t *toggleLog) Toggle(on bool) string {
	if on {
		return fmt.Sprintf("%s %s", t.timestamp.Format("2006-01-02T15:04:05.999Z07:00"), t.message)
	}
	return t.message
}

// logReader streams the container logs. stop is safe to call any number of
// times; the first call stops the collector and closes logs.
func logReader(logCollector collector.LogCollector) (logs chan widgets.ToggleText, stop func()) {

	stream := logCollector.Stream()
	logs = make(chan widgets.ToggleText)
	quit := make(chan struct{})

	var once sync.Once
	stop = func() { once.Do(func() { close(quit) }) }

	go func() {
		defer close(logs)
		defer logCollector.Stop()
		for {
			select {
			case log, ok := <-stream:
				if !ok {
					return
				}
				select {
				case logs <- &toggleLog{timestamp: log.Timestamp, message: log.Message}:
				case <-quit:
					return
				}
			case <-quit:
				return
			}
		}
	}()
	return
}
