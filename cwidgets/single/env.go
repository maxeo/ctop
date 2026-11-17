package single

import (
	"regexp"
	"sort"
	"strings"

	"github.com/maxeo/ctop/models"
	"github.com/maxeo/ctop/redact"
)

var envPattern = regexp.MustCompile(`(?P<KEY>[^=]+)=(?P<VALUE>.*)`)

// Env lists the environment variables of the container, sorted by name.
type Env struct {
	*KVList
}

func NewEnv() *Env {
	l := NewKVList("Env")
	l.Empty = "no environment variables"
	return &Env{l}
}

// Set takes the variables as "KEY=value;KEY=value".
func (w *Env) Set(allEnvs string) {
	var vars []models.DetailRow
	for _, env := range strings.Split(allEnvs, ";") {
		if m := envPattern.FindStringSubmatch(env); len(m) == 3 {
			vars = append(vars, models.DetailRow{Key: m[1], Value: redact.Env(m[1], m[2])})
		}
	}
	sort.SliceStable(vars, func(i, j int) bool { return naturalLess(vars[i].Key, vars[j].Key) })
	w.SetSections([]models.DetailSection{{Rows: vars}})
}

// naturalLess compares a and b ignoring case, with the numbers in them
// compared by value (VAR_2 before VAR_10).
func naturalLess(a, b string) bool {
	a, b = strings.ToLower(a), strings.ToLower(b)
	for a != "" && b != "" {
		na, ra := leadingNumber(a)
		nb, rb := leadingNumber(b)
		if na != "" && nb != "" {
			if len(na) != len(nb) {
				return len(na) < len(nb)
			}
			if na != nb {
				return na < nb
			}
			a, b = ra, rb
			continue
		}
		if a[0] != b[0] {
			return a[0] < b[0]
		}
		a, b = a[1:], b[1:]
	}
	return len(a) < len(b)
}

// leadingNumber splits the digits at the start of s (without leading
// zeros) from the rest.
func leadingNumber(s string) (num, rest string) {
	i := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	num = strings.TrimLeft(s[:i], "0")
	if i > 0 && num == "" {
		num = "0"
	}
	return num, s[i:]
}
