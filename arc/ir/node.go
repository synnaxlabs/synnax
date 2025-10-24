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

// Node represents an instantiated function or stage in the dataflow graph. Each node
// is an instance of a Function with concrete configuration values.
type Node struct {
	Key          string          `json:"key"`
	Type         string          `json:"type"`
	ConfigValues map[string]any  `json:"config_values"`
	Channels     symbol.Channels `json:"channels"`
	Config       types.Params    `json:"config"`
	Inputs       types.Params    `json:"inputs"`
	Outputs      types.Params    `json:"outputs"`
}

// Nodes is a collection of node instances.
type Nodes []Node

// Get returns the node with the given key. Panics if not found.
func (n Nodes) Get(key string) Node {
	return lo.Must(lo.Find(n, func(n Node) bool { return n.Key == key }))
}
