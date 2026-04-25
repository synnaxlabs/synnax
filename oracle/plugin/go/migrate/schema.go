// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package migrate

import (
	"github.com/synnaxlabs/oracle/plugin/domain"
	"github.com/synnaxlabs/oracle/plugin/output"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/set"
)

// --- Schema equality (used by detectSchemaChange) ---

// schemasEqual returns true if two types have the same data shape. It compares
// field names, order, types, and optionality recursively through the type graph.
func schemasEqual(
	oldType, newType resolution.Type,
	oldTable, newTable *resolution.Table,
) bool {
	return typesEqual(oldType, newType, oldTable, newTable, make(set.Set[string]))
}

func typesEqual(
	old, new resolution.Type,
	oldTable, newTable *resolution.Table,
	visiting set.Set[string],
) bool {
	if visiting.Contains(old.QualifiedName) {
		return true
	}
	visiting.Add(old.QualifiedName)
	defer visiting.Remove(old.QualifiedName)

	switch oldForm := old.Form.(type) {
	case resolution.StructForm:
		newForm, ok := new.Form.(resolution.StructForm)
		if !ok {
			return false
		}
		if len(oldForm.Fields) != len(newForm.Fields) || len(oldForm.Extends) != len(newForm.Extends) {
			return false
		}
		for i := range oldForm.Fields {
			of, nf := oldForm.Fields[i], newForm.Fields[i]
			if of.Name != nf.Name || of.IsOptional != nf.IsOptional || of.IsHardOptional != nf.IsHardOptional {
				return false
			}
			if !refsEqual(of.Type, nf.Type, oldTable, newTable, visiting) {
				return false
			}
			if domain.GetStringFromField(of, "go", "marshal") !=
				domain.GetStringFromField(nf, "go", "marshal") {
				return false
			}
		}
		for i := range oldForm.Extends {
			if !refsEqual(oldForm.Extends[i], newForm.Extends[i], oldTable, newTable, visiting) {
				return false
			}
		}
		return true
	case resolution.EnumForm:
		newForm, ok := new.Form.(resolution.EnumForm)
		if !ok || oldForm.IsIntEnum != newForm.IsIntEnum || len(oldForm.Values) != len(newForm.Values) {
			return false
		}
		for i := range oldForm.Values {
			if oldForm.Values[i].Name != newForm.Values[i].Name || oldForm.Values[i].Value != newForm.Values[i].Value {
				return false
			}
		}
		return true
	case resolution.AliasForm:
		newForm, ok := new.Form.(resolution.AliasForm)
		return ok && refsEqual(oldForm.Target, newForm.Target, oldTable, newTable, visiting)
	case resolution.DistinctForm:
		newForm, ok := new.Form.(resolution.DistinctForm)
		return ok && refsEqual(oldForm.Base, newForm.Base, oldTable, newTable, visiting)
	case resolution.PrimitiveForm:
		newForm, ok := new.Form.(resolution.PrimitiveForm)
		return ok && oldForm.Name == newForm.Name
	case resolution.BuiltinGenericForm:
		newForm, ok := new.Form.(resolution.BuiltinGenericForm)
		return ok && oldForm.Name == newForm.Name
	default:
		return false
	}
}

