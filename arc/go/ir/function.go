// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ir

import (
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/samber/lo"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
)

// Body represents the source code and AST of a function or stage implementation.
// It contains both the original textual representation and the parsed abstract
// syntax tree for code generation and analysis.
type Body struct {
	// Raw is the original source code as written by the user.
	Raw string `json:"raw" msgpack:"raw"`
	// AST is the parsed abstract syntax tree from the parser.
	AST antlr.ParserRuleContext `json:"-" msgpack:"-"`
}

// Function represents a function or stage definition in the IR. Functions are
// templates that can be instantiated as nodes in the dataflow graph.
type Function struct {
	// Key is the unique identifier for this function.
	Key string `json:"key"`
	// Body contains the source code and AST of the function implementation.
	Body Body `json:"body"`
	// Config contains the type definitions of configuration parameters.
	Config types.Params `json:"config"`
	// Inputs contains the type definitions of input parameters.
	Inputs types.Params `json:"inputs"`
	// Outputs contains the type definitions of output parameters.
	Outputs types.Params `json:"outputs"`
	// Channels contains references to external channels used by this function.
	Channels symbol.Channels `json:"channels"`
}

// Type returns the type signature of f.
func (f Function) Type() types.Type {
	return types.Function(types.FunctionProperties{
		Config:  f.Config,
		Inputs:  f.Inputs,
		Outputs: f.Outputs,
	})
}

// Functions is a collection of function definitions.
type Functions []Function

// Get returns the function with the given key. Panics if not found.
func (f Functions) Get(key string) Function {
	return lo.Must(f.Find(key))
}

// Find searches for a function by key. Returns the function and true if found,
// or zero value and false otherwise.
func (f Functions) Find(key string) (Function, bool) {
	return lo.Find(f, func(fn Function) bool { return fn.Key == key })
}

// String returns the string representation of the function.
func (f Function) String() string {
	return f.stringWithPrefix("")
}

// stringWithPrefix returns the string representation with tree formatting.
func (f Function) stringWithPrefix(prefix string) string {
	var b strings.Builder
	b.WriteString(f.Key)
	b.WriteString("\n")

	hasConfig := len(f.Config) > 0
	hasInputs := len(f.Inputs) > 0
	hasOutputs := len(f.Outputs) > 0

	// Channels
	isLast := !hasConfig && !hasInputs && !hasOutputs
	b.WriteString(prefix)
	b.WriteString(treePrefix(isLast))
	b.WriteString("channels: ")
	b.WriteString(formatChannels(f.Channels))
	b.WriteString("\n")

	if hasConfig {
		isLast = !hasInputs && !hasOutputs
		b.WriteString(prefix)
		b.WriteString(treePrefix(isLast))
		b.WriteString("config: ")
		b.WriteString(formatParams(f.Config))
		b.WriteString("\n")
	}

	if hasInputs {
		isLast = !hasOutputs
		b.WriteString(prefix)
		b.WriteString(treePrefix(isLast))
		b.WriteString("inputs: ")
		b.WriteString(formatParams(f.Inputs))
		b.WriteString("\n")
	}

	if hasOutputs {
		b.WriteString(prefix)
		b.WriteString(treePrefix(true))
		b.WriteString("outputs: ")
		b.WriteString(formatParams(f.Outputs))
		b.WriteString("\n")
	}

	return b.String()
}
