// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package authority

import (
	"context"

	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/zyn"
)

const (
	bareSymbolName      = "set_authority"
	qualifiedMemberName = "set"
	moduleName          = "authority"
)

// Two separate resolvers are needed because the bare name ("set_authority")
// differs from the qualified member name ("set"). Most STL modules share a
// single resolver for both forms (e.g. time uses "interval" for both bare
// and time.interval). The bare form will be deprecated and removed once
// users migrate to authority.set{}.
var (
	symbolProps = types.Function(types.FunctionProperties{
		Config: types.Params{
			{Name: "value", Type: types.U8()},
			{Name: "channel", Type: types.WriteChan(types.Variable("T", nil)), Value: uint32(0)},
		},
		Inputs: types.Params{
			{Name: ir.DefaultOutputParam, Type: types.U8(), Value: uint8(0)},
		},
	})
	bareResolver = symbol.MapResolver{
		bareSymbolName: {
			Name:       bareSymbolName,
			Kind:       symbol.KindFunction,
			Exec:       symbol.ExecFlow,
			Type:       symbolProps,
			Deprecated: "authority.set",
		},
	}
	moduleResolver = &symbol.ModuleResolver{
		Name: moduleName,
		Members: symbol.MapResolver{
			qualifiedMemberName: {
				Name: qualifiedMemberName,
				Kind: symbol.KindFunction,
				Exec: symbol.ExecFlow,
				Type: symbolProps,
			},
		},
	}
	SymbolResolver = symbol.CompoundResolver{bareResolver, moduleResolver}
)

type Module struct {
	auth *ProgramState
}

func NewModule(ab *ProgramState) *Module { return &Module{auth: ab} }

func (m *Module) ModuleName() string { return moduleName }

func (m *Module) Create(_ context.Context, cfg node.Config) (node.Node, error) {
	if cfg.Node.Type != bareSymbolName && cfg.Node.Type != qualifiedMemberName {
		return nil, query.ErrNotFound
	}
	var nodeCfg nodeConfig
	if err := schema.Parse(cfg.Node.Config.ValueMap(), &nodeCfg); err != nil {
		return nil, errors.Wrap(err, "authority.set config")
	}
	var channelKey *uint32
	if nodeCfg.Channel != 0 {
		channelKey = &nodeCfg.Channel
	}
	return &setAuthority{
		auth:       m.auth,
		authority:  nodeCfg.Value,
		channelKey: channelKey,
	}, nil
}

var schema = zyn.Object(map[string]zyn.Schema{
	"value":   zyn.Number().Uint8(),
	"channel": zyn.Number().Uint32(),
})

type nodeConfig struct {
	Value   uint8  `json:"value"`
	Channel uint32 `json:"channel"`
}

type setAuthority struct {
	auth        *ProgramState
	authority   uint8
	channelKey  *uint32
	initialized bool
}

func (s *setAuthority) Reset()                  { s.initialized = false }
func (s *setAuthority) IsOutputTruthy(int) bool { return false }

func (s *setAuthority) Next(node.Context) {
	if s.initialized {
		return
	}
	s.initialized = true
	s.auth.Set(s.channelKey, s.authority)
}
