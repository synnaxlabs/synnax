// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package compiler

import "github.com/synnaxlabs/slate/graph"

// Module represents the compiled output with WASM bytecode and metadata
type Module struct {
	Version   string       `json:"version"`
	Module    []byte       `json:"module"`    // Base64-encoded WASM bytes
	Tasks     []TaskSpec   `json:"tasks"`     // Task specifications
	Functions []FuncSpec   `json:"functions"` // Function specifications
	Nodes     []graph.Node `json:"nodes"`     // Graph nodes (including virtual)
	Edges     []graph.Edge `json:"edges"`     // Graph edges
}

// TaskSpec describes a task's interface
type TaskSpec struct {
	Name         string        `json:"name"`
	Key          string        `json:"key"`
	Config       []ParamSpec   `json:"config"`
	Args         []ParamSpec   `json:"args"`
	StatefulVars []StateVar    `json:"stateful_vars"`
	Return       *string       `json:"return"`
}

// FuncSpec describes a function's interface
type FuncSpec struct {
	Name   string      `json:"name"`
	Key    string      `json:"key"`
	Params []ParamSpec `json:"params"`
	Return *string     `json:"return"`
}

// ParamSpec describes a parameter
type ParamSpec struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// StateVar describes a stateful variable
type StateVar struct {
	Name string `json:"name"`
	Type string `json:"type"`
	Key  int    `json:"key"`
}