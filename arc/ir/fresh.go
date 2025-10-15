// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ir

import "github.com/synnaxlabs/arc/types"

// FreshType creates a fresh copy of t with renamed type variables. This is used
// during polymorphic function instantiation to avoid type variable conflicts. The
// prefix is prepended to type variable names, and the transformation is applied
// recursively to compound types.
func FreshType(t types.Type, prefix string) types.Type {
	if t.Kind == types.KindTypeVariable {
		freshConstraint := t.Constraint
		if freshConstraint != nil {
			fresh := FreshType(*freshConstraint, prefix)
			freshConstraint = &fresh
		}
		return types.Type{
			Kind:       types.KindTypeVariable,
			Name:       prefix + "_" + t.Name,
			Constraint: freshConstraint,
		}
	}
	if t.Kind == types.KindChan || t.Kind == types.KindSeries {
		ft := FreshType(*t.ValueType, prefix)
		return types.Type{Kind: t.Kind, ValueType: &ft}
	}
	if t.Kind == types.KindFunction {
		return types.Type{
			Kind:               types.KindFunction,
			FunctionProperties: t.FunctionProperties.Copy(),
		}
	}
	return t
}
