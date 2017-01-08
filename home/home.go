// Package home can extract metrics from FritzBox home automation devices.
package home

import (
	"log"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	varLabels = []string{
		"host",
		"module",
	}

	tempDesc = prometheus.NewDesc(
		"fritzbox_home_temperature_celsius",
		"Current temperature measurement of home automation device in celsius.",
		varLabels, nil)
)

// NewCollector creates a new collector for the specified host.
func NewCollector(hostname string, password string) prometheus.Collector {
	labels := prometheus.Labels{
		"host": hostname,
	}

	return &homeCollector{
		Hostname: hostname,
		Password: password,
		UpMetric: prometheus.NewGauge(prometheus.GaugeOpts{
			Name:        "fritzbox_home_up",
			Help:        "Indicates if the last scrape to the FritzBox was successful.",
			ConstLabels: labels,
		}),
		AuthMetric: prometheus.NewGauge(prometheus.GaugeOpts{
			Name:        "fritzbox_home_authenticated",
			Help:        "Indicates if the authentication to the FritzBox is successful.",
			ConstLabels: labels,
		}),
		Client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

type homeCollector struct {
	Hostname string
	Password string

	Sid          string
	SidTimestamp time.Time
	UpMetric     prometheus.Gauge
	AuthMetric   prometheus.Gauge
	Client       *http.Client
}

func (c *homeCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.UpMetric.Desc()
	ch <- c.AuthMetric.Desc()
	ch <- tempDesc
}

func (c *homeCollector) Collect(ch chan<- prometheus.Metric) {
	if !c.sidValid() {
		sid, err := c.authenticate()
		if err != nil {
			log.Printf("Error during authentication: %s", err)

			c.AuthMetric.Set(0)
			ch <- c.AuthMetric

			return
		}
		c.Sid = sid
		c.SidTimestamp = time.Now()
		c.AuthMetric.Set(1)
	}
	ch <- c.AuthMetric
	ch <- c.UpMetric
}
