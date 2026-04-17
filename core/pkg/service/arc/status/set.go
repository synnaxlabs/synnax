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

var (
	symbolName = "set_status"
	symbolSet  = symbol.Symbol{
		Name: "set_status",
		Kind: symbol.KindFunction,
		Type: types.Function(types.FunctionProperties{
			Config: types.Params{
				{Name: "status_key", Type: types.String()},
				{Name: "variant", Type: types.String()},
				{Name: "message", Type: types.String()},
				{Name: "name", Type: types.String(), Value: ""},
			},
			Inputs: types.Params{
				{Name: ir.DefaultOutputParam, Type: types.U8()},
			},
		}),
	}
	SymbolResolver = symbol.MapResolver{symbolName: symbolSet}
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

func (m *Module) Create(ctx context.Context, cfg node.Config) (node.Node, error) {
	if cfg.Node.Type != symbolName {
		return nil, query.ErrNotFound
	}
	var nodeCfg setNodeConfig
	if err := setNodeConfigSchema.Parse(cfg.Node.Config.ValueMap(), &nodeCfg); err != nil {
		return nil, err
	}
	var stat status.Status[any]
	if err := m.stat.NewRetrieve().
		WhereKeys(nodeCfg.StatusKey).
		Entry(&stat).
		Exec(ctx, nil); errors.Skip(err, query.ErrNotFound) != nil {
		return nil, err
	}
	stat.Key = nodeCfg.StatusKey
	stat.Message = nodeCfg.Message
	stat.Variant = xstatus.Variant(nodeCfg.Variant)
	return &setNode{ins: cfg.Instrumentation, stat: stat, statusSvc: m.stat}, nil
}

type setNodeConfig struct {
	StatusKey string `json:"status_key"`
	Message   string `json:"message"`
	Variant   string `json:"variant"`
}

var setNodeConfigSchema = zyn.Object(map[string]zyn.Schema{
	"status_key": zyn.String(),
	"message":    zyn.String(),
	"variant":    zyn.String(),
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


