// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package resolution

import (
	"strings"

	"github.com/samber/lo"
	"github.com/synnaxlabs/x/errors"
)

// Sort is a list order that a field's `@index sorted` mark declares. Each value of a
// mark names a sub-field of a struct field and declares its own order: `@index sorted
// start end` on `time_range` orders by `time_range.start` and by `time_range.end`.
type Sort struct {
	// Name names the order on the wire: the field name, then the sub-field name, joined
	// by an underscore.
	Name string
	// Path holds the field, then the sub-field if the mark names one.
	Path []Field
	// Type is the type of the value that the order compares.
	Type TypeRef
	// Textual is true when the value is a string, which orders by its lower case.
	Textual bool
}

// Leaf returns the field whose value the order compares.
func (s Sort) Leaf() Field { return s.Path[len(s.Path)-1] }

// Sorts returns the orders that the `@index sorted` marks of fields declare, in field
// order. It returns an error for a mark that names no sub-field of a struct, that
// orders a value that is neither a string nor a number, or that shares its field with
// `@index lookup`.
func Sorts(fields []Field, table *Table) ([]Sort, error) {
	var sorts []Sort
	for _, field := range fields {
		dom, ok := field.Domains["index"]
		if !ok {
			continue
		}
		expr, ok := dom.Expressions.Find("sorted")
		if !ok {
			continue
		}
		if _, ok := dom.Expressions.Find("lookup"); ok {
			return nil, errors.Newf(
				"%s: a sorted index answers exact matches, so drop @index lookup",
				field.Name,
			)
		}
		if len(expr.Values) == 0 {
			sort, err := newSort(Sort{Name: field.Name, Path: []Field{field}}, table)
			if err != nil {
				return nil, err
			}
			sorts = append(sorts, sort)
			continue
		}
		for _, value := range expr.Values {
			sub, err := subField(field, value, table)
			if err != nil {
				return nil, err
			}
			sort, err := newSort(Sort{
				Name: field.Name + "_" + sub.Name,
				Path: []Field{field, sub},
			}, table)
			if err != nil {
				return nil, err
			}
			sorts = append(sorts, sort)
		}
	}
	return sorts, nil
}

// newSort returns sort with the type and textual flag of its leaf.
func newSort(sort Sort, table *Table) (Sort, error) {
	leaf := sort.Leaf()
	if leaf.Optional {
		return Sort{}, errors.Newf(
			"%s: an optional field cannot be sorted",
			sort.path(),
		)
	}
	sort.Type = leaf.Type
	switch base := PrimitiveBase(sort.Type, table); {
	case base == "string":
		sort.Textual = true
	case IsNumberPrimitive(base):
	default:
		return Sort{}, errors.Newf(
			"%s: only a string or a number can be sorted",
			sort.path(),
		)
	}
	return sort, nil
}

func (s Sort) path() string {
	return strings.Join(lo.Map(s.Path, func(f Field, _ int) string {
		return f.Name
	}), ".")
}

func subField(field Field, value ExpressionValue, table *Table) (Field, error) {
	name := value.IdentValue
	if value.Kind == ValueKindString {
		name = value.StringValue
	}
	typ, _, ok := Dealias(field.Type, table)
	if !ok {
		return Field{}, errors.Newf(
			"%s: @index sorted %s needs a struct",
			field.Name,
			name,
		)
	}
	fields, ok := RecordFields(typ, table)
	if !ok {
		return Field{}, errors.Newf(
			"%s: @index sorted %s needs a struct",
			field.Name,
			name,
		)
	}
	sub, ok := lo.Find(fields, func(f Field) bool { return f.Name == name })
	if !ok {
		return Field{}, errors.Newf(
			"%s: %s has no field %s",
			field.Name,
			typ.Name,
			name,
		)
	}
	return sub, nil
}
