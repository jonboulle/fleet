package main

import (
	"os"
	"path/filepath"

	"github.com/coreos/fleet"
	"github.com/coreos/fleet/fleetctl"
)

func main() {
	binary := filepath.Base(os.Args[0])
	switch binary {
	case "fleet":
		fleet.Main()
	case "fleetctl":
		fleetctl.Main()
	default:
		os.Stderr.WriteString("unrecognised binary:" + binary + "\n")
	}
}
