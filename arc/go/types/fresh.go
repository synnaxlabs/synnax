// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package types

import (
	"slices"
)

// Freshen creates a copy of t with all type variables renamed using the given prefix.
// This is essential when instantiating generic functions to avoid type variable conflicts
// during unification.
//
// For example, when instantiating a generic function `add<T>(a: T, b: T) -> T` twice,
// each instantiation needs fresh type variables (e.g., "node1_T" and "node2_T") to
// prevent unification from incorrectly constraining both instantiations to the same type.
//
// The function recursively freshens type variables in:
//   - Direct type variables (including their constraints)
//   - Channel and series element types
//   - Function input, output, and config parameters
//
// Primitive types (i64, f64, string, etc.) are returned unchanged.
func Freshen(t Type, prefix string) Type {
	return freshen(t, prefix, make(map[string]Type))
}

func freshenParams(
	params Params,
	prefix string,
	mapping map[string]Type,
) Params {
	fresh := slices.Clone(params)
	for i, v := range params {
		fresh[i].Type = freshen(v.Type, prefix, mapping)
	}
	return fresh
}

func freshen(t Type, prefix string, mapping map[string]Type) Type {
	if t.Kind == KindVariable {
		if cached, ok := mapping[t.Name]; ok {
			return cached
		}
		freshConstraint := t.Constraint
		if freshConstraint != nil {
			fresh := freshen(*freshConstraint, prefix, mapping)
			freshConstraint = &fresh
		}
		freshVar := Type{
			Kind:       KindVariable,
			Name:       prefix + "_" + t.Name,
			Constraint: freshConstraint,
		}
		mapping[t.Name] = freshVar
		return freshVar
	}
	if t.Kind == KindChan || t.Kind == KindSeries {
		ft := freshen(t.Unwrap(), prefix, mapping)
		return Type{Kind: t.Kind, ValueType: &ft}
	}
	if t.Kind == KindFunction {
		props := FunctionProperties{
			Inputs:  freshenParams(t.Inputs, prefix, mapping),
			Outputs: freshenParams(t.Outputs, prefix, mapping),
			Config:  freshenParams(t.Config, prefix, mapping),
		}
		return Type{Kind: KindFunction, FunctionProperties: props}
	}
	return t
}
