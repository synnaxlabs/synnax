// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package module

import (
	"fmt"
	"strings"
)

// Task represents a task definition with its signature
type Task struct {
	Key      string                      `json:"key"`      // WASM export name (same as Name for user-defined)
	Config   map[string]string           `json:"config"`   // Config parameter names -> types
	Params   map[string]string           `json:"params"`   // Runtime parameter names -> types
	Returns  string                      `json:"returns"`  // Return type (empty if void)
	Stateful map[string]StatefulVariable `json:"stateful"` // Stateful variables
	Channels struct {
		Read  []string `json:"read"`
		Write []string `json:"write"`
	} `json:"channels"`
}

// Function represents a function definition with its signature
type Function struct {
	Key     string            `json:"key"`     // WASM export name (same as Name)
	Params  map[string]string `json:"params"`  // Parameter names -> types
	Returns string            `json:"returns"` // Return type (empty if void)
}

// StatefulVariable represents a stateful variable in a task
type StatefulVariable struct {
	Type string `json:"type"`
	ID   uint32 `json:"index"` // Used internally for state persistence
}

// Node represents a task instance in the flow graph
type Node struct {
	Key    string         `json:"key"`    // Unique instance identifier
	Type   string         `json:"type"`   // References Task.Key or special types like "channel"
	Config map[string]any `json:"config"` // Configuration values
}

// Handle identifies a specific port on a node
type Handle struct {
	Node  string `json:"node"`  // Node.Key
	Param string `json:"param"` // Port name (e.g., "output", "input", or param name)
}

// Edge represents a connection between nodes in the flow graph
type Edge struct {
	Source Handle `json:"source"`
	Target Handle `json:"target"`
}

// Module represents a compiled arc program's structure
type Module struct {
	Tasks     []Task     `json:"tasks"`     // Task definitions
	Functions []Function `json:"functions"` // Function definitions
	Nodes     []Node     `json:"nodes"`     // Node instances
	Edges     []Edge     `json:"edges"`     // Connections between nodes
	Wasm      []byte     `json:"wasm"`
}

// String returns a pretty-printed representation of the StatefulVariable
func (v StatefulVariable) String() string {
	return fmt.Sprintf("type: %s, index: %d", v.Type, v.ID)
}

// String returns a pretty-printed representation of the Task
func (t Task) String() string {
	return t.stringWithIndent("")
}

func (t Task) stringWithIndent(indent string) string {
	b := strings.Builder{}
	b.WriteString(indent)
	b.WriteString("task: ")
	b.WriteString(t.Key)
	b.WriteString("\n")

	if len(t.Config) > 0 {
		b.WriteString(indent)
		b.WriteString("config:\n")
		for name, typ := range t.Config {
			b.WriteString(indent)
			b.WriteString("  ")
			b.WriteString(name)
			b.WriteString(": ")
			b.WriteString(typ)
			b.WriteString("\n")
		}
	}

	if len(t.Params) > 0 {
		b.WriteString(indent)
		b.WriteString("params:\n")
		for name, typ := range t.Params {
			b.WriteString(indent)
			b.WriteString("  ")
			b.WriteString(name)
			b.WriteString(": ")
			b.WriteString(typ)
			b.WriteString("\n")
		}
	}

	if t.Returns != "" {
		b.WriteString(indent)
		b.WriteString("returns: ")
		b.WriteString(t.Returns)
		b.WriteString("\n")
	}

	if len(t.Stateful) > 0 {
		b.WriteString(indent)
		b.WriteString("stateful:\n")
		for name, v := range t.Stateful {
			b.WriteString(indent)
			b.WriteString("  ")
			b.WriteString(name)
			b.WriteString(": ")
			b.WriteString(v.String())
			b.WriteString("\n")
		}
	}

	return b.String()
}

// String returns a pretty-printed representation of the Function
func (f Function) String() string {
	return f.stringWithIndent("")
}

func (f Function) stringWithIndent(indent string) string {
	b := strings.Builder{}
	b.WriteString(indent)
	b.WriteString("function: ")
	b.WriteString(f.Key)
	b.WriteString("\n")

	if len(f.Params) > 0 {
		b.WriteString(indent)
		b.WriteString("params:\n")
		for name, typ := range f.Params {
			b.WriteString(indent)
			b.WriteString("  ")
			b.WriteString(name)
			b.WriteString(": ")
			b.WriteString(typ)
			b.WriteString("\n")
		}
	}

	if f.Returns != "" {
		b.WriteString(indent)
		b.WriteString("returns: ")
		b.WriteString(f.Returns)
		b.WriteString("\n")
	}

	return b.String()
}

// String returns a pretty-printed representation of the Node
func (n Node) String() string {
	return n.stringWithIndent("")
}

func (n Node) stringWithIndent(indent string) string {
	b := strings.Builder{}
	b.WriteString(indent)
	b.WriteString("node: ")
	b.WriteString(n.Key)
	b.WriteString("\n")
	b.WriteString(indent)
	b.WriteString("type: ")
	b.WriteString(n.Type)
	b.WriteString("\n")

	if len(n.Config) > 0 {
		b.WriteString(indent)
		b.WriteString("config:\n")
		for name, val := range n.Config {
			b.WriteString(indent)
			b.WriteString("  ")
			b.WriteString(name)
			b.WriteString(": ")
			b.WriteString(fmt.Sprintf("%v", val))
			b.WriteString("\n")
		}
	}

	return b.String()
}

// String returns a pretty-printed representation of the Handle
func (h Handle) String() string {
	return fmt.Sprintf("%s.%s", h.Node, h.Param)
}

// String returns a pretty-printed representation of the Edge
func (e Edge) String() string {
	return fmt.Sprintf("%s -> %s", e.Source.String(), e.Target.String())
}

// String returns a pretty-printed representation of the Module
func (m Module) String() string {
	b := strings.Builder{}
	b.WriteString("module:\n")

	if len(m.Tasks) > 0 {
		b.WriteString("tasks:\n")
		for _, task := range m.Tasks {
			b.WriteString(task.stringWithIndent("  "))
			b.WriteString("  ---\n")
		}
	}

	if len(m.Functions) > 0 {
		b.WriteString("functions:\n")
		for _, fn := range m.Functions {
			b.WriteString(fn.stringWithIndent("  "))
			b.WriteString("  ---\n")
		}
	}

	if len(m.Nodes) > 0 {
		b.WriteString("nodes:\n")
		for _, node := range m.Nodes {
			b.WriteString(node.stringWithIndent("  "))
			b.WriteString("  ---\n")
		}
	}

	if len(m.Edges) > 0 {
		b.WriteString("edges:\n")
		for _, edge := range m.Edges {
			b.WriteString("  ")
			b.WriteString(edge.String())
			b.WriteString("\n")
		}
	}

	if len(m.Wasm) > 0 {
		b.WriteString(fmt.Sprintf("wasm: %d bytes\n", len(m.Wasm)))
	}

	return b.String()
}
