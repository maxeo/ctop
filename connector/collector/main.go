package collector

import (
	"math"
	"time"

	"github.com/eqms/ctop/logging"
	"github.com/eqms/ctop/models"
)

var log = logging.Init()

type LogCollector interface {
	Stream() chan models.Log
	Stop()
}

// LogHistory is implemented by log collectors that can load older lines on
// demand, so the log view can page back through history.
type LogHistory interface {
	// Older returns up to one page of lines logged before the given time,
	// oldest first. skip is the number of lines already loaded. more is
	// false once the start of the log has been reached.
	Older(before time.Time, skip int) (logs []models.Log, more bool, err error)
	// AllBefore returns every line logged before the given time, oldest first.
	AllBefore(before time.Time) ([]models.Log, error)
	// Newer returns up to one page of lines logged after the given time,
	// oldest first; a zero time means from the start of the log. more is
	// false once the end of the log (at the time of the call) is reached.
	Newer(after time.Time) (logs []models.Log, more bool, err error)
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
