// Package home can extract metrics from FritzBox home automation devices.
package home

import (
	"encoding/xml"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type function int

const (
	homeURLFormat = "http://%s/webservices/homeautoswitch.lua?switchcmd=getdevicelistinfos&sid=%s"

	functionAlarm       function = 1 << 4
	functionHeating     function = 1 << 6
	functionEnergy      function = 1 << 7
	functionTemperature function = 1 << 8
	functionSwitch      function = 1 << 9
	functionRepeater    function = 1 << 10

	thermostatTargetTempOff = 126.5
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
	switchStateDesc = prometheus.NewDesc(
		"fritzbox_home_switch_state",
		"State of power switch.",
		varLabels, nil)
	powerDesc = prometheus.NewDesc(
		"fritzbox_home_power_watt",
		"Current power measurement in watt.",
		varLabels, nil)
	energyDesc = prometheus.NewDesc(
		"fritzbox_home_energy_watthours",
		"Energy consumption in watt-hours.",
		varLabels, nil)

	dynamicDescs = []*prometheus.Desc{
		tempDesc,
		targetTempDesc,
		switchStateDesc,
		powerDesc,
		energyDesc,
	}
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

	for _, d := range dynamicDescs {
		ch <- d
	}
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

		if th.TargetTemperature != thermostatTargetTempOff {
			sendMetric(ch, targetTempDesc, th.TargetTemperature, labels)
		}
	}

	for _, sw := range home.Switches {
		labels := []string{
			c.Hostname,
			sw.Name,
		}

		state := 0.0
		if sw.On {
			state = 1.0
		}

		sendMetric(ch, switchStateDesc, state, labels)
		sendMetric(ch, powerDesc, sw.CurrentPower, labels)
		sendMetric(ch, energyDesc, sw.UsedEnergy, labels)
		sendMetric(ch, tempDesc, sw.CurrentTemperature, labels)
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
	Switches    []powerSwitch
}

type thermostat struct {
	Name               string
	CurrentTemperature float64
	TargetTemperature  float64
}

type powerSwitch struct {
	Name               string
	On                 bool
	Mode               string
	CurrentPower       float64
	UsedEnergy         float64
	CurrentTemperature float64
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

	var homeData homeDeviceList
	if err := xml.NewDecoder(res.Body).Decode(&homeData); err != nil {
		return result, err
	}

	for _, d := range homeData.Devices {
		switch {
		case d.hasFunctions(functionHeating, functionTemperature):
			name := d.Name
			temp := float64(d.Heating.Current) * 0.5
			targetTemp := float64(d.Heating.Target) * 0.5

			result.Thermostats = append(result.Thermostats, thermostat{
				Name:               name,
				CurrentTemperature: temp,
				TargetTemperature:  targetTemp,
			})
		case d.hasFunctions(functionEnergy, functionSwitch, functionTemperature):
			result.Switches = append(result.Switches, powerSwitch{
				Name:               d.Name,
				On:                 d.Switch.State,
				Mode:               d.Switch.Mode,
				CurrentPower:       float64(d.PowerMeter.Power) / 1000,
				UsedEnergy:         float64(d.PowerMeter.Energy),
				CurrentTemperature: float64(d.Temperature.Celsius) / 10,
			})
		}
	}
	return result, nil
}

type homeDeviceList struct {
	Version int          `xml:"version,attr"`
	Devices []homeDevice `xml:"device"`
}

type homeDevice struct {
	ID          int             `xml:"id,attr"`
	Name        string          `xml:"name"`
	Functions   int             `xml:"functionbitmask,attr"`
	Temperature homeTemperature `xml:"temperature"`
	Heating     homeHeating     `xml:"hkr"`
	Switch      homeSwitch      `xml:"switch"`
	PowerMeter  homePowerMeter  `xml:"powermeter"`
}

func (d homeDevice) hasFunctions(functions ...function) bool {
	for _, f := range functions {
		if d.Functions&int(f) == 0 {
			return false
		}
	}
	return true
}

// Values as int in 0.1°C increments (220 == 22.5°C)
type homeTemperature struct {
	Celsius int `xml:"celsius"`
	Offset  int `xml:"offset"`
}

// Values as int in 0.5°C increments (8 == 16°C)
type homeHeating struct {
	Current int `xml:"tist"`
	Target  int `xml:"tsoll"`
	Comfort int `xml:"komfort"`
	Night   int `xml:"absenk"`
}

type homeSwitch struct {
	State      bool   `xml:"state"`
	Mode       string `xml:"mode"` // Either "auto" or "manuell"
	Lock       bool   `xml:"lock"`
	DeviceLock bool   `xml:"devicelock"`
}

type homePowerMeter struct {
	Power  int `xml:"power"`  // Power in Milliwatt (0.001 W)
	Energy int `xml:"energy"` // Energy in Watt-Hours
}
