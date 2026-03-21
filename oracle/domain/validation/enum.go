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

// EnumVariant contains resolved information about an enum variant reference
// from a @validate default expression.
type EnumVariant struct {
	Type    resolution.Type
	Variant resolution.EnumValue
}

// ResolveEnumVariant attempts to resolve an identifier-based default value as
// an enum variant. It uses the field's type reference to find the enum type,
// then matches the identifier against the enum's values.
//
// The identifier format is <Namespace>.<EnumName><VariantPascalCase> (e.g.,
// "control.ConcurrencyExclusive") or just <EnumName><VariantPascalCase> for
// same-namespace references.
func ResolveEnumVariant(
	identValue string,
	typeRef resolution.TypeRef,
	table *resolution.Table,
) (EnumVariant, bool) {
	resolved, ok := typeRef.Resolve(table)
	if !ok {
		return EnumVariant{}, false
	}
	enumForm, ok := resolved.Form.(resolution.EnumForm)
	if !ok {
		return EnumVariant{}, false
	}

	memberName := identValue
	if idx := strings.LastIndex(identValue, "."); idx >= 0 {
		memberName = identValue[idx+1:]
	}

	variantPascal := strings.TrimPrefix(memberName, resolved.Name)
	if variantPascal == memberName {
		return EnumVariant{}, false
	}

	variantLower := strings.ToLower(variantPascal)
	for _, v := range enumForm.Values {
		if strings.ToLower(v.Name) == variantLower {
			return EnumVariant{Type: resolved, Variant: v}, true
		}
	}

	return EnumVariant{}, false
}
