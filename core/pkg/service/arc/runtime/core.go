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

	"github.com/synnaxlabs/arc"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/core"
	"github.com/synnaxlabs/synnax/pkg/service/arc/stage"
	"github.com/synnaxlabs/synnax/pkg/service/arc/std"
	"github.com/synnaxlabs/synnax/pkg/service/arc/value"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
)

type Core struct {
	module    arc.Module
	nodes     map[string]stage.Node
	values    map[channel.Key]telem.Series
	writeFunc func(ctx context.Context, fr core.Frame) error
}

func (c *Core) Get(key channel.Key) telem.Series {
	return c.values[key]
}

func NewCore(ctx context.Context, cfgs ...Config) (*Core, error) {
	cfg, err := config.New(DefaultConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	c := &Core{
		module: cfg.Module,
		nodes:  make(map[string]stage.Node),
		values: make(map[channel.Key]telem.Series),
	}
	create := func(arcNode arc.Node) (stage.Node, error) {
		_, ok := cfg.Module.GetStage(arcNode.Type)
		if ok {
			return nil, errors.Newf("unsupported module type: %s", arcNode.Type)
		}
		return std.Create(ctx, std.Config{
			Node:          arcNode,
			Status:        cfg.Status,
			ChannelData:   c,
			ChannelWriter: c,
			Channel:       cfg.Channel,
		})
	}
	for _, nodeSpec := range c.module.Nodes {
		n, err := create(nodeSpec)
		if err != nil {
			return nil, err
		}
		n.OnOutput(c.createOutputHandler(nodeSpec.Key))
		c.nodes[n.Key()] = n
	}
	return c, nil
}

func (c *Core) Flow(ctx signal.Context) {
	for _, node := range c.nodes {
		node.Flow(ctx)
	}
	c.evaluateStrata(ctx)
}

func (c *Core) Next(ctx context.Context, fr core.Frame) {
	for rawI, key := range fr.RawKeys() {
		if fr.ShouldExcludeRaw(rawI) {
			continue
		}
		c.values[key] = fr.RawSeriesAt(rawI)
	}
	c.evaluateStrata(ctx)
}

func (c *Core) Write(ctx context.Context, fr core.Frame) error {
	return c.writeFunc(ctx, fr)
}

func (c *Core) OnWrite(write func(ctx context.Context, fr core.Frame) error) {
	c.writeFunc = write
}

func (c *Core) evaluateStrata(ctx context.Context) {
	for stratum := 0; stratum <= c.module.Strata.Max; stratum++ {
		for nodeKey, node := range c.nodes {
			if c.module.Strata.Nodes[nodeKey] == stratum {
				node.Next(ctx)
			}
		}
	}
}

func (c *Core) createOutputHandler(nodeKey string) stage.OutputHandler {
	return func(ctx context.Context, sourceParam string, val value.Value) {
		for _, edge := range c.module.Edges {
			if edge.Source.Node == nodeKey && edge.Source.Param == sourceParam {
				targetNode := c.nodes[edge.Target.Node]
				targetNode.Load(edge.Target.Param, val)
			}
		}
	}
}
