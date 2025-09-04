// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package module

// Task represents a task definition with its signature
type Task struct {
	Key      string              // WASM export name (same as Name for user-defined)
	Config   map[string]string   // Config parameter names -> types
	Params   map[string]string   // Runtime parameter names -> types
	Returns  string              // Return type (empty if void)
	Stateful map[string]Variable // Stateful variables
}

// Function represents a function definition with its signature
type Function struct {
	Key     string            // WASM export name (same as Name)
	Params  map[string]string // Parameter names -> types
	Returns string            // Return type (empty if void)
}

// Variable represents a stateful variable in a task
type Variable struct {
	Type  string
	Index uint32 // Used internally for state persistence
}

// Node represents a task instance in the flow graph
type Node struct {
	Key    string         // Unique instance identifier
	Type   string         // References Task.Key or special types like "channel"
	Config map[string]any // Configuration values
}

// Handle identifies a specific port on a node
type Handle struct {
	Node  string // Node.Key
	Param string // Port name (e.g., "output", "input", or param name)
}

// Edge represents a connection between nodes in the flow graph
type Edge struct {
	Source Handle
	Target Handle
}

// Module represents a compiled Slate program's structure
type Module struct {
	Tasks     []Task     // Task definitions
	Functions []Function // Function definitions
	Nodes     []Node     // Node instances
	Edges     []Edge     // Connections between nodes
	Wasm      []byte
}
