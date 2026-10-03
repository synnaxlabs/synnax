// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package validation

import (
	"strings"

	"github.com/synnaxlabs/oracle/resolution"
)

// UnionVariant contains resolved information about a union variant reference from
// a field's default expression.
type UnionVariant struct {
	Type    resolution.Type
	Union   resolution.UnionForm
	Variant resolution.UnionVariant
}

// ResolveUnionVariant attempts to resolve an identifier-based default value as a
// union variant. It uses the field's type reference to find the union type, then
// matches the identifier against the union's variants.
//
// The identifier is the bare variant name (`= built_in`), optionally namespace
// qualified (`= ni.built_in`), or the generated UnionName-prefixed form
// (`= CJCBuiltIn`), mirroring ResolveEnumVariant.
func ResolveUnionVariant(
	identValue string,
	typeRef resolution.TypeRef,
	table *resolution.Table,
) (UnionVariant, bool) {
	resolved, ok := typeRef.Resolve(table)
	if !ok {
		return UnionVariant{}, false
	}
	unionForm, ok := resolved.Form.(resolution.UnionForm)
	if !ok {
		return UnionVariant{}, false
	}

	memberName := identValue
	if _, bare, ok := strings.CutLast(identValue, "."); ok {
		memberName = bare
	}

	if variant, ok := unionForm.Variant(memberName); ok {
		return UnionVariant{Type: resolved, Union: unionForm, Variant: variant}, true
	}

	// Fallback: the generated UnionName-prefixed form, e.g. `= CJCBuiltIn` for
	// union CJC variant `built_in`. Underscores collapse when PascalCasing, so
	// compare with them stripped.
	variantPascal := strings.TrimPrefix(memberName, resolved.Name)
	if variantPascal == memberName {
		return UnionVariant{}, false
	}
	target := strings.ToLower(variantPascal)
	for _, v := range unionForm.Variants {
		if strings.ToLower(strings.ReplaceAll(v.Name, "_", "")) == target {
			return UnionVariant{Type: resolved, Union: unionForm, Variant: v}, true
		}
	}

	return UnionVariant{}, false
}
