// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package event

import (
	"context"

	"github.com/synnaxlabs/synnax/pkg/service/slate/spec"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
	"github.com/synnaxlabs/x/errors"
)

var factories = []factory{
	newConstant,
	newComparison,
	newTelemSource,
	newTelemSink,
	newSelectStatement,
	newAnnotationCreate,
}

type factoryConfig struct {
	spec.Config
	pipeline *plumber.Pipeline
	node     spec.Node
	graph    spec.Graph
}

type factory = func(context.Context, factoryConfig) (bool, error)

func create(ctx context.Context, cfg factoryConfig) error {
	for _, f := range factories {
		if ok, err := f(ctx, cfg); err != nil || ok {
			return err
		}
	}
	return errors.New("could not find node for")
}

func Create(ctx context.Context, cfg spec.Config, g spec.Graph) (confluence.Flow, error) {
	p := plumber.New()
	nodeMap := make(map[string]spec.Node, len(g.Nodes))
	for _, n := range g.Nodes {
		if err := create(ctx, factoryConfig{
			Config:   cfg,
			graph:    g,
			node:     n,
			pipeline: p,
		}); err != nil {
			return nil, err
		}
		nodeMap[n.Key] = n
	}
	for _, e := range g.Edges {
		sourceF, _ := plumber.GetSource[spec.Value](p, address.Address(e.Source.Node))
		sourceNode := nodeMap[e.Source.Node]
		sinkF, _ := plumber.GetSink[spec.Value](p, address.Address(e.Sink.Node))
		sinkNode := nodeMap[e.Sink.Node]
		stream := confluence.NewStream[spec.Value](1)
		output, _ := sourceNode.Schema.GetOutput(e.Source.Key)
		input, _ := sinkNode.Schema.GetInput(e.Sink.Key)
		stream.SetInletAddress(address.Address(output.Key))
		stream.SetOutletAddress(address.Address(input.Key))
		sourceF.OutTo(stream)
		sinkF.InFrom(stream)
	}
	return p, nil
}
