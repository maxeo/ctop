package connector

import (
	"strings"
	"testing"
	"time"

	api "github.com/fsouza/go-dockerclient"
	"github.com/maxeo/ctop/models"
)

func find(sections []models.DetailSection, title, key string) string {
	for _, s := range sections {
		if s.Title != title {
			continue
		}
		for _, r := range s.Rows {
			if r.Key == key {
				return r.Value
			}
		}
	}
	return ""
}

func TestDockerDetails(t *testing.T) {
	pids := int64(100)
	insp := &api.Container{
		ID:   "abc",
		Name: "/web",
		Path: "/init",
		Args: []string{"--flag", "two words"},
		State: api.State{
			Status: "running", Running: true, StartedAt: time.Now(),
			Health: api.Health{Status: "healthy", Log: []api.HealthCheck{{ExitCode: 0, Output: " ok \n"}}},
		},
		Config: &api.Config{
			Image:  "nginx",
			Labels: map[string]string{"b": "2", "a": "1", "db_password": "secret"},
		},
		HostConfig: &api.HostConfig{
			NetworkMode:   "bridge",
			RestartPolicy: api.RestartPolicy{Name: "on-failure", MaximumRetryCount: 3},
			NanoCPUs:      1500000000,
			Memory:        512 << 20,
			PidsLimit:     &pids,
		},
		NetworkSettings: &api.NetworkSettings{
			Networks: map[string]api.ContainerNetwork{
				"bridge": {IPAddress: "172.17.0.2", IPPrefixLen: 16, Gateway: "172.17.0.1", MacAddress: "02:42:ac:11:00:02"},
			},
			Ports: map[api.Port][]api.PortBinding{
				"80/tcp":  {{HostIP: "0.0.0.0", HostPort: "8080"}},
				"443/tcp": nil,
			},
		},
		Mounts: []api.Mount{
			{Source: "/srv/data", Destination: "/data", RW: true},
			{Name: "cfg", Source: "/var/lib/docker/volumes/cfg/_data", Destination: "/config", Mode: "z"},
		},
	}
	d := dockerDetails(insp)
	checks := []struct {
		sections         []models.DetailSection
		title, key, want string
	}{
		{d.Container, "Container", "name", "web"},
		{d.Container, "State", "restart policy", "on-failure (max 3 retries)"},
		{d.Container, "Process", "command", `/init --flag "two words"`},
		{d.Container, "Health", "last output", "ok"},
		{d.Container, "Labels", "db_password", "[REDACTED]"},
		{d.CPU, "Limits", "CPUs", "1.5"},
		{d.CPU, "Limits", "pids limit", "100"},
		{d.Mem, "Limits", "limit", "512M"},
		{d.Net, "Network bridge", "IP", "172.17.0.2/16"},
		{d.Net, "Network bridge", "MAC", "02:42:ac:11:00:02"},
		{d.Net, "Ports", "80/tcp", "published on 0.0.0.0:8080"},
		{d.Net, "Ports", "443/tcp", "exposed, not published"},
		{d.IO, "/data", "type", "bind"},
		{d.IO, "/config", "mode", "read-only, z"},
		{d.IO, "/config", "type", "volume cfg"},
	}
	for _, c := range checks {
		if got := find(c.sections, c.title, c.key); got != c.want {
			t.Errorf("%s / %s = %q, want %q", c.title, c.key, got, c.want)
		}
	}
	// labels sorted
	for _, s := range d.Container {
		if s.Title == "Labels" && (s.Rows[0].Key != "a" || s.Rows[1].Key != "b") {
			t.Errorf("labels not sorted: %v", s.Rows)
		}
	}
	if d.NetworkMode != "bridge" || !d.Running {
		t.Errorf("mode %q running %v", d.NetworkMode, d.Running)
	}
}

func TestIPsFormatHost(t *testing.T) {
	got := ipsFormat(map[string]api.ContainerNetwork{"host": {}, "b": {IPAddress: "1.2.3.4"}})
	if got != "b:1.2.3.4\nhost" {
		t.Fatalf("got %q", strings.ReplaceAll(got, "\n", `\n`))
	}
}
