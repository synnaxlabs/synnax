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

// Sort is a list order that a field's `@index sorted` mark declares. A mark with a
// value names a sub-field of a struct field: `@index sorted start` on `time_range`
// orders by `time_range.start`.
type Sort struct {
	// Name names the order on the wire: the field name, then the sub-field name, joined
	// by an underscore.
	Name string
	// Path holds the field, then the sub-field if the mark names one.
	Path []Field
	// Type is the type of the value that the order compares.
	Type TypeRef
	// Text is true when the value is a string, which orders by its lower case.
	Text bool
}

// Leaf returns the field whose value the order compares.
func (s Sort) Leaf() Field { return s.Path[len(s.Path)-1] }

// Sorts returns the orders that the `@index sorted` marks of fields declare, in field
// order. It returns an error for a mark that names no sub-field of a struct, or that
// orders a value that is neither a string nor a number.
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
		sort := Sort{Name: field.Name, Path: []Field{field}, Type: field.Type}
		if len(expr.Values) > 0 {
			sub, err := subField(field, expr.Values[0], table)
			if err != nil {
				return nil, err
			}
			sort.Name += "_" + sub.Name
			sort.Path = append(sort.Path, sub)
			sort.Type = sub.Type
		}
		if sort.Leaf().Optional {
			return nil, errors.Newf(
				"%s: an optional field cannot be sorted",
				sort.path(),
			)
		}
		switch base := PrimitiveBase(sort.Type, table); {
		case base == "string":
			sort.Text = true
		case IsNumberPrimitive(base):
		default:
			return nil, errors.Newf(
				"%s: only a string or a number can be sorted",
				sort.path(),
			)
		}
		sorts = append(sorts, sort)
	}
	return sorts, nil
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
