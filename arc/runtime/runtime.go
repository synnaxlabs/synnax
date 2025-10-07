// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package runtime

import (
	"context"

	"github.com/samber/lo"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/module"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
)

type Runtime struct {
	module module.Module
	state  struct {
		affectedNodes set.Set[string]
		channels      map[uint32]*channel
		nodes         map[string]*node
	}
}

func (r *Runtime) Next(ctx context.Context, fr telem.Frame[uint32]) {
	clear(r.state.affectedNodes)
	for rawI, k := range fr.RawKeys() {
		if fr.ShouldExcludeRaw(rawI) {
			continue
		}
		if state, ok := r.state.channels[k]; ok {
			state.buf = state.buf.Append(fr.RawSeriesAt(rawI))
			state.hasData = true
			r.state.affectedNodes.Add(state.deps...)
		}
	}

	for _, stratum := range r.module.Strata {
		for _, node := range stratum {
			// Execute node
			// Add affected ndoes
		}
	}
	// GC series
}

type Config struct {
	Module module.Module
}

func New(cfg Config) (*Runtime, error) {
	r := &Runtime{module: cfg.Module}
	r.state.channels = newChannels(cfg)
	r.state.nodes = newNodes(cfg)
	return r, nil
}

func newChannels(cfg Config) map[uint32]*channel {
	channelKeys := cfg.Module.ReadChannels()
	channelKeys.Add(cfg.Module.WriteChannels().Keys()...)
	channels := make(map[uint32]*channel, len(channelKeys))
	for key := range channelKeys {
		channels[key] = &channel{
			key:      key,
			dataType: ir.F64{},
			deps: lo.FilterMap(cfg.Module.Nodes, func(item ir.Node, index int) (string, bool) {
				return item.Key, item.Channels.Read.Contains(key)
			}),
		}
	}
	return channels
}

func newNodes(cfg Config) map[string]*node {
	nodes := make(map[string]*node)
	for _, modNode := range cfg.Module.Nodes {
		nodes[modNode.Key] = &node{
			key: modNode.Key,
			channelWaterMarks: lo.SliceToMap(modNode.Channels.Read.Keys(), func(key uint32) (uint32, telem.Alignment) {
				return key, telem.Alignment(0)
			}),
			requiredInputs: make(set.Set[uint32]),
			state:          make(map[string]any),
			incoming: lo.Filter(cfg.Module.Edges, func(item ir.Edge, index int) bool {
				return item.Target.Node == modNode.Key
			}),
			outgoing: lo.Filter(cfg.Module.Edges, func(item ir.Edge, index int) bool {
				return item.Target.Node == modNode.Key
			}),
		}
	}
	return nodes
}
