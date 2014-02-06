package main

import (
	"fmt"

	"github.com/codegangsta/cli"

	"github.com/coreos/fleet/job"
)

func newSubmitUnitCommand() cli.Command {
	return cli.Command{
		Name:   "submit",
		Usage:  "Upload one or more units to the cluster without starting them",
		Action: submitUnitsAction,
	}
}

func submitUnitsAction(c *cli.Context) {
	r := getRegistry(c)

	// First, validate each of the provided payloads
	payloads := make([]job.JobPayload, len(c.Args()))
	for i, v := range c.Args() {
		payload, err := getJobPayloadFromFile(v)
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		payloads[i] = *payload
	}

	// Only after all the provided payloads have been validated
	// do we push any changes to the Registry
	for _, payload := range payloads {
		err := r.CreatePayload(&payload)
		if err != nil {
			fmt.Printf("Creation of Payload %s failed: %v", err)
			return
		}
	}
}
