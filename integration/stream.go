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
)

type StreamParams struct {
	StartTimeStamp  int      `json:"start_time_stamp"`
	SamplesExpected int      `json:"samples_expected"`
	ExpectedError   string   `json:"expected_error"`
	Channels        []string `json:"channels"`
}

func (p StreamParams) serialize() []string {
	args := make([]string, 0)
	args = append(
		args,
		strconv.FormatInt(int64(p.StartTimeStamp), 10),
		strconv.Itoa(p.SamplesExpected),
		p.ExpectedError,
		strconv.Itoa(len(p.Channels)),
	)

	for _, g := range p.Channels {
		args = append(args, g)
	}

	return args
}

func (p StreamParams) ToPythonCommand(identifier string) string {
	cmd := "poetry run python stream.py "
	cmd += identifier + " " + strings.Join(p.serialize(), " ")
	return cmd
}

func (p StreamParams) ToTSCommand(identifier string) string {
	cmd := "npx tsx stream.ts "
	cmd += identifier + " " + strings.Join(p.serialize(), " ")
	return cmd
}

var _ NodeParams = &StreamParams{}
