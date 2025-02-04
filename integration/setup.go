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
)

type SetUpParam struct {
	IndexChannels int    `json:"index_channels"`
	DataChannels  int    `json:"data_channels"`
	Client        string `json:"client"`
}

func (p SetUpParam) serialize() []string {
	return []string{}
}

func (p SetUpParam) ToPythonCommand(_ string) string {
	if p == (SetUpParam{}) {
		return ""
	}

	return fmt.Sprintf(
		"poetry run python setup.py %d %d",
		p.IndexChannels,
		p.DataChannels,
	)
}

func (p SetUpParam) ToTSCommand(_ string) string {
	if p == (SetUpParam{}) {
		return ""
	}

	return fmt.Sprintf(
		"npx tsx setup.ts %d %d",
		p.IndexChannels,
		p.DataChannels,
	)
}

var _ NodeParams = &SetUpParam{}

func runSetUp(p SetUpParam, verbose bool) error {
	if p == (SetUpParam{}) {
		fmt.Printf("--cannot find setup configuration, skipping\n")
		return nil
	}

	fmt.Printf("--setting up\n")
	return runNode(
		context.Background(),
		TestNode{Client: p.Client, Params: p},
		"setup",
		verbose,
	)

}
