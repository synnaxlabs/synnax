// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package telem

import (
	"context"

	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/runtime/node"
	state2 "github.com/synnaxlabs/arc/runtime/state"
	"github.com/synnaxlabs/x/query"
	xtelem "github.com/synnaxlabs/x/telem"
)

type source struct {
	node          ir.Node
	telem         *State
	state         *state2.State
	key           uint32
	highWaterMark xtelem.Alignment
}

func (s *source) Next(_ context.Context, onOutputChange func(param string)) {
	passSeries := s.telem.Data[s.key].FilterGreaterThanOrEqualTo(s.highWaterMark)
	s.highWaterMark = passSeries.AlignmentBounds().Upper
	s.state.Outputs[ir.Handle{Param: ir.DefaultOutput, Node: s.node.Key}] = passSeries.Series[0]
	onOutputChange(ir.DefaultOutput)
}

type telemFactory struct {
	telem *State
}

func (t telemFactory) Create(cfg node.Config) (node.Node, error) {
	if cfg.Node.Type != "on" {
		return nil, query.NotFound
	}
	return &source{
		node:          cfg.Node,
		telem:         t.telem,
		state:         cfg.State,
		key:           cfg.Node.Channels.Read.Keys()[0],
		highWaterMark: 0,
	}, nil
}

func NewTelemFactory(state *State) node.Factory {
	return &telemFactory{telem: state}
}
