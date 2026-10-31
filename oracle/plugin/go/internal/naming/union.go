// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package naming

import (
	"github.com/samber/lo"
	"github.com/synnaxlabs/oracle/plugin/resolver"
	"github.com/synnaxlabs/oracle/resolution"
)

// variantField is the field of a Go union that holds the active variant. A method of
// the same name cannot exist.
const variantField = "Variant"

// UnionBase is a base that every variant of a union inherits whole.
type UnionBase struct {
	// Ref is the reference to the base in the union's extends list.
	Ref resolution.TypeRef
	// Type is the base.
	Type resolution.Type
	// Method is the method of the Go union that returns the base of the active variant.
	Method string
}

// UnionBases returns the bases that the Go union reads with a method, in extends order.
func UnionBases(form resolution.UnionForm, table *resolution.Table) []UnionBase {
	return lo.FilterMap(
		resolver.SharedBases(form, "go", table),
		func(ref resolution.TypeRef, _ int) (UnionBase, bool) {
			typ, ok := ref.Resolve(table)
			if !ok {
				return UnionBase{}, false
			}
			method := GetGoName(typ)
			return UnionBase{
				Ref:    ref,
				Type:   typ,
				Method: method,
			}, method != variantField
		},
	)
}

// UnionTagMethod returns the method of the Go union that returns the tag of the active
// variant. It returns false when the union has no such method.
func UnionTagMethod(form resolution.UnionForm) (string, bool) {
	method := ToPascalCase(form.Discriminator)
	return method, method != variantField
}
