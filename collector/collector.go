package collector

import (
	"fmt"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	upnp "github.com/ndecker/fritzbox_exporter/fritzbox_upnp"
)

const serviceLoadRetryTime = 1 * time.Minute

type metric struct {
	Service string
	Action  string
	Result  string
	OkValue string

	Desc       *prometheus.Desc
	MetricType prometheus.ValueType
}

var metrics = []*metric{
	{
		Service: "urn:schemas-upnp-org:service:WANCommonInterfaceConfig:1",
		Action:  "GetTotalPacketsReceived",
		Result:  "TotalPacketsReceived",
		Desc: prometheus.NewDesc(
			"gateway_wan_packets_received",
			"packets received on gateway WAN interface",
			[]string{"gateway"},
			nil,
		),
		MetricType: prometheus.CounterValue,
	},
	{
		Service: "urn:schemas-upnp-org:service:WANCommonInterfaceConfig:1",
		Action:  "GetTotalPacketsSent",
		Result:  "TotalPacketsSent",
		Desc: prometheus.NewDesc(
			"gateway_wan_packets_sent",
			"packets sent on gateway WAN interface",
			[]string{"gateway"},
			nil,
		),
		MetricType: prometheus.CounterValue,
	},
	{
		Service: "urn:schemas-upnp-org:service:WANCommonInterfaceConfig:1",
		Action:  "GetAddonInfos",
		Result:  "TotalBytesReceived",
		Desc: prometheus.NewDesc(
			"gateway_wan_bytes_received",
			"bytes received on gateway WAN interface",
			[]string{"gateway"},
			nil,
		),
		MetricType: prometheus.CounterValue,
	},
	{
		Service: "urn:schemas-upnp-org:service:WANCommonInterfaceConfig:1",
		Action:  "GetAddonInfos",
		Result:  "TotalBytesSent",
		Desc: prometheus.NewDesc(
			"gateway_wan_bytes_sent",
			"bytes sent on gateway WAN interface",
			[]string{"gateway"},
			nil,
		),
		MetricType: prometheus.CounterValue,
	},
	{
		Service: "urn:schemas-upnp-org:service:WANCommonInterfaceConfig:1",
		Action:  "GetCommonLinkProperties",
		Result:  "Layer1UpstreamMaxBitRate",
		Desc: prometheus.NewDesc(
			"gateway_wan_layer1_upstream_max_bitrate",
			"Layer1 upstream max bitrate",
			[]string{"gateway"},
			nil,
		),
		MetricType: prometheus.GaugeValue,
	},
	{
		Service: "urn:schemas-upnp-org:service:WANCommonInterfaceConfig:1",
		Action:  "GetCommonLinkProperties",
		Result:  "Layer1DownstreamMaxBitRate",
		Desc: prometheus.NewDesc(
			"gateway_wan_layer1_downstream_max_bitrate",
			"Layer1 downstream max bitrate",
			[]string{"gateway"},
			nil,
		),
		MetricType: prometheus.GaugeValue,
	},
	{
		Service: "urn:schemas-upnp-org:service:WANCommonInterfaceConfig:1",
		Action:  "GetCommonLinkProperties",
		Result:  "PhysicalLinkStatus",
		OkValue: "Up",
		Desc: prometheus.NewDesc(
			"gateway_wan_layer1_link_status",
			"Status of physical link (Up = 1)",
			[]string{"gateway"},
			nil,
		),
		MetricType: prometheus.GaugeValue,
	},
	{
		Service: "urn:schemas-upnp-org:service:WANIPConnection:1",
		Action:  "GetStatusInfo",
		Result:  "ConnectionStatus",
		OkValue: "Connected",
		Desc: prometheus.NewDesc(
			"gateway_wan_connection_status",
			"WAN connection status (Connected = 1)",
			[]string{"gateway"},
			nil,
		),
		MetricType: prometheus.GaugeValue,
	},
	{
		Service: "urn:schemas-upnp-org:service:WANIPConnection:1",
		Action:  "GetStatusInfo",
		Result:  "Uptime",
		Desc: prometheus.NewDesc(
			"gateway_wan_connection_uptime_seconds",
			"WAN connection uptime",
			[]string{"gateway"},
			nil,
		),
		MetricType: prometheus.GaugeValue,
	},
}

type fritzboxCollector struct {
	Gateway string
	Port    uint16

	errors     prometheus.Counter
	sync.Mutex // protects Root
	Root       *upnp.Root
}

// New creates a new prometheus collector which fetches metrics from a FritzBox UPNP interface.
func New(gateway string, port uint16) prometheus.Collector {
	collector := &fritzboxCollector{
		Gateway: gateway,
		Port:    port,
		errors: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "fritzbox_exporter_collect_errors",
			Help: "Number of collection errors.",
		}),
	}
	go collector.LoadServices()

	return collector
}

// LoadServices tries to load the service information. Retries until success.
func (fc *fritzboxCollector) LoadServices() {
	for {
		root, err := upnp.LoadServices(fc.Gateway, fc.Port)
		if err != nil {
			fmt.Printf("cannot load services: %s\n", err)

			time.Sleep(serviceLoadRetryTime)
			continue
		}

		fmt.Printf("services loaded\n")

		fc.Lock()
		fc.Root = root
		fc.Unlock()
		return
	}
}

func (fc *fritzboxCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, m := range metrics {
		ch <- m.Desc
	}
	ch <- fc.errors.Desc()
}

func (fc *fritzboxCollector) Collect(ch chan<- prometheus.Metric) {
	ch <- fc.errors

	fc.Lock()
	root := fc.Root
	fc.Unlock()

	if root == nil {
		// Services not loaded yet
		return
	}

	var err error
	var lastService string
	var lastMethod string
	var lastResult upnp.Result

	for _, m := range metrics {
		if m.Service != lastService || m.Action != lastMethod {
			service, ok := root.Services[m.Service]
			if !ok {
				// TODO
				fmt.Println("cannot find service", m.Service)
				fmt.Println(root.Services)
				continue
			}
			action, ok := service.Actions[m.Action]
			if !ok {
				// TODO
				fmt.Println("cannot find action", m.Action)
				continue
			}

			lastResult, err = action.Call()
			if err != nil {
				fmt.Println(err)
				fc.errors.Inc()
				continue
			}
		}

		val, ok := lastResult[m.Result]
		if !ok {
			fmt.Println("result not found", m.Result)
			fc.errors.Inc()
			continue
		}

		var floatval float64
		switch tval := val.(type) {
		case uint64:
			floatval = float64(tval)
		case bool:
			if tval {
				floatval = 1
			} else {
				floatval = 0
			}
		case string:
			if tval == m.OkValue {
				floatval = 1
			} else {
				floatval = 0
			}
		default:
			fmt.Println("unknown", val)
			fc.errors.Inc()
			continue

		}

		ch <- prometheus.MustNewConstMetric(
			m.Desc,
			m.MetricType,
			floatval,
			fc.Gateway,
		)
	}
}
