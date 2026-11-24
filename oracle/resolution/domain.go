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
	"reflect"

	"github.com/antlr4-go/antlr/v4"
	"github.com/samber/lo"
	"github.com/synnaxlabs/oracle/parser"
)

// Domain represents a domain annotation on a struct or field.
// The AST can be IDomainContext, IInlineDomainContext, or IFileDomainContext.
type Domain struct {
	AST         antlr.ParserRuleContext
	Name        string
	Expressions Expressions
}

func (d Domain) Merge(parent Domain) Domain {
	merged := Domain{AST: d.AST, Name: d.Name}
	exprMap := make(map[string]Expression)
	for _, expr := range parent.Expressions {
		exprMap[expr.Name] = expr
	}
	for _, expr := range d.Expressions {
		exprMap[expr.Name] = expr
	}
	for _, expr := range parent.Expressions {
		if e, ok := exprMap[expr.Name]; ok {
			merged.Expressions = append(merged.Expressions, e)
			delete(exprMap, expr.Name)
		}
	}
	for _, expr := range d.Expressions {
		if e, ok := exprMap[expr.Name]; ok {
			merged.Expressions = append(merged.Expressions, e)
			delete(exprMap, expr.Name)
		}
	}
	return merged
}

// Expression represents a single expression within a domain.
type Expression struct {
	AST    parser.IExpressionContext
	Name   string
	Values []ExpressionValue
}

type Expressions []Expression

func (e Expressions) Find(name string) (Expression, bool) {
	return lo.Find(e, func(item Expression) bool {
		return item.Name == name
	})
}

// ExpressionValue holds a parsed value from a domain expression.
type ExpressionValue struct {
	StringValue string
	IdentValue  string
	IntValue    int64
	FloatValue  float64
	Kind        ValueKind
	BoolValue   bool
	// Duration is the duration literal an int value was written as, such as 1m30s.
	// IntValue holds its nanoseconds. It is empty for a plain integer.
	Duration string
	// Elements holds the ordered element values when Kind is ValueKindArray.
	Elements []ExpressionValue
	// Fields holds the ordered field values when Kind is ValueKindStruct.
	Fields []StructFieldValue
}

// SameValue reports whether a and b hold the same value, however a duration literal
// spelled it. A nil value equals only another nil value.
func SameValue(a, b *ExpressionValue) bool {
	if a == nil || b == nil {
		return a == b
	}
	x, y := *a, *b
	x.Duration, y.Duration = "", ""
	return reflect.DeepEqual(x, y)
}

// StructFieldValue is a single `name = value` binding within a struct default
// literal. Name is the unqualified field name as written in the schema.
type StructFieldValue struct {
	Name  string
	Value ExpressionValue
}
