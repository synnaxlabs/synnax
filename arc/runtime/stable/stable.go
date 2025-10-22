// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package stable

import (
	"context"

	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/runtime/state"
	symbol2 "github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/zyn"
)

var (
	symbolName = "stable_for"
	symbol     = symbol2.Symbol{
		Name: symbolName,
		Kind: symbol2.KindFunction,
		Type: types.Function(types.FunctionProperties{
			Config: &types.Params{
				Keys:   []string{"duration"},
				Values: []types.Type{types.TimeSpan()},
			},
			Inputs: &types.Params{
				Keys:   []string{ir.DefaultInputParam},
				Values: []types.Type{types.NewTypeVariable("T", nil)},
			},
			Outputs: &types.Params{
				Keys:   []string{ir.DefaultOutputParam},
				Values: []types.Type{types.NewTypeVariable("T", nil)},
			},
		}),
	}
	SymbolResolver = symbol2.MapResolver{symbolName: symbol}
)

type stableFor struct {
	state       *state.Node
	duration    telem.TimeSpan
	value       *uint8
	lastSent    *uint8
	lastChanged telem.TimeStamp
	now         func() telem.TimeStamp
}

func (s *stableFor) Init(ctx node.Context) {}

func (s *stableFor) Next(ctx node.Context) {
	if s.state.RefreshInputs() {
		inputData := s.state.Input(0)
		inputTime := s.state.InputTime(0)
		if inputData.Len() > 0 {
			for i := int64(0); i < inputData.Len(); i++ {
				currentValue := telem.ValueAt[uint8](inputData, int(i))
				currentTime := telem.ValueAt[telem.TimeStamp](inputTime, int(i))
				if s.value == nil || *s.value != currentValue {
					s.value = &currentValue
					s.lastChanged = currentTime
				}
			}
		}
	}

	if s.value == nil {
		return
	}
	currentValue := *s.value
	if telem.TimeSpan(s.now()-s.lastChanged) >= s.duration {
		if s.lastSent == nil || *s.lastSent != currentValue {
			*s.state.Output(0) = telem.NewSeriesV[uint8](currentValue)
			*s.state.OutputTime(0) = telem.NewSeriesV[telem.TimeStamp](s.now())
			s.lastSent = &currentValue
			ctx.MarkChanged(ir.DefaultOutputParam)
		}
	}
}

type stableFactory struct {
	cfg FactoryConfig
}

type FactoryConfig struct {
	Now func() telem.TimeStamp
}

func NewFactory(cfg FactoryConfig) node.Factory {
	return &stableFactory{cfg: cfg}
}

type config struct {
	Duration telem.TimeSpan
}

var configSchema = zyn.Object(map[string]zyn.Schema{
	"duration": zyn.Int64().Coerce(),
})

func (f *stableFactory) Create(_ context.Context, cfg node.Config) (node.Node, error) {
	if cfg.Node.Type != symbolName {
		return nil, query.NotFound
	}
	var configVals config
	if err := configSchema.Parse(cfg.Node.ConfigValues, &configVals); err != nil {
		return nil, err
	}
	now := f.cfg.Now
	if now == nil {
		now = telem.Now
	}
	return &stableFor{state: cfg.State, duration: configVals.Duration, now: now}, nil
}
