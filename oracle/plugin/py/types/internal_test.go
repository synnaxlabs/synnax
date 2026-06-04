// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package types

import (
	"testing"

	"github.com/synnaxlabs/oracle/resolution"
)

// TestStructDefaultToPython_UnresolvedTypeRef exercises the defensive `!ok` path:
// the analyzer never lets an object-literal default reach code generation with a
// type ref that isn't in the resolution table, but the helper must still bail
// safely if it ever did.
func TestStructDefaultToPython_UnresolvedTypeRef(t *testing.T) {
	table := resolution.NewTable()
	data := &templateData{Namespace: "log"}
	ev := resolution.ExpressionValue{
		Kind: resolution.ValueKindStruct,
		ObjectFields: []resolution.ObjectField{
			{Name: "format", Value: resolution.ExpressionValue{
				Kind:        resolution.ValueKindString,
				StringValue: "preciseDate",
			}},
		},
	}
	typeRef := resolution.TypeRef{Name: "log.DoesNotExist"}

	got := structDefaultToPython(ev, typeRef, table, data)
	if got != "" {
		t.Fatalf("expected empty string for unresolved type, got %q", got)
	}
}

// TestExpressionValueToPython covers every branch of the helper, including the
// default fallback that renders a non-scalar value as %q on an empty
// StringValue.
func TestExpressionValueToPython(t *testing.T) {
	cases := []struct {
		name string
		in   resolution.ExpressionValue
		want string
	}{
		{
			name: "string",
			in:   resolution.ExpressionValue{Kind: resolution.ValueKindString, StringValue: "hi"},
			want: `"hi"`,
		},
		{
			name: "int",
			in:   resolution.ExpressionValue{Kind: resolution.ValueKindInt, IntValue: 42},
			want: "42",
		},
		{
			name: "float",
			in:   resolution.ExpressionValue{Kind: resolution.ValueKindFloat, FloatValue: 1.5},
			want: "1.500000",
		},
		{
			name: "bool true",
			in:   resolution.ExpressionValue{Kind: resolution.ValueKindBool, BoolValue: true},
			want: "True",
		},
		{
			name: "bool false",
			in:   resolution.ExpressionValue{Kind: resolution.ValueKindBool, BoolValue: false},
			want: "False",
		},
		{
			name: "ident falls through to empty-string fallback",
			in:   resolution.ExpressionValue{Kind: resolution.ValueKindIdent, IdentValue: "Level.high"},
			want: `""`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := expressionValueToPython(tc.in)
			if got != tc.want {
				t.Fatalf("expressionValueToPython(%+v) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
