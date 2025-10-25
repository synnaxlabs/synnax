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
	Raw string
	// AST is the parsed abstract syntax tree from the parser.
	AST antlr.ParserRuleContext
}

// Function represents a function or stage definition in the IR. Functions are
// templates that can be instantiated as nodes in the dataflow graph.
type Function struct {
	Key      string          `json:"key"`
	Body     Body            `json:"body"`
	Config   types.Params    `json:"config"`
	Inputs   types.Params    `json:"inputs"`
	Outputs  types.Params    `json:"outputs"`
	Channels symbol.Channels `json:"channels"`
}

// Type returns the type signature of f.
func (f Function) Type() types.Type {
	return types.Function(types.FunctionProperties{
		Config:  &f.Config,
		Inputs:  &f.Inputs,
		Outputs: &f.Outputs,
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
