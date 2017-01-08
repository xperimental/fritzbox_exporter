// Package home can extract metrics from FritzBox home automation devices.
package home

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	htmlquery "github.com/antchfx/xquery/html"
	"golang.org/x/net/html"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	homeURLFormat = "http://%s/net/home_auto_overview.lua?sid=%s&update=uiSmarthomeTables&view=&xhr=1"
)

var (
	varLabels = []string{
		"host",
		"module",
	}

	tempDesc = prometheus.NewDesc(
		"fritzbox_home_current_temperature_celsius",
		"Current temperature measurement of home automation device in celsius.",
		varLabels, nil)
	targetTempDesc = prometheus.NewDesc(
		"fritzbox_home_target_temperature_celsius",
		"Target temperature setting of home automation device in celsius.",
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
	ch <- targetTempDesc
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

	home, err := c.getHomeData()
	if err != nil {
		log.Printf("Error getting home data: %s", err)

		c.UpMetric.Set(0)
		ch <- c.UpMetric
		return
	}
	c.UpMetric.Set(1)
	ch <- c.UpMetric

	for _, th := range home.Thermostats {
		labels := []string{
			c.Hostname,
			th.Name,
		}

		sendMetric(ch, tempDesc, th.CurrentTemperature, labels)
		sendMetric(ch, targetTempDesc, th.TargetTemperature, labels)
	}
}

func sendMetric(ch chan<- prometheus.Metric, desc *prometheus.Desc, value float64, labels []string) {
	m, err := prometheus.NewConstMetric(desc, prometheus.GaugeValue, value, labels...)
	if err != nil {
		log.Printf("Error creating metric %s: %s", tempDesc.String(), err)
		return
	}
	ch <- m
}

type homeData struct {
	Thermostats []thermostat
}

type thermostat struct {
	Name               string
	CurrentTemperature float64
	TargetTemperature  float64
}

func (c *homeCollector) getHomeData() (homeData, error) {
	var result homeData
	url := fmt.Sprintf(homeURLFormat, c.Hostname, c.Sid)
	res, err := c.Client.Get(url)
	if err != nil {
		return result, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return result, fmt.Errorf("invalid status code: %d", res.StatusCode)
	}

	if err := parseHomeData(&result, res.Body); err != nil {
		return result, err
	}

	return result, nil
}

func parseHomeData(result *homeData, r io.Reader) error {
	contextNode := &html.Node{
		Type: html.ElementNode,
	}
	nodes, err := html.ParseFragment(r, contextNode)
	if err != nil {
		return err
	}

	for _, node := range nodes {
		if node.Type == html.ElementNode && node.Data == "table" {
			return parseTableNode(result, node)
		}
	}

	return errors.New("no table node found")
}

func parseTableNode(result *homeData, table *html.Node) error {
	items := htmlquery.Find(table, "//tr")
itemLoop:
	for _, item := range items {
		for _, a := range item.Attr {
			if a.Key == "class" && a.Val == "thead" {
				continue itemLoop
			}
		}

		nameNode := htmlquery.FindOne(item, "//td[@class='name cut_overflow']/span/text()")
		if nameNode == nil {
			log.Println("Warning: no name node found")
			continue
		}

		tempNode := htmlquery.FindOne(item, "//td[@class='temperature']/text()")
		if tempNode == nil {
			log.Println("Warning: no temperature node found")
			continue
		}

		targetTempNodes := htmlquery.Find(item, "//td[@class='target_temperature']//span[@class='numdisplay']/span/text()")
		if len(targetTempNodes) != 2 {
			log.Printf("Warning: not two elements in target temperature %d", len(targetTempNodes))
			continue
		}

		name := strings.TrimSpace(nameNode.Data)
		tempStr := strings.Replace(strings.Trim(tempNode.Data, " \n°C"), ",", ".", 1)
		targetTempStr := strings.Replace(strings.TrimSpace(targetTempNodes[0].Data), ",", ".", 1)

		temp, err := strconv.ParseFloat(tempStr, 64)
		if err != nil {
			log.Printf("Error parsing temperature: %s", err)
			continue
		}

		targetTemp, err := strconv.ParseFloat(targetTempStr, 64)
		if err != nil {
			log.Printf("Error parsing target temperature: %s", err)
			continue
		}

		result.Thermostats = append(result.Thermostats, thermostat{
			Name:               name,
			CurrentTemperature: temp,
			TargetTemperature:  targetTemp,
		})
	}
	return nil
}
