// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ir

import (
	"github.com/samber/lo"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
)

// Node represents an instantiated function or stage in the dataflow graph.
// Each node is a concrete instance of a Function with specific configuration values.
type Node struct {
	// Key is the unique identifier for this node instance.
	Key string `json:"key"`
	// Type is the name of the function or stage this node instantiates.
	Type string `json:"type"`
	// ConfigValues contains the runtime configuration values for this node.
	ConfigValues map[string]any `json:"config_values"`
	// Channels contains references to external channels used by this node.
	Channels symbol.Channels `json:"channels"`
	// Config contains the type definitions of configuration parameters.
	Config types.Params `json:"config"`
	// Inputs contains the type definitions of input parameters.
	Inputs types.Params `json:"inputs"`
	// Outputs contains the type definitions of output parameters.
	Outputs types.Params `json:"outputs"`
}

// Nodes is a collection of node instances.
type Nodes []Node

// Find searches for a node by key. Returns the node and true if found,
// or zero value and false otherwise.
func (n Nodes) Find(key string) (Node, bool) {
	return lo.Find(n, func(n Node) bool { return n.Key == key })
}

// Get returns the node with the given key. Panics if not found.
func (n Nodes) Get(key string) Node { return lo.Must(n.Find(key)) }
