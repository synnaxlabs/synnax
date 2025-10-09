// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package wasm

import (
	"context"

	"github.com/samber/lo"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/module"
	node2 "github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/x/query"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

type factory struct {
	wasm api.Module
}

func (w *factory) Create(ctx context.Context, cfg node2.Config) (node2.Node, error) {
	stage, ok := cfg.Module.GetStage(cfg.Node.Type)
	if !ok {
		return nil, query.NotFound
	}
	fn := w.wasm.ExportedFunction(cfg.Node.Type)
	return &node{
		ir: cfg.Node,
		wasm: WrapFunction(
			fn,
			w.wasm.Memory(),
			stage.Outputs,
			cfg.Module.OutputMemoryBases[cfg.Node.Type],
		),
		inputs: lo.Filter(cfg.Module.Edges, func(item ir.Edge, index int) bool {
			return item.Target.Node == cfg.Node.Key
		}),
		outputs: lo.Filter(cfg.Module.Edges, func(item ir.Edge, index int) bool {
			return item.Source.Node == cfg.Node.Key
		}),
		state:  cfg.State,
		params: make([]uint64, len(stage.Outputs.Keys)),
	}, nil
}

type FactoryConfig struct {
	Module module.Module
}

func NewFactory(ctx context.Context, cfg FactoryConfig) (node2.Factory, error) {
	runtime := wazero.NewRuntime(ctx)
	wasmModule, err := runtime.Instantiate(ctx, cfg.Module.WASM)
	if err != nil {
		return nil, err
	}
	return &factory{wasm: wasmModule}, nil
}
