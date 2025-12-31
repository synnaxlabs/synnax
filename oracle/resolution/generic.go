// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package resolution provides the resolution table for Oracle plugins.
package resolution

type TypeParam struct {
	Name       string
	Optional   bool
	Constraint *TypeRef
	Default    *TypeRef
}

func substituteTypeRef(tr *TypeRef, typeArgMap map[string]*TypeRef) *TypeRef {
	if tr == nil {
		return nil
	}
	if tr.Kind == TypeKindTypeParam && tr.TypeParamRef != nil {
		if concrete, ok := typeArgMap[tr.TypeParamRef.Name]; ok {
			result := &TypeRef{
				Kind:           concrete.Kind,
				Primitive:      concrete.Primitive,
				StructRef:      concrete.StructRef,
				EnumRef:        concrete.EnumRef,
				TypeParamRef:   concrete.TypeParamRef,
				IsArray:        tr.IsArray || concrete.IsArray,
				IsOptional:     tr.IsOptional || concrete.IsOptional,
				IsHardOptional: tr.IsHardOptional || concrete.IsHardOptional,
				RawType:        concrete.RawType,
				MapKeyType:     concrete.MapKeyType,
				MapValueType:   concrete.MapValueType,
			}
			if len(concrete.TypeArgs) > 0 {
				result.TypeArgs = make([]*TypeRef, len(concrete.TypeArgs))
				for i, arg := range concrete.TypeArgs {
					result.TypeArgs[i] = substituteTypeRef(arg, typeArgMap)
				}
			}
			return result
		}
	}
	needsSubstitution := false
	for _, arg := range tr.TypeArgs {
		if arg.Kind == TypeKindTypeParam {
			needsSubstitution = true
			break
		}
	}
	if tr.Kind == TypeKindMap {
		if tr.MapKeyType != nil && tr.MapKeyType.Kind == TypeKindTypeParam {
			needsSubstitution = true
		}
		if tr.MapValueType != nil && tr.MapValueType.Kind == TypeKindTypeParam {
			needsSubstitution = true
		}
	}
	if !needsSubstitution && len(tr.TypeArgs) == 0 {
		return tr
	}

	result := &TypeRef{
		Kind:           tr.Kind,
		Primitive:      tr.Primitive,
		StructRef:      tr.StructRef,
		EnumRef:        tr.EnumRef,
		TypeParamRef:   tr.TypeParamRef,
		IsArray:        tr.IsArray,
		IsOptional:     tr.IsOptional,
		IsHardOptional: tr.IsHardOptional,
		RawType:        tr.RawType,
	}

	if len(tr.TypeArgs) > 0 {
		result.TypeArgs = make([]*TypeRef, len(tr.TypeArgs))
		for i, arg := range tr.TypeArgs {
			result.TypeArgs[i] = substituteTypeRef(arg, typeArgMap)
		}
	}

	if tr.Kind == TypeKindMap {
		result.MapKeyType = substituteTypeRef(tr.MapKeyType, typeArgMap)
		result.MapValueType = substituteTypeRef(tr.MapValueType, typeArgMap)
	}

	return result
}
