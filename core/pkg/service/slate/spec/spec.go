// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package spec

import (
	"context"
	"fmt"

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/service/annotation"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/status"
	"github.com/synnaxlabs/x/validate"
	"github.com/synnaxlabs/x/zyn"
)

// Node is a node in the arc graph. Nodes can be thought of as functions/expressions
// that receive inputs and produce outputs.
type Node struct {
	// Key is a unique key for the node.
	Key string `json:"key" msgpack:"key"`
	// Type is the type of the node.
	Type   string                 `json:"type" msgpack:"type"`
	Config map[string]interface{} `json:"config" msgpack:"config"`
	Schema *NodeSchema            `json:"schema" msgpack:"schema"`
}

func (n Node) String() string {
	return fmt.Sprintf("%s<%s>", n.Key, n.Type)
}

type Handle struct {
	Node string `json:"node" msgpack:"node"`
	Key  string `json:"key" msgpack:"key"`
}

type Edge struct {
	Source Handle `json:"source" msgpack:"source"`
	Sink   Handle `json:"sink" msgpack:"sink"`
}

type Graph struct {
	Nodes []Node `json:"nodes" msgpack:"nodes"`
	Edges []Edge `json:"edges" msgpack:"edges"`
}

func (g Graph) FindEdge(match func(item Edge) bool) (Edge, bool) {
	return lo.Find(g.Edges, match)
}

type Input struct {
	Key             string
	AcceptsDataType zyn.Schema
}

type Output struct {
	Key      string
	DataType zyn.DataType
}

type NodeSchema struct {
	Type    string
	Inputs  []Input
	Outputs []Output
	Config  zyn.Schema
}

func (n NodeSchema) GetOutput(key string) (Output, bool) {
	return lo.Find(n.Outputs, func(item Output) bool { return item.Key == key })
}

func (n NodeSchema) GetInput(key string) (Input, bool) {
	return lo.Find(n.Inputs, func(item Input) bool { return item.Key == key })
}

type Value struct {
	DataType zyn.DataType
	Value    any
}

type SchemaMatcher = func(context.Context, Config, Node) (NodeSchema, bool, error)

var schemaMatchers = []SchemaMatcher{
	constant,
	operator,
	telemSource,
	telemSink,
	selectStatement,
	createStatusChanger,
	stableFor,
	rangeCreator,
}

type Config struct {
	Channel        channel.Service
	Framer         *framer.Service
	Ranger         *ranger.Service
	Annotation     *annotation.Service
	OnStatusChange func(ctx context.Context, status status.Status[any])
}

func Validate(ctx context.Context, cfg Config, g Graph) (Graph, error) {
	nodeMap := make(map[string]Node)
	for i, n := range g.Nodes {
		nodeSchema, err := GetSchema(ctx, cfg, n)
		if err != nil {
			return g, err
		}
		n.Schema = &nodeSchema
		g.Nodes[i] = n
		nodeMap[n.Key] = n
	}
	for _, e := range g.Edges {
		sourceNode, ok := nodeMap[e.Source.Node]
		if !ok {
			return g, errors.Wrapf(
				query.NotFound,
				"source node with key %s not found for edge", e.Source.Node,
			)
		}
		sinkNode, ok := nodeMap[e.Sink.Node]
		if !ok {
			return g, errors.Wrapf(
				query.NotFound,
				"sink node with key %s not found for edge", e.Sink.Node,
			)
		}
		output, ok := sourceNode.Schema.GetOutput(e.Source.Key)
		if !ok {
			return g, errors.Wrapf(
				query.NotFound,
				"output %s not found on source node %s", e.Source.Key, sourceNode,
			)
		}
		input, ok := sinkNode.Schema.GetInput(e.Sink.Key)
		if !ok {
			return g, errors.Wrapf(
				query.NotFound,
				"input %s not found for sink node %s", e.Sink.Key, sinkNode,
			)
		}
		if input.AcceptsDataType.Validate(output.DataType) != nil {
			return g, errors.Wrapf(
				validate.Error,
				"output %s.%s with data type %s is not acceptable for input %s.%s",
				sourceNode,
				e.Source.Key,
				output.DataType,
				sinkNode,
				e.Sink.Key,
			)
		}
	}
	return g, nil
}

func GetSchema(ctx context.Context, cfg Config, n Node) (ns NodeSchema, err error) {
	for _, matcher := range schemaMatchers {
		if ns, ok, err := matcher(ctx, cfg, n); ok {
			return ns, err
		}
	}
	return ns, errors.Wrapf(query.NotFound, "no node found for type %s", n)
}
