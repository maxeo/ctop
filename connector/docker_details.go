package connector

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	api "github.com/fsouza/go-dockerclient"
	"github.com/maxeo/ctop/cwidgets"
	"github.com/maxeo/ctop/models"
	"github.com/maxeo/ctop/redact"
)

// dockerDetails collects what the single view shows in its detail views,
// from the inspect of a container.
func dockerDetails(insp *api.Container) models.Details {
	d := models.Details{
		StartedAt: insp.State.StartedAt,
		Running:   insp.State.Running,
	}
	cfg := insp.Config
	if cfg == nil {
		cfg = &api.Config{}
	}
	host := insp.HostConfig
	if host == nil {
		host = &api.HostConfig{}
	}
	d.NetworkMode = host.NetworkMode

	// --- container ----------------------------------------------------------

	general := rows(
		"name", shortName(insp.Name),
		"id", insp.ID,
		"image", cfg.Image,
		"image id", insp.Image,
		"platform", insp.Platform,
		"created", fmtTime(insp.Created),
		"hostname", joinNonEmpty(".", cfg.Hostname, cfg.Domainname),
	)
	state := rows(
		"status", insp.State.Status,
		"started", fmtTime(insp.State.StartedAt),
	)
	if !insp.State.Running {
		state = append(state, rows(
			"finished", fmtTime(insp.State.FinishedAt),
			"exit code", strconv.Itoa(insp.State.ExitCode),
		)...)
	}
	if insp.State.Pid > 0 {
		state = append(state, models.DetailRow{Key: "pid on host", Value: strconv.Itoa(insp.State.Pid)})
	}
	if insp.State.OOMKilled {
		state = append(state, models.DetailRow{Key: "OOM killed", Value: "yes"})
	}
	state = append(state, rows("error", insp.State.Error)...)
	restart := host.RestartPolicy.Name
	if restart == "" {
		restart = "no"
	}
	if host.RestartPolicy.MaximumRetryCount > 0 {
		restart += fmt.Sprintf(" (max %d retries)", host.RestartPolicy.MaximumRetryCount)
	}
	state = append(state, rows(
		"restart policy", restart,
		"restarts", strconv.Itoa(insp.RestartCount),
	)...)

	command := quoteArgs(append([]string{insp.Path}, insp.Args...))
	cmd := quoteArgs(cfg.Cmd)
	if cmd == command {
		cmd = "" // no entrypoint: the same line twice
	}
	process := rows(
		"command", command,
		"entrypoint", quoteArgs(cfg.Entrypoint),
		"cmd", cmd,
		"working dir", cfg.WorkingDir,
		"user", cfg.User,
		"stop signal", cfg.StopSignal,
	)
	if cfg.Tty {
		process = append(process, models.DetailRow{Key: "tty", Value: "yes"})
	}
	if host.Init {
		process = append(process, models.DetailRow{Key: "init", Value: "yes"})
	}

	security := rows(
		"privileged", yesNo(host.Privileged),
		"read-only rootfs", yesNo(host.ReadonlyRootfs),
		"cap add", strings.Join(host.CapAdd, ", "),
		"cap drop", strings.Join(host.CapDrop, ", "),
	)

	health := healthRows(insp, cfg)

	logs := rows("driver", host.LogConfig.Type, "path", insp.LogPath)
	for _, k := range sortedKeys(host.LogConfig.Config) {
		logs = append(logs, models.DetailRow{Key: k, Value: host.LogConfig.Config[k]})
	}

	var labels []models.DetailRow
	for _, k := range sortedKeys(cfg.Labels) {
		labels = append(labels, models.DetailRow{Key: k, Value: redact.Env(k, cfg.Labels[k])})
	}

	d.Container = sections(
		"Container", general,
		"State", state,
		"Process", process,
		"Health", health,
		"Security", security,
		"Logging", logs,
		"Labels", labels,
	)

	// --- CPU and memory limits -------------------------------------------

	cpu := rows(
		"CPUs", cpusLimit(host.NanoCPUs, host.CPUQuota, host.CPUPeriod),
		"CPU set", host.CPUSetCPUs,
	)
	if host.CPUShares > 0 {
		cpu = append(cpu, models.DetailRow{Key: "CPU shares", Value: strconv.FormatInt(host.CPUShares, 10)})
	}
	if host.PidsLimit != nil && *host.PidsLimit > 0 {
		cpu = append(cpu, models.DetailRow{Key: "pids limit", Value: strconv.FormatInt(*host.PidsLimit, 10)})
	}
	d.CPU = sections("Limits", cpu)

	mem := rows(
		"limit", bytesOrNone(host.Memory),
		"reservation", bytesOrNone(host.MemoryReservation),
	)
	switch {
	case host.MemorySwap < 0:
		mem = append(mem, models.DetailRow{Key: "memory + swap", Value: "unlimited"})
	case host.MemorySwap > 0:
		mem = append(mem, models.DetailRow{Key: "memory + swap", Value: cwidgets.ByteFormat64Short(host.MemorySwap)})
	}
	if host.MemorySwappiness != nil && *host.MemorySwappiness >= 0 {
		mem = append(mem, models.DetailRow{Key: "swappiness", Value: strconv.FormatInt(*host.MemorySwappiness, 10)})
	}
	d.Mem = sections("Limits", mem)

	// --- network ------------------------------------------------------------

	var nets []any
	nets = append(nets, "Network", rows("mode", host.NetworkMode))
	if ns := insp.NetworkSettings; ns != nil {
		for _, name := range sortedKeys(ns.Networks) {
			n := ns.Networks[name]
			ip := n.IPAddress
			if ip != "" && n.IPPrefixLen > 0 {
				ip += "/" + strconv.Itoa(n.IPPrefixLen)
			}
			ip6 := n.GlobalIPv6Address
			if ip6 != "" && n.GlobalIPv6PrefixLen > 0 {
				ip6 += "/" + strconv.Itoa(n.GlobalIPv6PrefixLen)
			}
			r := rows(
				"IP", ip,
				"gateway", n.Gateway,
				"MAC", n.MacAddress,
				"IPv6", ip6,
				"IPv6 gateway", n.IPv6Gateway,
				"aliases", strings.Join(n.Aliases, ", "),
				"DNS names", strings.Join(n.DNSNames, ", "),
			)
			if len(r) == 0 {
				r = rows("IP", "the host's")
			}
			nets = append(nets, "Network "+name, r)
		}
		nets = append(nets, "Ports", portRows(ns.Ports))
	}
	dns := rows(
		"DNS servers", strings.Join(host.DNS, ", "),
		"extra hosts", strings.Join(host.ExtraHosts, ", "),
	)
	nets = append(nets, "DNS", dns)
	d.Net = sections(nets...)

	// --- mounts -------------------------------------------------------------

	var mounts []any
	for _, m := range insp.Mounts {
		kind := "bind"
		if m.Name != "" {
			kind = "volume " + m.Name
			if m.Driver != "" && m.Driver != "local" {
				kind += " (" + m.Driver + ")"
			}
		}
		mode := "read-write"
		if !m.RW {
			mode = "read-only"
		}
		if m.Mode != "" && m.Mode != "rw" && m.Mode != "ro" {
			mode += ", " + m.Mode
		}
		mounts = append(mounts, m.Destination, rows(
			"source", m.Source,
			"type", kind,
			"mode", mode,
		))
	}
	if len(mounts) == 0 {
		mounts = append(mounts, "Mounts", rows("mounts", "none"))
	}
	d.IO = sections(mounts...)

	return d
}

