package models

import "time"

// DetailRow is one line of a detail view: a name and its value.
type DetailRow struct {
	Key   string
	Value string
}

// DetailSection is a titled group of rows of a detail view.
type DetailSection struct {
	Title string
	Rows  []DetailRow
}

// Details holds what the connector knows about a container beyond its
// metrics, grouped by the panel of the single view that shows it.
type Details struct {
	Container []DetailSection // command, restart policy, labels...
	CPU       []DetailSection // CPU limits
	Mem       []DetailSection // memory limits
	Net       []DetailSection // networks, ports, DNS
	IO        []DetailSection // mounts

	NetworkMode string    // e.g. "bridge", "host", "container:<id>"
	StartedAt   time.Time // zero if unknown
	Running     bool
}
