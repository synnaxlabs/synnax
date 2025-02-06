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

type ReadParams struct {
	NumIterators    int             `json:"num_iterators"`
	ChunkSize       int             `json:"chunk_size"`
	Bounds          telem.TimeRange `json:"bounds"`
	ChannelGroups   [][]string      `json:"channel_groups"`
	SamplesExpected int             `json:"samples_expected"`
	ExpectedError   string          `json:"expected_error"`
}

func (p ReadParams) serialize() []string {
	args := make([]string, 0)
	args = append(
		args,
		strconv.Itoa(p.NumIterators),
		strconv.Itoa(p.ChunkSize),
		strconv.FormatInt(int64(p.Bounds.Start), 10),
		strconv.FormatInt(int64(p.Bounds.End), 10),
		strconv.FormatInt(int64(p.SamplesExpected), 10),
		p.ExpectedError,
		strconv.Itoa(len(p.ChannelGroups)),
	)

	for _, g := range p.ChannelGroups {
		args = append(args, strconv.Itoa(len(g)))
		args = append(args, g...)
	}

	return args
}

func (p ReadParams) ToPythonCommand(identifier string) string {
	cmd := "poetry run python read.py "
	cmd += identifier + " " + strings.Join(p.serialize(), " ")
	return cmd
}

func (p ReadParams) ToTSCommand(identifier string) string {
	cmd := "npx tsx read.ts "
	cmd += identifier + " " + strings.Join(p.serialize(), " ")
	return cmd
}

var _ NodeParams = &ReadParams{}
