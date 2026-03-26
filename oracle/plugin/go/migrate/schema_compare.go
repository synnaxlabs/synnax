// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package migrate

import "github.com/synnaxlabs/oracle/resolution"

// schemasEqual returns true if two types have the same data shape, meaning
// existing serialized data is compatible without transformation. It compares
// field names, order, types, and optionality recursively through the type graph.
// Annotations, docs, and code generation hints are ignored.
func schemasEqual(
	oldType, newType resolution.Type,
	oldTable, newTable *resolution.Table,
) bool {
	return typesEqual(oldType, newType, oldTable, newTable, make(map[string]bool))
}

func typesEqual(
	old, new resolution.Type,
	oldTable, newTable *resolution.Table,
	visiting map[string]bool,
) bool {
	if visiting[old.QualifiedName] {
		return true // recursive cycle, assume equal
	}
	visiting[old.QualifiedName] = true
	defer delete(visiting, old.QualifiedName)

	switch oldForm := old.Form.(type) {
	case resolution.StructForm:
		newForm, ok := new.Form.(resolution.StructForm)
		if !ok {
			return false
		}
		return structsEqual(oldForm, newForm, oldTable, newTable, visiting)
	case resolution.EnumForm:
		newForm, ok := new.Form.(resolution.EnumForm)
		if !ok {
			return false
		}
		return enumsEqual(oldForm, newForm)
	case resolution.AliasForm:
		newForm, ok := new.Form.(resolution.AliasForm)
		if !ok {
			return false
		}
		return refsEqual(oldForm.Target, newForm.Target, oldTable, newTable, visiting)
	case resolution.DistinctForm:
		newForm, ok := new.Form.(resolution.DistinctForm)
		if !ok {
			return false
		}
		return refsEqual(oldForm.Base, newForm.Base, oldTable, newTable, visiting)
	case resolution.PrimitiveForm:
		newForm, ok := new.Form.(resolution.PrimitiveForm)
		if !ok {
			return false
		}
		return oldForm.Name == newForm.Name
	case resolution.BuiltinGenericForm:
		newForm, ok := new.Form.(resolution.BuiltinGenericForm)
		if !ok {
			return false
		}
		return oldForm.Name == newForm.Name
	default:
		return false
	}
}

func structsEqual(
	old, new resolution.StructForm,
	oldTable, newTable *resolution.Table,
	visiting map[string]bool,
) bool {
	if len(old.Fields) != len(new.Fields) {
		return false
	}
	for i := range old.Fields {
		if !fieldsEqual(old.Fields[i], new.Fields[i], oldTable, newTable, visiting) {
			return false
		}
	}
	if len(old.Extends) != len(new.Extends) {
		return false
	}
	for i := range old.Extends {
		if !refsEqual(old.Extends[i], new.Extends[i], oldTable, newTable, visiting) {
			return false
		}
	}
	return true
}

func fieldsEqual(
	old, new resolution.Field,
	oldTable, newTable *resolution.Table,
	visiting map[string]bool,
) bool {
	if old.Name != new.Name {
		return false
	}
	if old.IsOptional != new.IsOptional {
		return false
	}
	if old.IsHardOptional != new.IsHardOptional {
		return false
	}
	return refsEqual(old.Type, new.Type, oldTable, newTable, visiting)
}

func enumsEqual(old, new resolution.EnumForm) bool {
	if old.IsIntEnum != new.IsIntEnum {
		return false
	}
	if len(old.Values) != len(new.Values) {
		return false
	}
	for i := range old.Values {
		if old.Values[i].Name != new.Values[i].Name {
			return false
		}
		if old.Values[i].Value != new.Values[i].Value {
			return false
		}
	}
	return true
}

func refsEqual(
	old, new resolution.TypeRef,
	oldTable, newTable *resolution.Table,
	visiting map[string]bool,
) bool {
	if len(old.TypeArgs) != len(new.TypeArgs) {
		return false
	}
	for i := range old.TypeArgs {
		if !refsEqual(old.TypeArgs[i], new.TypeArgs[i], oldTable, newTable, visiting) {
			return false
		}
	}
	// Compare array sizes (fixed-size arrays).
	oldHasSize := old.ArraySize != nil
	newHasSize := new.ArraySize != nil
	if oldHasSize != newHasSize {
		return false
	}
	if oldHasSize && *old.ArraySize != *new.ArraySize {
		return false
	}
	// Resolve and compare the underlying types.
	oldResolved, oldOk := old.Resolve(oldTable)
	newResolved, newOk := new.Resolve(newTable)
	if oldOk != newOk {
		return false
	}
	if !oldOk {
		return old.Name == new.Name
	}
	return typesEqual(oldResolved, newResolved, oldTable, newTable, visiting)
}
