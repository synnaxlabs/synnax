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
	"github.com/synnaxlabs/arc/runtime/state"
	"github.com/synnaxlabs/x/query"
	xtelem "github.com/synnaxlabs/x/telem"
)

var (
	sourceSymbolName = "on"
	sourceSymbol     = ir.Symbol{
		Name: sourceSymbolName,
		Kind: ir.KindStage,
		Type: ir.Stage{
			Config: ir.NamedTypes{
				Keys:   []string{"channel"},
				Values: []ir.Type{ir.Chan{}},
			},
			Outputs: ir.NamedTypes{
				Keys:   []string{ir.DefaultOutput},
				Values: []ir.Type{ir.NewTypeVariable("T", nil)},
			},
		},
	}
	Resolver = ir.MapResolver{
		"on": sourceSymbol,
	}
)

type source struct {
	node          ir.Node
	telem         *State
	state         *state.State
	key           uint32
	highWaterMark xtelem.Alignment
}

func (s *source) Init(ctx context.Context, changed func(output string)) {}

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
	if cfg.Node.Type != sourceSymbolName {
		return nil, query.NotFound
	}
	key := cfg.Node.Channels.Read.Keys()[0]
	t.telem.Register(key, cfg.Node.Key)
	return &source{
		node:          cfg.Node,
		telem:         t.telem,
		state:         cfg.State,
		key:           key,
		highWaterMark: 0,
	}, nil
}

func NewTelemFactory(state *State) node.Factory {
	return &telemFactory{telem: state}
}