func healthRows(insp *api.Container, cfg *api.Config) []models.DetailRow {
	h := insp.State.Health
	var r []models.DetailRow
	if hc := cfg.Healthcheck; hc != nil && len(hc.Test) > 0 && hc.Test[0] != "NONE" {
		test := hc.Test
		if test[0] == "CMD" || test[0] == "CMD-SHELL" {
			test = test[1:]
		}
		r = append(r, rows(
			"check", quoteArgs(test),
			"interval", durOrEmpty(hc.Interval),
			"timeout", durOrEmpty(hc.Timeout),
		)...)
		if hc.Retries > 0 {
			r = append(r, models.DetailRow{Key: "retries", Value: strconv.Itoa(hc.Retries)})
		}
	}
	if h.Status == "" {
		if len(r) == 0 {
			return rows("status", "N/A (no healthcheck)")
		}
		return r
	}
	r = append(rows("status", h.Status), r...)
	if h.FailingStreak > 0 {
		r = append(r, models.DetailRow{Key: "failing streak", Value: strconv.Itoa(h.FailingStreak)})
	}
	if n := len(h.Log); n > 0 {
		last := h.Log[n-1]
		r = append(r, rows(
			"last check", fmtTime(last.End),
			"last exit code", strconv.Itoa(last.ExitCode),
			"last output", strings.TrimSpace(last.Output),
		)...)
	}
	return r
}

