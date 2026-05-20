// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package control

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
	qualifiedMemberName = "set_authority"
	moduleName          = "control"
)

// Two separate resolvers are needed because the bare top-level form
// ("set_authority") and the qualified form ("control.set_authority") both
// need to resolve to the same function. The bare form is deprecated;
// its Deprecated field points at the canonical member inside the module.
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
	memberSymbol = symbol.Symbol{
		Name: qualifiedMemberName,
		Kind: symbol.KindFunction,
		Exec: symbol.ExecFlow,
		Type: symbolProps,
	}
	module     = symbol.NewModule(moduleName, memberSymbol)
	bareSymbol = symbol.Symbol{
		Name:       bareSymbolName,
		Kind:       symbol.KindFunction,
		Exec:       symbol.ExecFlow,
		Type:       symbolProps,
		Deprecated: module.FindChildByName(qualifiedMemberName),
	}
	bareResolver   = symbol.MapResolver{bareSymbolName: bareSymbol}
	moduleResolver = &symbol.ModuleResolver{
		Name:    moduleName,
		Members: symbol.MapResolver{qualifiedMemberName: memberSymbol},
	}
	SymbolResolver = symbol.CompoundResolver{bareResolver, moduleResolver}
)

// BuildModule returns the control module with its sealed namespace populated.
func BuildModule() *symbol.Symbol { return module }

// BareGlobals returns the deprecated bare alias installed at the root scope
// so legacy programs continue to resolve. Deprecated points at the
// canonical control.set_authority member inside the module.
func BareGlobals() []symbol.Symbol { return []symbol.Symbol{bareSymbol} }

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
		return nil, errors.Wrap(err, "control.set_authority config")
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
