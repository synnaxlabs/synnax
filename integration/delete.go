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
	"strconv"
	"strings"

	"github.com/synnaxlabs/x/telem"
)

type DeleteParams struct {
	TimeRange     telem.TimeRange `json:"time_range"`
	Channels      []string        `json:"channels"`
	ExpectedError string          `json:"expected_error"`
}

func (p DeleteParams) serialize() []string {
	args := make([]string, 0)
	args = append(
		args,
		strconv.FormatInt(int64(p.TimeRange.Start), 10),
		strconv.FormatInt(int64(p.TimeRange.End), 10),
		p.ExpectedError,
		strconv.Itoa(len(p.Channels)),
	)

	for _, g := range p.Channels {
		args = append(args, g)
	}

	return args
}

func (p DeleteParams) ToPythonCommand(identifier string) string {
	cmd := "poetry run python delete.py "
	cmd += identifier + " " + strings.Join(p.serialize(), " ")
	return cmd
}

func (p DeleteParams) ToTSCommand(identifier string) string {
	cmd := "npx tsx delete.ts "
	cmd += identifier + " " + strings.Join(p.serialize(), " ")
	return cmd
}

var _ NodeParams = &DeleteParams{}
