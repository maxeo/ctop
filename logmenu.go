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
// screen, <home> shows the start of the log, <end> the newest lines, [t]
// toggles timestamps, Ctrl+F or [/] searches ([n] older match, [N] newer,
// <escape> ends the search), a click or [s] switches to text selection
// (<escape> or [s] goes back), anything else
// closes. Home/End also work in terminals whose sequences termbox doesn't
// decode.
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
		m.SetOnBottom(history.LoadNext)
	}
	ui.Render(m)

	m.BeforeRender(func() { tm.SetInputMode(logInputNormal) })
	defer func() {
		// turn the mouse off once the view no longer draws (see BeforeRender)
		stop()
		select {
		case <-m.Stopped():
		case <-time.After(time.Second):
		}
		tm.SetInputMode(tm.InputAlt)
	}()

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

	lv := &logView{view: m, history: history, close: closeView}
	lv.showFooter("")

	ui.Handle("/sys/wnd/resize", func(e ui.Event) {
		m.Resize()
	})
	ui.Handle("/sys/mouse", func(e ui.Event) {
		if me, ok := e.Data.(ui.EvtMouse); ok {
			lv.mouse(me.Press)
		}
	})
	// one handler for every key: the meaning of a key depends on the mode
	// (e.g. "t" is typed into the search prompt)
	ui.Handle("/sys/kbd/", func(e ui.Event) {
		lv.key(strings.TrimPrefix(e.Path, "/sys/kbd/"))
	})
	ui.Handle(escEvent, func(e ui.Event) {
		if id, ok := e.Data.(int); ok {
			lv.escTimeout(id)
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
		return fmt.Sprintf("%s %s", t.timestamp.Local().Format("2006-01-02 15:04:05.000"), t.message)
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
