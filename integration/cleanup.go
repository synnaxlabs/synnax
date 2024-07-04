package main

import (
	"context"
	"fmt"
	"time"

	"github.com/synnaxlabs/x/telem"
)

type CleanUpParam struct {
	DeleteAllChannels bool   `json:"delete_all_channels"`
	Client            string `json:"client"`
}

func (p CleanUpParam) serialize() []string {
	return []string{}
}

func (p CleanUpParam) ToPythonCommand(_ string) string {
	if !p.DeleteAllChannels {
		return ""
	}

	return "poetry run python delete_all.py"
}

func (p CleanUpParam) ToTSCommand(_ string) string {
	panic("unimplemented")
}

var _ NodeParams = &CleanUpParam{}

func runCleanUp(p CleanUpParam, verbose bool) error {
	if p == (CleanUpParam{}) {
		fmt.Printf("--cannot find cleanup configuration, skipping\n")
		return nil
	}
	fmt.Printf("--cleaning up\n")
	time.Sleep((5 * telem.Second).Duration())

	return runNode(
		context.Background(),
		TestNode{Client: p.Client, Params: p},
		"cleanup",
		verbose,
	)
}
