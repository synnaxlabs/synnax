// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package codes declares the diagnostic kinds of the Arc compiler.
package codes

//go:generate go run gen.go

import (
	"github.com/synnaxlabs/x/diagnostics"
	"go.lsp.dev/protocol"
)

// Reserved codes. The compiler does not report them.
const (
	// SymbolUndefined indicates a referenced symbol was not found.
	SymbolUndefined diagnostics.ErrorCode = "ARC4001"
	// SymbolRedefined indicates a symbol was declared more than once in the same scope.
	SymbolRedefined diagnostics.ErrorCode = "ARC4002"
)

var (
	// Syntax is source text that the parser rejects.
	Syntax = diagnostics.Kind{
		Code:     "E1000",
		Severity: protocol.DiagnosticSeverityError,
	}
	// SourceLimit is source text that is too large or too deeply nested to parse.
	SourceLimit = diagnostics.Kind{
		Code:     "E1001",
		Severity: protocol.DiagnosticSeverityError,
	}
	// InvalidStringLiteral is a malformed string literal or format placeholder.
	InvalidStringLiteral = diagnostics.Kind{
		Code:     "E1002",
		Severity: protocol.DiagnosticSeverityError,
	}
	// SubtractionSpacing is a binary minus with no space on one of its sides.
	SubtractionSpacing = diagnostics.Kind{
		Code:     "E1003",
		Severity: protocol.DiagnosticSeverityError,
		Message:  "subtraction requires whitespace on both sides of '-'",
	}

	// DivisionByZero is a constant integer division or modulo by zero.
	DivisionByZero = diagnostics.Kind{
		Code:     "E2000",
		Severity: protocol.DiagnosticSeverityError,
		Message:  "integer %s by zero",
	}
	// TypeMismatch is a value of one type where the code needs a different type.
	TypeMismatch = diagnostics.Kind{
		Code:     "E2001",
		Severity: protocol.DiagnosticSeverityError,
	}
	// UnknownType is a type name or a unit name that does not exist.
	UnknownType = diagnostics.Kind{
		Code:     "E2002",
		Severity: protocol.DiagnosticSeverityError,
	}
	// UnresolvedType is a value that has no type the compiler can determine.
	UnresolvedType = diagnostics.Kind{
		Code:     "E2003",
		Severity: protocol.DiagnosticSeverityError,
	}
	// InvalidOperand is an operator or statement on a type that it does not support.
	InvalidOperand = diagnostics.Kind{
		Code:     "E2005",
		Severity: protocol.DiagnosticSeverityError,
	}
	// InvalidCast is a cast between two types that Arc does not convert.
	InvalidCast = diagnostics.Kind{
		Code:     "E2006",
		Severity: protocol.DiagnosticSeverityError,
	}
	// NotConstant is a computed value where the code needs a constant.
	NotConstant = diagnostics.Kind{
		Code:     "E2007",
		Severity: protocol.DiagnosticSeverityError,
	}
	// ConstantOverflow is a constant that does not fit its type.
	ConstantOverflow = diagnostics.Kind{
		Code:     "E2008",
		Severity: protocol.DiagnosticSeverityError,
		Message:  "constant %s overflows %s",
	}
	// DimensionMismatch is an operation on units that have different dimensions.
	DimensionMismatch = diagnostics.Kind{
		Code:     "E2009",
		Severity: protocol.DiagnosticSeverityError,
	}
	// NaNComparison is a comparison with a NaN constant, which always has one result.
	NaNComparison = diagnostics.Kind{
		Code:     "E2010",
		Severity: protocol.DiagnosticSeverityError,
		Message:  "%s with %s is always %t, use %s instead",
	}
	// InvalidLiteral is a number literal that is malformed or has no exact value.
	InvalidLiteral = diagnostics.Kind{
		Code:     "E2011",
		Severity: protocol.DiagnosticSeverityError,
	}
	// TruthyCondition is a number or a string used as a condition.
	TruthyCondition = diagnostics.Kind{
		Code:     "W2000",
		Severity: protocol.DiagnosticSeverityWarning,
		Message:  "non-boolean conditions are deprecated, use an explicit comparison like x != 0",
	}
	// BoolCast is a cast of a bool to a number type other than u8.
	BoolCast = diagnostics.Kind{
		Code:     "W2001",
		Severity: protocol.DiagnosticSeverityWarning,
		Message:  "bool to %s is experimental; a future release may only allow bool to u8",
	}
	// UnitPrecisionLoss is a unit operation that can lose precision or overflow.
	UnitPrecisionLoss = diagnostics.Kind{
		Code:     "W2003",
		Severity: protocol.DiagnosticSeverityWarning,
	}

	// FuncArgMissing is a call with no argument for a required parameter.
	FuncArgMissing = diagnostics.Kind{
		Code:     "ARC3001",
		Severity: protocol.DiagnosticSeverityError,
		Message:  "missing required argument for parameter '%s' of func '%s'",
	}
	// FuncArgExtra is a call with more arguments than parameters.
	FuncArgExtra = diagnostics.Kind{
		Code:     "ARC3003",
		Severity: protocol.DiagnosticSeverityError,
		Message:  "too many arguments for func '%s': expected at most %d",
	}

	// DeprecatedSymbol is a reference to a name that has a replacement.
	DeprecatedSymbol = diagnostics.Kind{
		Code:     "ARC4003",
		Severity: protocol.DiagnosticSeverityWarning,
		Message:  "'%s' is deprecated, use '%s' instead",
	}
	// DeclarationAfterStep is a variable declared after a step of its scope.
	DeclarationAfterStep = diagnostics.Kind{
		Code:     "ARC4004",
		Severity: protocol.DiagnosticSeverityWarning,
		Message:  "declare variables at the top of a stage or sequence",
	}

	// UnusedImport is an imported module that nothing references.
	UnusedImport = diagnostics.Kind{
		Code:     "ARC5001",
		Severity: protocol.DiagnosticSeverityWarning,
		Message:  "imported module %q is unused",
	}
)

// All is each kind of the table.
var All = []diagnostics.Kind{
	Syntax,
	SourceLimit,
	InvalidStringLiteral,
	SubtractionSpacing,
	DivisionByZero,
	TypeMismatch,
	UnknownType,
	UnresolvedType,
	InvalidOperand,
	InvalidCast,
	NotConstant,
	ConstantOverflow,
	DimensionMismatch,
	NaNComparison,
	InvalidLiteral,
	TruthyCondition,
	BoolCast,
	UnitPrecisionLoss,
	FuncArgMissing,
	FuncArgExtra,
	DeprecatedSymbol,
	DeclarationAfterStep,
	UnusedImport,
}
