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
	// Return are the names and types of the return values for the stage.
	Return Type `json:"return"`
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
}

type IR struct {
	Stages      []Stage          `json:"stages"`
	Functions   []Function       `json:"functions"`
	Nodes       []Node           `json:"nodes"`
	Edges       []Edge           `json:"edges"`
	Symbols     *Scope           `json:"-"`
	Constraints ConstraintSystem `json:"-"`
}

func (ir IR) GetStage(key string) (Stage, bool) {
	return lo.Find(ir.Stages, func(item Stage) bool {
		return item.Key == key
	})
}
