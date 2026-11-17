package container

import (
	"sync"

	"github.com/maxeo/ctop/connector/collector"
	"github.com/maxeo/ctop/connector/manager"
	"github.com/maxeo/ctop/cwidgets"
	"github.com/maxeo/ctop/cwidgets/compact"
	"github.com/maxeo/ctop/logging"
	"github.com/maxeo/ctop/models"
)

var (
	log = logging.Init()
)

const (
	running = "running"
)

// Metrics and metadata representing a container
type Container struct {
	models.Metrics
	Id        string
	Meta      models.Meta
	Widgets   *compact.CompactRow
	Display   bool // display this container in compact view
	updater   cwidgets.WidgetUpdater
	collector collector.Collector
	manager   manager.Manager

	detailsMu sync.Mutex
	details   *models.Details
}

// DetailsUpdater is a widget that also shows the container details.
type DetailsUpdater interface {
	SetDetails(models.Details)
}

// SetDetails stores what the connector knows about the container beyond
// its metrics, and passes it to the widget if it shows it.
func (c *Container) SetDetails(d models.Details) {
	c.detailsMu.Lock()
	c.details = &d
	u := c.updater
	c.detailsMu.Unlock()
	if du, ok := u.(DetailsUpdater); ok {
		du.SetDetails(d)
	}
}

func New(id string, collector collector.Collector, manager manager.Manager) *Container {
	widgets := compact.NewCompactRow()
	shortID := id
	if len(shortID) > 12 {
		shortID = shortID[0:12]
	}
	return &Container{
		Metrics:   models.NewMetrics(),
		Id:        id,
		Meta:      models.NewMeta("id", shortID),
		Widgets:   widgets,
		updater:   widgets,
		collector: collector,
		manager:   manager,
	}
}

func (c *Container) RecreateWidgets() {
	c.SetUpdater(cwidgets.NullWidgetUpdater{})
	c.Widgets = compact.NewCompactRow()
	c.SetUpdater(c.Widgets)
}

func (c *Container) SetUpdater(u cwidgets.WidgetUpdater) {
	c.detailsMu.Lock()
	c.updater = u
	d := c.details
	c.detailsMu.Unlock()
	c.updater.SetMeta(c.Meta)
	if du, ok := u.(DetailsUpdater); ok && d != nil {
		du.SetDetails(*d)
	}
}

func (c *Container) SetMeta(k, v string) {
	c.Meta[k] = v
	c.updater.SetMeta(c.Meta)
}

func (c *Container) GetMeta(k string) string {
	return c.Meta.Get(k)
}

func (c *Container) SetState(s string) {
	c.SetMeta("state", s)
	// start collector, if needed
	if s == running && !c.collector.Running() {
		c.collector.Start()
		c.Read(c.collector.Stream())
	}
	// stop collector, if needed
	if s != running && c.collector.Running() {
		c.collector.Stop()
	}
}

// Logs returns container log collector
func (c *Container) Logs() collector.LogCollector {
	return c.collector.Logs()
}

// Read metric stream, updating widgets
func (c *Container) Read(stream chan models.Metrics) {
	go func() {
		for metrics := range stream {
			c.Metrics = metrics
			c.updater.SetMetrics(metrics)
		}
		log.Infof("reader stopped for container: %s", c.Id)
		c.Metrics = models.NewMetrics()
		c.Widgets.Reset()
	}()
	log.Infof("reader started for container: %s", c.Id)
}

func (c *Container) Start() {
	if c.Meta["state"] != running {
		if err := c.manager.Start(); err != nil {
			log.Warningf("container %s: %v", c.Id, err)
			log.StatusErr(err)
			return
		}
		c.SetState(running)
	}
}

func (c *Container) Stop() {
	if c.Meta["state"] == running {
		if err := c.manager.Stop(); err != nil {
			log.Warningf("container %s: %v", c.Id, err)
			log.StatusErr(err)
			return
		}
		c.SetState("exited")
	}
}

func (c *Container) Remove() {
	if err := c.manager.Remove(); err != nil {
		log.Warningf("container %s: %v", c.Id, err)
		log.StatusErr(err)
	}
}

func (c *Container) Pause() {
	if c.Meta["state"] == running {
		if err := c.manager.Pause(); err != nil {
			log.Warningf("container %s: %v", c.Id, err)
			log.StatusErr(err)
			return
		}
		c.SetState("paused")
	}
}

func (c *Container) Unpause() {
	if c.Meta["state"] == "paused" {
		if err := c.manager.Unpause(); err != nil {
			log.Warningf("container %s: %v", c.Id, err)
			log.StatusErr(err)
			return
		}
		c.SetState(running)
	}
}

func (c *Container) Restart() {
	if c.Meta["state"] == running {
		if err := c.manager.Restart(); err != nil {
			log.Warningf("container %s: %v", c.Id, err)
			log.StatusErr(err)
			return
		}
	}
}

func (c *Container) Commit(repo, tag string) {
	if err := c.manager.Commit(repo, tag); err != nil {
		log.Warningf("container %s: %v", c.Id, err)
		log.StatusErr(err)
		return
	}
	log.Statusf("committed %s as %s:%s", c.GetMeta("name"), repo, tag)
}

func (c *Container) StopAndRemove() {
	if c.Meta["state"] == running {
		if err := c.manager.Stop(); err != nil {
			log.Warningf("container %s: %v", c.Id, err)
			log.StatusErr(err)
			return
		}
		c.SetState("exited")
	}
	if err := c.manager.Remove(); err != nil {
		log.Warningf("container %s: %v", c.Id, err)
		log.StatusErr(err)
	}
}

func (c *Container) Exec(cmd []string) error {
	return c.manager.Exec(cmd)
}
