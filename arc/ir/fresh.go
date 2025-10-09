// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ir

import "github.com/synnaxlabs/x/maps"

// FreshType creates a fresh instance of a type with unique type variables.
//
// This implements type variable instantiation as required by Hindley-Milner type
// systems. When a polymorphic type is used, each use site must get its own
// independent type variables to prevent unwanted unification.
//
// Parameters:
//   - t: The type to freshen (may contain type variables)
//   - prefix: A unique prefix to prepend to type variable names (typically the node key)
//
// Returns:
//   - A new type with all type variables renamed to include the prefix
//
// Example:
//   original: T:numeric
//   FreshType(original, "add_node1") -> add_node1_T:numeric
//   FreshType(original, "add_node2") -> add_node2_T:numeric
func FreshType(t Type, prefix string) Type {
	if tv, ok := t.(TypeVariable); ok {
		// Create a fresh type variable with a unique name
		return TypeVariable{
			Name:       prefix + "_" + tv.Name,
			Constraint: tv.Constraint,
		}
	}

	// Recursively freshen compound types
	if ch, ok := t.(Chan); ok {
		return Chan{ValueType: FreshType(ch.ValueType, prefix)}
	}

	if s, ok := t.(Series); ok {
		return Series{ValueType: FreshType(s.ValueType, prefix)}
	}

	// Concrete types don't need freshening
	return t
}

// FreshStage creates a fresh instance of a stage with unique type variables.
//
// This is critical for polymorphic stages: each node instance must have its own
// type variables to prevent unwanted unification between separate uses of the stage.
//
// Without fresh instantiation, two nodes using the same polymorphic stage would
// be forced to have identical types, breaking polymorphism.
//
// Parameters:
//   - stage: The stage definition to instantiate
//   - nodeKey: Unique identifier for this node instance
//
// Returns:
//   - A new stage with fresh type variables in all params, outputs, and config
//
// Example:
//   stage: add<T>(a: T, b: T) -> T
//   FreshStage(stage, "add1") -> add<add1_T>(a: add1_T, b: add1_T) -> add1_T
//   FreshStage(stage, "add2") -> add<add2_T>(a: add2_T, b: add2_T) -> add2_T
func FreshStage(stage Stage, nodeKey string) Stage {
	freshParams := &maps.Ordered[string, Type]{}
	for k, v := range stage.Params.Iter() {
		freshParams.Put(k, FreshType(v, nodeKey))
	}

	freshOutputs := &maps.Ordered[string, Type]{}
	for k, v := range stage.Outputs.Iter() {
		freshOutputs.Put(k, FreshType(v, nodeKey))
	}

	freshConfig := &maps.Ordered[string, Type]{}
	for k, v := range stage.Config.Iter() {
		freshConfig.Put(k, FreshType(v, nodeKey))
	}

	return Stage{
		Key:               stage.Key,
		Config:            *freshConfig,
		Params:            *freshParams,
		Outputs:           *freshOutputs,
		StatefulVariables: stage.StatefulVariables,
		Channels:          stage.Channels,
		Body:              stage.Body,
	}
}

// FreshFunction creates a fresh instance of a function with unique type variables.
//
// Similar to FreshStage, but for function types.
//
// Parameters:
//   - fn: The function definition to instantiate
//   - callKey: Unique identifier for this function call
//
// Returns:
//   - A new function with fresh type variables in all params and outputs
func FreshFunction(fn Function, callKey string) Function {
	freshParams := &maps.Ordered[string, Type]{}
	for k, v := range fn.Params.Iter() {
		freshParams.Put(k, FreshType(v, callKey))
	}

	freshOutputs := &maps.Ordered[string, Type]{}
	for k, v := range fn.Outputs.Iter() {
		freshOutputs.Put(k, FreshType(v, callKey))
	}

	return Function{
		Key:     fn.Key,
		Params:  *freshParams,
		Outputs: *freshOutputs,
		Body:    fn.Body,
	}
}
