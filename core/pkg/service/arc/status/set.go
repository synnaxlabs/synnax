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

// Two separate resolvers are needed because the bare name ("set_status")
// differs from the qualified member name ("set"). The bare form will be
// deprecated and removed once users migrate to status.set{}.
var (
	symbolProps = types.Function(types.FunctionProperties{
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
	bareResolver = symbol.MapResolver{
		bareSymbolName: {
			Name: bareSymbolName,
			Kind: symbol.KindFunction,
			Exec: symbol.ExecFlow,
			Type: symbolProps,
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
	stat *status.Service
}

func NewModule(stat *status.Service) *Module {
	return &Module{stat: stat}
}

func (m *Module) Resolve(ctx context.Context, name string) (symbol.Symbol, error) {
	return SymbolResolver.Resolve(ctx, name)
}

func (m *Module) Search(ctx context.Context, term string) ([]symbol.Symbol, error) {
	return SymbolResolver.Search(ctx, term)
}

func (m *Module) ModuleName() string { return moduleName }

func (m *Module) Create(ctx context.Context, cfg node.Config) (node.Node, error) {
	f := &statusFactory{stat: m.stat}
	return f.Create(ctx, cfg)
}

type setStatus struct {
	statusSvc *status.Service
	ins       alamos.Instrumentation
	stat      status.Status[any]
}

func (s *setStatus) Init(node.Context) {}

func (s *setStatus) Reset() {}

func (s *setStatus) IsOutputTruthy(int) bool {
	return false
}

func (s *setStatus) Next(ctx node.Context) {
	s.stat.Time = telem.Now()
	if err := s.statusSvc.NewWriter(nil).Set(ctx, &s.stat); err != nil {
		s.ins.L.Error("error setting status", zap.Error(err))
	}
}

type statusFactory struct {
	stat *status.Service
}

var schema = zyn.Object(map[string]zyn.Schema{
	"status_key": zyn.String(),
	"message":    zyn.String(),
	"variant":    zyn.String(),
	"name":       zyn.String().Optional(),
})

type nodeConfig struct {
	StatusKey string `json:"status_key"`
	Message   string `json:"message"`
	Variant   string `json:"variant"`
	Name      string `json:"name"`
}

func (s *statusFactory) Create(ctx context.Context, cfg node.Config) (node.Node, error) {
	if cfg.Node.Type != bareSymbolName && cfg.Node.Type != qualifiedMemberName {
		return nil, query.ErrNotFound
	}
	var nodeCfg nodeConfig
	if err := schema.Parse(cfg.Node.Config.ValueMap(), &nodeCfg); err != nil {
		return nil, err
	}
	var stat status.Status[any]
	if err := s.stat.NewRetrieve().
		WhereKeys(nodeCfg.StatusKey).
		Entry(&stat).
		Exec(ctx, nil); errors.Skip(err, query.ErrNotFound) != nil {
		return nil, err
	}
	stat.Key = nodeCfg.StatusKey
	stat.Name = nodeCfg.Name
	stat.Message = nodeCfg.Message
	stat.Variant = xstatus.Variant(nodeCfg.Variant)
	return &setStatus{ins: cfg.Instrumentation, stat: stat, statusSvc: s.stat}, nil
}
