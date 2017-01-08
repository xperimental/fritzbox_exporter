package main

import (
	"fmt"

	"github.com/ndecker/fritzbox_exporter/upnp"
)

func printUPNPTree(address string, port uint16) {
	root, err := upnp.LoadServices(address, port)
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
