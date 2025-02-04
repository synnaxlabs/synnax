// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

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
	time.Sleep((2 * telem.Second).Duration())

	return runNode(
		context.Background(),
		TestNode{Client: p.Client, Params: p},
		"cleanup",
		verbose,
	)
}
