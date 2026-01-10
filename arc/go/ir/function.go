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

	"github.com/samber/lo"
	"github.com/synnaxlabs/arc/types"
)

// Type returns the type signature of f.
func (f Function) Type() types.Type {
	return types.Function(types.FunctionProperties{
		Config:  f.Config,
		Inputs:  f.Inputs,
		Outputs: f.Outputs,
	})
}

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
