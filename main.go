package main

// Copyright 2016 Nils Decker
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import (
	"flag"
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/ndecker/fritzbox_exporter/collector"
	upnp "github.com/ndecker/fritzbox_exporter/fritzbox_upnp"
)

var (
	flag_test = flag.Bool("test", false, "print all available metrics to stdout")
	flag_addr = flag.String("listen-address", ":9133", "The address to listen on for HTTP requests.")

	flag_gateway_address = flag.String("gateway-address", "fritz.box", "The URL of the upnp service")
	flag_gateway_port    = flag.Int("gateway-port", 49000, "The URL of the upnp service")
)

func test() {
	root, err := upnp.LoadServices(*flag_gateway_address, uint16(*flag_gateway_port))
	if err != nil {
		panic(err)
	}

	for _, s := range root.Services {
		fmt.Printf("%s: %s\n", s.Device.FriendlyName, s.ServiceType)
		for _, a := range s.Actions {
			if !a.IsGetOnly() {
				continue
			}

			res, err := a.Call()
			if err != nil {
				panic(err)
			}

			fmt.Printf("  %s\n", a.Name)
			for _, arg := range a.Arguments {
				fmt.Printf("    %s: %v\n", arg.RelatedStateVariable, res[arg.StateVariable.Name])
			}
		}
	}
}

func main() {
	flag.Parse()

	if *flag_test {
		test()
		return
	}

	collector := collector.New(*flag_gateway_address, uint16(*flag_gateway_port))
	prometheus.MustRegister(collector)

	go collector.LoadServices()

	http.Handle("/metrics", prometheus.Handler())
	http.Handle("/", http.RedirectHandler("/metrics", http.StatusFound))
	http.ListenAndServe(*flag_addr, nil)
}
