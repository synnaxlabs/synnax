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

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/schema"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

type Node struct {
	Key    string
	Type   string
	Data   map[string]interface{}
	Schema *NodeSchema
}

func (n Node) String() string {
	return n.Key
}

type Handle struct {
	Node string
	Key  string
}

type Edge struct {
	Source Handle
	Sink   Handle
}

type Graph struct {
	Nodes []Node
	Edges []Edge
}

type Input struct {
	Key             string
	AcceptsDataType func(dt string) bool
}

type Output struct {
	Key      string
	DataType string
}

type NodeSchema struct {
	Type    string
	Inputs  []Input
	Outputs []Output
	Data    map[string]schema.Field
}

func (n NodeSchema) GetOutput(key string) (Output, bool) {
	return lo.Find(n.Outputs, func(item Output) bool { return item.Key == key })
}

func (n NodeSchema) GetInput(key string) (Input, bool) {
	return lo.Find(n.Inputs, func(item Input) bool { return item.Key == key })
}

type Value struct {
	DataType string
	Value    interface{}
}

func strictlyMatchDataType(expected string) func(input string) bool {
	return func(input string) bool {
		return expected == input
	}
}

func acceptsNumericDataType(input string) bool {
	return !telem.DataType(input).IsVariable()
}

type SchemaMatcher = func(context.Context, Config, Node) (NodeSchema, bool, error)

func constant(_ context.Context, _ Config, n Node) (ns NodeSchema, ok bool, err error) {
	if n.Type != "constant" {
		return ns, false, err
	}
	fields := map[string]schema.Field{
		"data_type": {
			Type: schema.String,
		},
	}
	dt, ok := schema.Get[string](schema.Resource{Data: n.Data}, "data_type")
	if !ok {
		return ns, true, errors.WithStack(validate.FieldError{
			Field:   "data_type",
			Message: "invalid data type",
		})
	}
	fields["value"] = schema.Field{Type: schema.FieldType(dt)}
	ns.Outputs = []Output{
		{
			Key:      "value",
			DataType: dt,
		},
	}
	ns.Data = fields
	ns.Type = "constant"
	return ns, true, nil
}

func greaterThan(_ context.Context, _ Config, n Node) (ns NodeSchema, ok bool, err error) {
	if n.Type != "comparison.ge" {
		return ns, false, err
	}
	ns.Inputs = []Input{
		{
			Key:             "x",
			AcceptsDataType: acceptsNumericDataType,
		},
		{
			Key:             "y",
			AcceptsDataType: acceptsNumericDataType,
		},
	}
	ns.Outputs = []Output{
		{
			Key:      "value",
			DataType: "uint8",
		},
	}
	ns.Type = "comparison.ge"
	return ns, true, nil
}

func telemSource(ctx context.Context, cfg Config, n Node) (ns NodeSchema, ok bool, err error) {
	if n.Type != "telem_source" {
		return ns, false, err
	}
	chKey, ok := schema.Get[uint32](schema.Resource{Data: n.Data}, "channel")
	if !ok {
		return ns, true, errors.WithStack(validate.FieldError{
			Field:   "channel",
			Message: "invalid channel",
		})
	}
	var ch channel.Channel
	if err = cfg.Channel.NewRetrieve().
		WhereKeys(channel.Key(chKey)).
		Entry(&ch).
		Exec(ctx, nil); err != nil {
		return ns, ok, err
	}
	ns.Outputs = []Output{
		{
			Key:      "value",
			DataType: string(ch.DataType),
		},
	}
	ns.Type = "telem_source"
	return ns, true, nil
}

func telemSink(ctx context.Context, cfg Config, n Node) (ns NodeSchema, ok bool, err error) {
	if n.Type != "telem_sink" {
		return ns, false, err
	}
	chKey, ok := schema.Get[uint32](schema.Resource{Data: n.Data}, "channel")
	if !ok {
		return ns, true, errors.WithStack(validate.FieldError{})
	}
	var ch channel.Channel
	if err = cfg.Channel.NewRetrieve().WhereKeys(channel.Key(chKey)).Entry(&ch).Exec(ctx, nil); err != nil {
		return ns, ok, err
	}
	ns.Inputs = []Input{
		{
			Key:             "value",
			AcceptsDataType: strictlyMatchDataType(string(ch.DataType)),
		},
	}
	ns.Type = "telem_sink"
	return ns, true, nil
}

var schemaMatchers = []SchemaMatcher{
	constant,
	greaterThan,
	telemSource,
	telemSink,
}

type Config struct {
	Channel channel.Service
	Framer  *framer.Service
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
				"output not found for source node %s", sourceNode,
			)
		}
		input, ok := sinkNode.Schema.GetInput(e.Sink.Key)
		if !ok {
			return g, errors.Wrapf(
				query.NotFound,
				"input %s not found for sink node %s", e.Sink.Key, sinkNode,
			)
		}
		if !input.AcceptsDataType(output.DataType) {
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
