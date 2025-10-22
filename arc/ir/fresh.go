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

func FreshType(t types.Type, prefix string) types.Type {
	return freshTypeWithMap(t, prefix, make(map[string]types.Type))
}

func freshTypeWithMap(t types.Type, prefix string, mapping map[string]types.Type) types.Type {
	if t.Kind == types.KindTypeVariable {
		if cached, ok := mapping[t.Name]; ok {
			return cached
		}
		freshConstraint := t.Constraint
		if freshConstraint != nil {
			fresh := freshTypeWithMap(*freshConstraint, prefix, mapping)
			freshConstraint = &fresh
		}
		freshVar := types.Type{
			Kind:       types.KindTypeVariable,
			Name:       prefix + "_" + t.Name,
			Constraint: freshConstraint,
		}
		mapping[t.Name] = freshVar
		return freshVar
	}
	if t.Kind == types.KindChan || t.Kind == types.KindSeries {
		ft := freshTypeWithMap(t.Unwrap(), prefix, mapping)
		return types.Type{Kind: t.Kind, ValueType: &ft}
	}
	if t.Kind == types.KindFunction {
		props := types.FunctionProperties{
			Inputs:  &types.Params{},
			Outputs: &types.Params{},
			Config:  &types.Params{},
		}
		if t.Inputs != nil {
			for k, v := range t.Inputs.Iter() {
				props.Inputs.Put(k, freshTypeWithMap(v, prefix, mapping))
			}
		}
		if t.Outputs != nil {
			for k, v := range t.Outputs.Iter() {
				props.Outputs.Put(k, freshTypeWithMap(v, prefix, mapping))
			}
		}
		if t.Config != nil {
			for k, v := range t.Config.Iter() {
				props.Config.Put(k, freshTypeWithMap(v, prefix, mapping))
			}
		}
		return types.Type{Kind: types.KindFunction, FunctionProperties: props}
	}
	return t
}
