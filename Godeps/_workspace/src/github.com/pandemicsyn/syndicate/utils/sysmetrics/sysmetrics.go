package sysmetrics

import (
	"fmt"
	"strings"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/pandemicsyn/node_exporter/collector"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	DefaultCollectors = "conntrack,cpu,diskstats,entropy,filefd,filesystem,loadavg,mdadm,meminfo,netdev,netstat,sockstat,stat,textfile,time,uname,version,vmstat"
)

var (
	scrapeDurations = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace: collector.Namespace,
			Subsystem: "exporter",
			Name:      "scrape_duration_seconds",
			Help:      "node_exporter: Duration of a scrape job.",
		},
		[]string{"collector", "result"},
	)
)

func New(collectors map[string]collector.Collector) NodeCollector {
	return NodeCollector{collectors: collectors}
}

// NodeCollector implements the prometheus.Collector interface.
type NodeCollector struct {
	collectors map[string]collector.Collector
}

// Describe implements the prometheus.Collector interface.
func (n NodeCollector) Describe(ch chan<- *prometheus.Desc) {
	scrapeDurations.Describe(ch)
}

// Collect implements the prometheus.Collector interface.
func (n NodeCollector) Collect(ch chan<- prometheus.Metric) {
	wg := sync.WaitGroup{}
	wg.Add(len(n.collectors))
	for name, c := range n.collectors {
		go func(name string, c collector.Collector) {
			execute(name, c, ch)
			wg.Done()
		}(name, c)
	}
	wg.Wait()
	scrapeDurations.Collect(ch)
}

func FilterAvailableCollectors(collectors string) string {
	availableCollectors := make([]string, 0)
	for _, c := range strings.Split(collectors, ",") {
		_, ok := collector.Factories[c]
		if ok {
			availableCollectors = append(availableCollectors, c)
		}
	}
	return strings.Join(availableCollectors, ",")
}

func execute(name string, c collector.Collector, ch chan<- prometheus.Metric) {
	begin := time.Now()
	err := c.Update(ch)
	duration := time.Since(begin)
	var result string

	if err != nil {
		log.Errorf("ERROR: %s collector failed after %fs: %s", name, duration.Seconds(), err)
		result = "error"
	} else {
		log.Debugf("OK: %s collector succeeded after %fs.", name, duration.Seconds())
		result = "success"
	}
	scrapeDurations.WithLabelValues(name, result).Observe(duration.Seconds())
}

func LoadCollectors(list string) (map[string]collector.Collector, error) {
	collectors := map[string]collector.Collector{}
	for _, name := range strings.Split(list, ",") {
		fn, ok := collector.Factories[name]
		if !ok {
			return nil, fmt.Errorf("collector '%s' not available", name)
		}
		c, err := fn()
		if err != nil {
			return nil, err
		}
		collectors[name] = c
	}
	return collectors, nil
}
