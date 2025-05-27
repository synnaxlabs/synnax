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

// Node is a node in the slate graph. Nodes can be thought of as functions/expressions
// that receive inputs and produce outputs.
type Node struct {
	// Key is a unique key for the node.
	Key string `json:"key" msgpack:"key"`
	// Type is the type of the node.
	Type   string                 `json:"type" msgpack:"type"`
	Data   map[string]interface{} `json:"data" msgpack:"data"`
	Schema *NodeSchema            `json:"schema" msgpack:"schema"`
}

func (n Node) String() string {
	return n.Key
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

func greaterThanEq(_ context.Context, _ Config, n Node) (ns NodeSchema, ok bool, err error) {
	if n.Type != "operator.gte" {
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
	ns.Type = "operator.gte"
	return ns, true, nil
}

func telemSource(ctx context.Context, cfg Config, n Node) (ns NodeSchema, ok bool, err error) {
	if n.Type != "source" {
		return ns, false, err
	}
	chKey, ok := schema.Get[float64](schema.Resource{Data: n.Data}, "channel")
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
	ns.Type = "source"
	return ns, true, nil
}

func telemSink(ctx context.Context, cfg Config, n Node) (ns NodeSchema, ok bool, err error) {
	if n.Type != "sink" {
		return ns, false, err
	}
	chKey, ok := schema.Get[float64](schema.Resource{Data: n.Data}, "channel")
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
	ns.Type = "sink"
	return ns, true, nil
}

func selectStatement(_ context.Context, _ Config, n Node) (ns NodeSchema, ok bool, err error) {
	if n.Type != "select" {
		return ns, false, err
	}
	ns.Inputs = []Input{
		{
			Key:             "value",
			AcceptsDataType: strictlyMatchDataType("uint8"),
		},
	}
	ns.Outputs = []Output{
		{
			Key:      "true",
			DataType: "uint8",
		},
		{
			Key:      "false",
			DataType: "uint8",
		},
	}
	ns.Type = "select"
	return ns, true, nil
}

func sendNotification(_ context.Context, cfg Config, n Node) (ns NodeSchema, ok bool, err error) {
	if n.Type != "send_notification" {
		return ns, false, err
	}
	ns.Inputs = []Input{
		{
			Key:             "value",
			AcceptsDataType: strictlyMatchDataType("uint8"),
		},
	}
	ns.Outputs = []Output{}
	return ns, true, nil
}

var schemaMatchers = []SchemaMatcher{
	constant,
	greaterThanEq,
	telemSource,
	telemSink,
	selectStatement,
	sendNotification,
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
