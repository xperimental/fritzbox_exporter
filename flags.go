package main

import (
	"errors"
	"flag"
	"fmt"
)

type config struct {
	Test            bool
	Addr            string
	GatewayAddress  string
	GatewayPort     int
	GatewayPassword string
}

func parseFlags() (config, error) {
	cfg := config{
		Test:            false,
		Addr:            ":9133",
		GatewayAddress:  "fritz.box",
		GatewayPort:     49000,
		GatewayPassword: "",
	}

	flag.BoolVar(&cfg.Test, "test", cfg.Test, "print all available metrics to stdout")
	flag.StringVar(&cfg.Addr, "listen-address", cfg.Addr, "The address to listen on for HTTP requests.")
	flag.StringVar(&cfg.GatewayAddress, "gateway-address", cfg.GatewayAddress, "The URL of the upnp service")
	flag.IntVar(&cfg.GatewayPort, "gateway-port", cfg.GatewayPort, "The URL of the upnp service")
	flag.StringVar(&cfg.GatewayPassword, "gateway-password", cfg.GatewayPassword, "Password for the router admin interface.")
	flag.Parse()

	if len(cfg.Addr) == 0 {
		return cfg, errors.New("no listen address")
	}

	if len(cfg.GatewayAddress) == 0 {
		return cfg, errors.New("no gateway address")
	}

	if cfg.GatewayPort < 1 || cfg.GatewayPort > 65534 {
		return cfg, fmt.Errorf("invalid gateway port: %d", cfg.GatewayPort)
	}

	return cfg, nil
}