func portRows(ports map[api.Port][]api.PortBinding) []models.DetailRow {
	var r []models.DetailRow
	keys := make([]string, 0, len(ports))
	for p := range ports {
		keys = append(keys, string(p))
	}
	sort.Strings(keys)
	for _, k := range keys {
		bindings := ports[api.Port(k)]
		if len(bindings) == 0 {
			r = append(r, models.DetailRow{Key: k, Value: "exposed, not published"})
			continue
		}
		var to []string
		for _, b := range bindings {
			hostIP := b.HostIP
			if hostIP == "" {
				hostIP = "0.0.0.0"
			}
			to = append(to, hostIP+":"+b.HostPort)
		}
		r = append(r, models.DetailRow{Key: k, Value: "published on " + strings.Join(to, ", ")})
	}
	if len(r) == 0 {
		r = rows("ports", "none")
	}
	return r
}

// rows makes detail rows from key, value pairs, leaving out empty values.
func rows(kv ...string) []models.DetailRow {
	var r []models.DetailRow
	for i := 0; i+1 < len(kv); i += 2 {
		if v := strings.TrimSpace(kv[i+1]); v != "" {
			r = append(r, models.DetailRow{Key: kv[i], Value: v})
		}
	}
	return r
}

// sections makes detail sections from title, rows pairs, leaving out empty
// sections.
func sections(tr ...any) []models.DetailSection {
	var s []models.DetailSection
	for i := 0; i+1 < len(tr); i += 2 {
		title, _ := tr[i].(string)
		r, _ := tr[i+1].([]models.DetailRow)
		if len(r) > 0 {
			s = append(s, models.DetailSection{Title: title, Rows: r})
		}
	}
	return s
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func fmtTime(t time.Time) string {
	if t.IsZero() || t.Year() < 2000 {
		return ""
	}
	return t.Local().Format("2006-01-02 15:04:05")
}

func durOrEmpty(d time.Duration) string {
	if d <= 0 {
		return ""
	}
	return d.String()
}

func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return ""
}

func joinNonEmpty(sep string, parts ...string) string {
	var out []string
	for _, p := range parts {
		if p != "" {
			out = append(out, p)
		}
	}
	return strings.Join(out, sep)
}

// quoteArgs joins a command line, quoting the arguments with spaces.
func quoteArgs(args []string) string {
	var out []string
	for _, a := range args {
		if a == "" {
			continue
		}
		if strings.ContainsAny(a, " \t\"'") {
			a = strconv.Quote(a)
		}
		out = append(out, a)
	}
	return strings.Join(out, " ")
}

func bytesOrNone(n int64) string {
	if n <= 0 {
		return "none"
	}
	return cwidgets.ByteFormat64Short(n)
}

// cpusLimit describes the CPU limit set with --cpus (NanoCPUs) or with a
// quota and period.
func cpusLimit(nano, quota, period int64) string {
	switch {
	case nano > 0:
		return strconv.FormatFloat(float64(nano)/1e9, 'f', -1, 64)
	case quota > 0 && period > 0:
		return strconv.FormatFloat(float64(quota)/float64(period), 'f', 2, 64) + " (quota)"
	}
	return "no limit"
}
