package container

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/eqms/ctop/config"
)

type sortMethod func(c1, c2 *Container) bool

var stateMap = map[string]int{
	"running": 3,
	"paused":  2,
	"exited":  1,
	"created": 0,
	"":        0,
}

var idSorter = func(c1, c2 *Container) bool { return c1.Id < c2.Id }
var nameSorter = func(c1, c2 *Container) bool { return c1.GetMeta("name") < c2.GetMeta("name") }

var Sorters = map[string]sortMethod{
	"id":   idSorter,
	"name": nameSorter,
	"cpu": func(c1, c2 *Container) bool {
		// Use secondary sort method if equal values
		if c1.CPUUtil == c2.CPUUtil {
			return nameSorter(c1, c2)
		}
		return c1.CPUUtil > c2.CPUUtil
	},
	"mem": func(c1, c2 *Container) bool {
		// Use secondary sort method if equal values
		if c1.MemUsage == c2.MemUsage {
			return nameSorter(c1, c2)
		}
		return c1.MemUsage > c2.MemUsage
	},
	"mem %": func(c1, c2 *Container) bool {
		// Use secondary sort method if equal values
		if c1.MemPercent == c2.MemPercent {
			return nameSorter(c1, c2)
		}
		return c1.MemPercent > c2.MemPercent
	},
	"net": func(c1, c2 *Container) bool {
		sum1 := sumNet(c1)
		sum2 := sumNet(c2)
		// Use secondary sort method if equal values
		if sum1 == sum2 {
			return nameSorter(c1, c2)
		}
		return sum1 > sum2
	},
	"pids": func(c1, c2 *Container) bool {
		// Use secondary sort method if equal values
		if c1.Pids == c2.Pids {
			return nameSorter(c1, c2)
		}
		return c1.Pids > c2.Pids
	},
	"io": func(c1, c2 *Container) bool {
		sum1 := sumIO(c1)
		sum2 := sumIO(c2)
		// Use secondary sort method if equal values
		if sum1 == sum2 {
			return nameSorter(c1, c2)
		}
		return sum1 > sum2
	},
	"state": func(c1, c2 *Container) bool {
		// Use secondary sort method if equal values
		c1state := c1.GetMeta("state")
		c2state := c2.GetMeta("state")
		if c1state == c2state {
			return nameSorter(c1, c2)
		}
		return stateMap[c1state] > stateMap[c2state]
	},
	"uptime": func(c1, c2 *Container) bool {
		// Use secondary sort method if equal values
		c1Uptime := uptimeSeconds(c1.GetMeta("uptime"))
		c2Uptime := uptimeSeconds(c2.GetMeta("uptime"))
		if c1Uptime == c2Uptime {
			return nameSorter(c1, c2)
		}
		return c1Uptime > c2Uptime
	},
}

// seconds of each unit durafmt writes
var uptimeUnits = map[string]float64{
	"microsecond": 1e-6,
	"millisecond": 1e-3,
	"second":      1,
	"minute":      60,
	"hour":        3600,
	"day":         86400,
	"week":        7 * 86400,
	"year":        365 * 86400,
}

// uptimeSeconds converts an uptime as shown ("2 hours", "25 weeks") to
// seconds, so that "3 weeks" sorts above "2 hours" (as text it sorted below
// "25 weeks"). Unknown text gives -1.
func uptimeSeconds(s string) float64 {
	f := strings.Fields(s)
	if len(f) < 2 {
		return -1
	}
	n, err := strconv.ParseFloat(f[0], 64)
	if err != nil {
		return -1
	}
	unit, ok := uptimeUnits[strings.TrimSuffix(f[1], "s")]
	if !ok {
		return -1
	}
	return n * unit
}

func SortFields() (fields []string) {
	for k := range Sorters {
		fields = append(fields, k)
	}
	return fields
}

type Containers []*Container

func (a Containers) Sort()         { sort.Sort(a) }
func (a Containers) Len() int      { return len(a) }
func (a Containers) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a Containers) Less(i, j int) bool {
	f := Sorters[config.GetVal("sortField")]
	if config.GetSwitchVal("sortReversed") {
		return f(a[j], a[i])
	}
	return f(a[i], a[j])
}

func (a Containers) Filter() {
	filter := config.GetVal("filterStr")
	re := regexp.MustCompile(fmt.Sprintf(".*%s", filter))

	for _, c := range a {
		c.Display = true
		// Apply name filter
		if re.FindAllString(c.GetMeta("name"), 1) == nil {
			c.Display = false
		}
		// Apply state filter
		if !config.GetSwitchVal("allContainers") && c.GetMeta("state") != "running" {
			c.Display = false
		}
	}
}

func sumNet(c *Container) int64 { return c.NetRx + c.NetTx }

func sumIO(c *Container) int64 { return c.IOBytesRead + c.IOBytesWrite }
