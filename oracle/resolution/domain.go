// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package resolution

import (
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

// mergeDomainExpressions merges two domains' expressions.
// Parent expressions are kept, child expressions override on name conflict.
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
	Kind        ValueKind
	StringValue string
	IdentValue  string
	IntValue    int64
	FloatValue  float64
	BoolValue   bool
}
