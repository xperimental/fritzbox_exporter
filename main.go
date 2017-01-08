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
	"log"
	"net/http"

	"github.com/ndecker/fritzbox_exporter/collector"
	"github.com/prometheus/client_golang/prometheus"
)

func main() {
	cfg, err := parseFlags()
	if err != nil {
		log.Fatalf("Error in parameters: %s", err)
	}

	if cfg.Test {
		printUPNPTree(cfg.GatewayAddress, uint16(cfg.GatewayPort))
		return
	}

	collector := collector.New(cfg.GatewayAddress, uint16(cfg.GatewayPort))
	prometheus.MustRegister(collector)

	http.Handle("/metrics", prometheus.Handler())
	http.Handle("/", http.RedirectHandler("/metrics", http.StatusFound))

	log.Printf("Listening on %s...", cfg.Addr)
	log.Fatal(http.ListenAndServe(cfg.Addr, nil))
}
