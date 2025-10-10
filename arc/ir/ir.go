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
	"github.com/synnaxlabs/x/maps"
	"github.com/synnaxlabs/x/set"
)

type NamedTypes = maps.Ordered[string, Type]

const (
	DefaultOutputParam = "output"
	DefaultInputParam  = "input"
	LHSInputParam      = "a"
	RHSInputParam      = "b"
)

type Channels struct {
	Read  set.Set[uint32] `json:"read"`
	Write set.Set[uint32] `json:"write"`
}

func NewChannels() Channels {
	return Channels{
		Read:  make(set.Set[uint32]),
		Write: make(set.Set[uint32]),
	}
}

func OverrideChannels(other Channels) Channels {
	return Channels{
		Read:  lo.Ternary(other.Read != nil, other.Read, make(set.Set[uint32])),
		Write: lo.Ternary(other.Write != nil, other.Write, make(set.Set[uint32])),
	}
}

// Stage defines the structural type of a stage within the automation.
type Stage struct {
	// Key is a unique key identifying the type of the stage. This refers to the
	// stage name. We use a key field instead since it fits better with the structural
	// semantics of synnax.
	Key string `json:"key"`
	// Config are the names and types configuration parameters for the stage.
	Config NamedTypes `json:"config"`
	// Params are the names and type of the function parameters for the stage.
	Params NamedTypes `json:"params"`
	// Outputs are the names and types of return values for the stage.
	// Single anonymous returns create an output named "output".
	// Multi-output stages have explicit names.
	Outputs NamedTypes `json:"outputs,omitempty"`
	// StatefulVariables are names and types for the stateful variables on
	// the stage.
	StatefulVariables maps.Ordered[string, Type] `json:"stateful_variables"`
	//
	Channels Channels `json:"channels"`
	// Body is the logical body of the stage.
	Body Body
}

func (s Stage) String() string { return "stage" }

var _ Type = (*Stage)(nil)

// Node is an invocation of a stage within the flow graph.
type Node struct {
	// Key is a unique key for the node within the graph.
	Key string `json:"key"`
	// Type refers to the key of the stage type to use for the node.
	Type string `json:"type"`
	// Config are the configuration parameters to pass to the stage.
	Config map[string]any `json:"config"`
	// Channels are the channels that the stage needs access to.
	Channels Channels `json:"channels"`
	// Params are the resolved parameter types for this specific node instance.
	// For polymorphic stages, these are the concrete types after unification.
	Params NamedTypes `json:"params"`
	// Outputs are the resolved output types for this specific node instance.
	// For polymorphic stages, these are the concrete types after unification.
	Outputs NamedTypes `json:"outputs"`
}

// Handle is a connection point on a node.
type Handle struct {
	// Node is the key of the node being connected to.
	Node string `json:"node"`
	// Param is the parameter of the node being connected.
	Param string `json:"param"`
}

type Edge struct {
	Source Handle `json:"source"`
	Target Handle `json:"target"`
}

// ConstraintSystem is an interface for type constraint resolution
type ConstraintSystem interface {
	ApplySubstitutions(t Type) Type
	String() string
	HasTypeVariables() bool
	Unify() error
}

// Strata represents the execution stratification of a dataflow graph.
// Each node is assigned a stratum (execution level) based on its dependencies.
// Stratification enables single-pass, glitch-free reactive execution.
type Strata [][]string

func (s Strata) Get(key string) int {
	for i, nodes := range s {
		for _, node := range nodes {
			if node == key {
				return i
			}
		}
	}
	return -1
}

func (s Strata) Has(key string) bool {
	return s.Get(key) >= 0
}

func (s Strata) NodeCount() int {
	count := 0
	for _, nodes := range s {
		count += len(nodes)
	}
	return count
}

type IR struct {
	Stages      []Stage          `json:"stages"`
	Functions   []Function       `json:"functions"`
	Nodes       []Node           `json:"nodes"`
	Edges       []Edge           `json:"edges"`
	Symbols     *Scope           `json:"-"`
	Constraints ConstraintSystem `json:"-"`
	// Strata maps node keys to their execution stratum (level).
	// Computed during analysis for stratified reactive execution.
	Strata Strata `json:"-"`
}

func (ir IR) GetEdgeBySourceHandle(sourceHandle Handle) Edge {
	return lo.Must(lo.Find(ir.Edges, func(item Edge) bool {
		return item.Source == sourceHandle
	}))
}

func (ir IR) GetEdgeByTargetHandle(targetHandle Handle) Edge {
	return lo.Must(lo.Find(ir.Edges, func(item Edge) bool {
		return item.Target == targetHandle
	}))
}

func (ir IR) TryGetEdgeByTargetHandle(targetHandle Handle) (Edge, bool) {
	return lo.Find(ir.Edges, func(item Edge) bool {
		return item.Target == targetHandle
	})
}

func (ir IR) GetStage(key string) (Stage, bool) {
	return lo.Find(ir.Stages, func(item Stage) bool {
		return item.Key == key
	})
}

func (ir IR) GetNode(key string) (Node, bool) {
	return lo.Find(ir.Nodes, func(item Node) bool {
		return item.Key == key
	})
}

func (ir IR) ReadChannels() set.Set[uint32] {
	channels := make(set.Set[uint32])
	for _, node := range ir.Nodes {
		channels.Add(node.Channels.Read.Keys()...)
	}
	return channels
}

func (ir IR) WriteChannels() set.Set[uint32] {
	channels := make(set.Set[uint32])
	for _, node := range ir.Nodes {
		channels.Add(node.Channels.Write.Keys()...)
	}
	return channels
}