func refsEqual(
	old, new resolution.TypeRef,
	oldTable, newTable *resolution.Table,
	visiting set.Set[string],
) bool {
	if len(old.TypeArgs) != len(new.TypeArgs) {
		return false
	}
	for i := range old.TypeArgs {
		if !refsEqual(old.TypeArgs[i], new.TypeArgs[i], oldTable, newTable, visiting) {
			return false
		}
	}
	oldHasSize, newHasSize := old.ArraySize != nil, new.ArraySize != nil
	if oldHasSize != newHasSize || (oldHasSize && *old.ArraySize != *new.ArraySize) {
		return false
	}
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

// --- Schema diff (used by auto-copy generation) ---

type TypeChangeKind int

const (
	TypeUnchanged         TypeChangeKind = iota
	TypeChanged                          // fields added/removed/modified
	TypeDescendantChanged                // own fields unchanged, nested type changed
)

type TypeDiff struct {
	QualifiedName string
	GoPath        string
	Kind          TypeChangeKind
	ChangedFields []FieldDiff
}

type FieldDiffKind int

const (
	FieldKindUnchanged FieldDiffKind = iota
	FieldKindAdded
	FieldKindRemoved
	FieldKindTypeChanged
	FieldKindOptionalityChanged
)

type FieldDiff struct {
	Name     string
	Kind     FieldDiffKind
	OldField *resolution.Field
	NewField *resolution.Field
}

// SchemaDiff walks old and new entry types and produces a TypeDiff for every
// Oracle-defined type that changed or has changed descendants.
func SchemaDiff(
	oldEntry, newEntry resolution.Type,
	oldTable, newTable *resolution.Table,
) map[string]TypeDiff {
	result := make(map[string]TypeDiff)
	diffWalk(oldEntry, newEntry, oldTable, newTable, result, make(set.Set[string]))
	return result
}

func diffWalk(
	old, new resolution.Type,
	oldTable, newTable *resolution.Table,
	result map[string]TypeDiff,
	visiting set.Set[string],
) TypeChangeKind {
	if visiting.Contains(old.QualifiedName) {
		return TypeUnchanged
	}
	if existing, ok := result[old.QualifiedName]; ok {
		return existing.Kind
	}
	visiting.Add(old.QualifiedName)
	defer visiting.Remove(old.QualifiedName)

	goPath := output.GetPath(old, "go")

	switch oldForm := old.Form.(type) {
	case resolution.AliasForm:
		if k := diffRefWalk(oldForm.Target, oldTable, newTable, result, visiting); k != TypeUnchanged {
			result[old.QualifiedName] = TypeDiff{QualifiedName: old.QualifiedName, GoPath: goPath, Kind: TypeDescendantChanged}
			return TypeDescendantChanged
		}
		return TypeUnchanged
	case resolution.DistinctForm:
		if k := diffRefWalk(oldForm.Base, oldTable, newTable, result, visiting); k != TypeUnchanged {
			result[old.QualifiedName] = TypeDiff{QualifiedName: old.QualifiedName, GoPath: goPath, Kind: TypeDescendantChanged}
			return TypeDescendantChanged
		}
		return TypeUnchanged
	}

	oldStruct, oldOk := old.Form.(resolution.StructForm)
	newStruct, newOk := new.Form.(resolution.StructForm)
	if !oldOk || !newOk {
		if !typesEqual(old, new, oldTable, newTable, make(set.Set[string])) {
			result[old.QualifiedName] = TypeDiff{QualifiedName: old.QualifiedName, GoPath: goPath, Kind: TypeChanged}
			return TypeChanged
		}
		return TypeUnchanged
	}

	fieldDiffs, selfChanged := diffStructFields(oldStruct, newStruct, oldTable, newTable)

	hasDescendantChange := false
	for _, f := range oldStruct.Fields {
		if diffRefWalk(f.Type, oldTable, newTable, result, visiting) != TypeUnchanged {
			hasDescendantChange = true
		}
	}
	for _, ext := range oldStruct.Extends {
		if diffRefWalk(ext, oldTable, newTable, result, visiting) != TypeUnchanged {
			hasDescendantChange = true
		}
	}

	if selfChanged {
		result[old.QualifiedName] = TypeDiff{
			QualifiedName: old.QualifiedName, GoPath: goPath,
			Kind: TypeChanged, ChangedFields: fieldDiffs,
		}
		return TypeChanged
	}
	if hasDescendantChange {
		result[old.QualifiedName] = TypeDiff{QualifiedName: old.QualifiedName, GoPath: goPath, Kind: TypeDescendantChanged}
		return TypeDescendantChanged
	}
	return TypeUnchanged
}

func diffStructFields(
	old, new resolution.StructForm,
	oldTable, newTable *resolution.Table,
) (diffs []FieldDiff, selfChanged bool) {
	newByName := make(map[string]resolution.Field, len(new.Fields))
	for _, f := range new.Fields {
		newByName[f.Name] = f
	}
	oldByName := make(map[string]resolution.Field, len(old.Fields))
	for _, f := range old.Fields {
		oldByName[f.Name] = f
	}
	if len(old.Fields) != len(new.Fields) {
		selfChanged = true
	} else {
		for i := range old.Fields {
			if old.Fields[i].Name != new.Fields[i].Name {
				selfChanged = true
				break
			}
		}
	}
	for _, of := range old.Fields {
		nf, exists := newByName[of.Name]
		if !exists {
			diffs = append(diffs, FieldDiff{Name: of.Name, Kind: FieldKindRemoved, OldField: &of})
			selfChanged = true
			continue
		}
		if of.IsOptional != nf.IsOptional || of.IsHardOptional != nf.IsHardOptional {
			diffs = append(diffs, FieldDiff{Name: of.Name, Kind: FieldKindOptionalityChanged, OldField: &of, NewField: &nf})
			selfChanged = true
			continue
		}
		if !refsIdentityEqual(of.Type, nf.Type, oldTable, newTable) {
			diffs = append(diffs, FieldDiff{Name: of.Name, Kind: FieldKindTypeChanged, OldField: &of, NewField: &nf})
			selfChanged = true
			continue
		}
		if domain.GetStringFromField(of, "go", "marshal") !=
			domain.GetStringFromField(nf, "go", "marshal") {
			selfChanged = true
		}
		diffs = append(diffs, FieldDiff{Name: of.Name, Kind: FieldKindUnchanged, OldField: &of, NewField: &nf})
	}
	for _, nf := range new.Fields {
		if _, exists := oldByName[nf.Name]; !exists {
			diffs = append(diffs, FieldDiff{Name: nf.Name, Kind: FieldKindAdded, NewField: &nf})
			selfChanged = true
		}
	}
	return diffs, selfChanged
}

// refsIdentityEqual checks if two type references point to the same type by
// qualified name (not deep structural comparison).
func refsIdentityEqual(old, new resolution.TypeRef, oldTable, newTable *resolution.Table) bool {
	if len(old.TypeArgs) != len(new.TypeArgs) {
		return false
	}
	for i := range old.TypeArgs {
		if !refsIdentityEqual(old.TypeArgs[i], new.TypeArgs[i], oldTable, newTable) {
			return false
		}
	}
	oldHasSize, newHasSize := old.ArraySize != nil, new.ArraySize != nil
	if oldHasSize != newHasSize || (oldHasSize && *old.ArraySize != *new.ArraySize) {
		return false
	}
	oldResolved, oldOk := old.Resolve(oldTable)
	newResolved, newOk := new.Resolve(newTable)
	if oldOk != newOk {
		return false
	}
	if !oldOk {
		return old.Name == new.Name
	}
	return oldResolved.QualifiedName == newResolved.QualifiedName
}

func diffRefWalk(
	ref resolution.TypeRef,
	oldTable, newTable *resolution.Table,
	result map[string]TypeDiff,
	visiting set.Set[string],
) TypeChangeKind {
	oldResolved, oldOk := ref.Resolve(oldTable)
	if !oldOk {
		return TypeUnchanged
	}
	newResolved, newOk := newTable.Get(oldResolved.QualifiedName)
	if !newOk {
		return TypeUnchanged
	}
	kind := diffWalk(oldResolved, newResolved, oldTable, newTable, result, visiting)
	for _, arg := range ref.TypeArgs {
		if argKind := diffRefWalk(arg, oldTable, newTable, result, visiting); argKind != TypeUnchanged && kind == TypeUnchanged {
			kind = TypeDescendantChanged
		}
	}
	return kind
}
