// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package status

import (
	"context"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/zyn"

	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	xstatus "github.com/synnaxlabs/x/status"
	"go.uber.org/zap"
)

const (
	bareSymbolName      = "set_status"
	qualifiedMemberName = "set"
	moduleName          = "status"
)

var symbolProps = types.Function(types.FunctionProperties{
	Config: types.Params{
		{Name: "status_key", Type: types.String()},
		{Name: "variant", Type: types.String()},
		{Name: "message", Type: types.String()},
		{Name: "name", Type: types.String(), Value: ""},
	},
	Inputs: types.Params{
		{Name: ir.DefaultOutputParam, Type: types.U8()},
	},
})

// memberSymbol is the canonical status.set symbol that lives inside the
// status module. The deprecated bare global below points at it via
// Deprecated so analyzer hints can name the replacement.
var memberSymbol = symbol.Symbol{
	Name: qualifiedMemberName,
	Kind: symbol.KindFunction,
	Exec: symbol.ExecFlow,
	Type: symbolProps,
}

// module is the status module, built once at package init. Its sole child
// is the `set` function above.
var module = symbol.NewModule(moduleName, memberSymbol)

// bareSymbolPtr is the deprecated `set_status` bare global. Its
// Deprecated field points at the module's `set` member so warnings name
// the qualified replacement.
var bareSymbolPtr = &symbol.Symbol{
	Name:       bareSymbolName,
	Kind:       symbol.KindFunction,
	Exec:       symbol.ExecFlow,
	Type:       symbolProps,
	Deprecated: module.FindChild(qualifiedMemberName),
}

// Symbols are the symbols this package contributes to a program's
// ambient prelude: the status module plus the deprecated bare global.
var Symbols = []*symbol.Symbol{module, bareSymbolPtr}

type Module struct {
	stat *status.Service
}

func NewModule(stat *status.Service) *Module {
	return &Module{stat: stat}
}

func (m *Module) ModuleName() string { return moduleName }

func (m *Module) Create(ctx context.Context, cfg node.Config) (node.Node, error) {
	if cfg.Node.Type != bareSymbolName && cfg.Node.Type != qualifiedMemberName {
		return nil, query.ErrNotFound
	}
	var nodeCfg setNodeConfig
	if err := setNodeConfigSchema.Parse(cfg.Node.Config.ValueMap(), &nodeCfg); err != nil {
		return nil, err
	}
	var stat status.Status[any]
	if err := m.stat.NewRetrieve().
		Where(status.MatchKeys[any](nodeCfg.StatusKey)).
		Entry(&stat).
		Exec(ctx, nil); errors.Skip(err, query.ErrNotFound) != nil {
		return nil, err
	}
	stat.Key = nodeCfg.StatusKey
	stat.Name = nodeCfg.Name
	stat.Message = nodeCfg.Message
	stat.Variant = xstatus.Variant(nodeCfg.Variant)
	return &setNode{ins: cfg.Instrumentation, stat: stat, statusSvc: m.stat}, nil
}

type setNodeConfig struct {
	StatusKey string `json:"status_key"`
	Message   string `json:"message"`
	Variant   string `json:"variant"`
	Name      string `json:"name"`
}

var setNodeConfigSchema = zyn.Object(map[string]zyn.Schema{
	"status_key": zyn.String(),
	"message":    zyn.String(),
	"variant":    zyn.String(),
	"name":       zyn.String().Optional(),
})

type setNode struct {
	statusSvc *status.Service
	ins       alamos.Instrumentation
	stat      status.Status[any]
}

func (s *setNode) Init(node.Context) {}

func (s *setNode) Reset() {}

func (s *setNode) IsOutputTruthy(int) bool { return false }

func (s *setNode) Next(ctx node.Context) {
	s.stat.Time = telem.Now()
	if err := s.statusSvc.NewWriter(nil).Set(ctx, &s.stat); err != nil {
		s.ins.L.Error("error setting status", zap.Error(err))
	}
}
