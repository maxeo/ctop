package collector

import (
	"math"

	"github.com/maxeo/ctop/logging"
	"github.com/maxeo/ctop/models"
)

var log = logging.Init()

type LogCollector interface {
	Stream() chan models.Log
	Stop()
}

// LogHistory is implemented by log collectors that can load older lines on
// demand, so the log view can page back through history. None of the methods
// keeps more than a page of lines in memory, whatever the size of the log.
type LogHistory interface {
	// Older returns up to one page of lines logged before the cursor,
	// oldest first. skip is the number of lines loaded from the cursor to
	// the tail of the log, or -1 if unknown. more is false once the start of
	// the log has been reached.
	Older(before LogCursor, skip int) (logs []models.Log, more bool, err error)
	// Newer returns up to one page of lines logged after the cursor, oldest
	// first; a zero cursor means from the start of the log. more is false
	// once the end of the log (at the time of the call) is reached.
	Newer(after LogCursor) (logs []models.Log, more bool, err error)
	// FindOlder returns the newest line before the cursor whose message
	// matches, with the lines around it, oldest first.
	FindOlder(before LogCursor, match func(string) bool) (logs []models.Log, found bool, err error)
	// FindNewer returns the oldest line after the cursor whose message
	// matches, with the lines around it, oldest first.
	FindNewer(after LogCursor, match func(string) bool) (logs []models.Log, found bool, err error)
}

type Collector interface {
	Stream() chan models.Metrics
	Logs() LogCollector
	Running() bool
	Start()
	Stop()
}

func round(num float64) int {
	return int(num + math.Copysign(0.5, num))
}

// return rounded percentage
func percent(val float64, total float64) int {
	if total <= 0 {
		return 0
	}
	return round((val / total) * 100)
}
