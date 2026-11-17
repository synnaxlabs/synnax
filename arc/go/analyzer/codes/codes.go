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

	// NotCallable is a call of a name that the code cannot call there.
	NotCallable = diagnostics.Kind{
		Code:     "E3000",
		Severity: protocol.DiagnosticSeverityError,
	}
	// FuncArgMissing is a call with no argument for a required parameter.
	FuncArgMissing = diagnostics.Kind{
		Code:     "E3001",
		Severity: protocol.DiagnosticSeverityError,
		Message:  "missing required argument for parameter '%s' of func '%s'",
	}
	// FuncArgExtra is a call with more arguments than parameters.
	FuncArgExtra = diagnostics.Kind{
		Code:     "E3002",
		Severity: protocol.DiagnosticSeverityError,
		Message:  "too many arguments for func '%s': expected at most %d",
	}
	// FuncArgName is a named argument that is a duplicate or has no parameter.
	FuncArgName = diagnostics.Kind{
		Code:     "E3003",
		Severity: protocol.DiagnosticSeverityError,
	}
	// FuncArgValue is a constant argument with a value the function rejects.
	FuncArgValue = diagnostics.Kind{
		Code:     "E3004",
		Severity: protocol.DiagnosticSeverityError,
	}
	// ParamOrder is a required parameter after an optional parameter.
	ParamOrder = diagnostics.Kind{
		Code:     "E3005",
		Severity: protocol.DiagnosticSeverityError,
	}
	// ReturnValue is a return with a value missing or a value not allowed.
	ReturnValue = diagnostics.Kind{
		Code:     "E3006",
		Severity: protocol.DiagnosticSeverityError,
	}
	// MisplacedStatement is a break, continue, or return outside its construct.
	MisplacedStatement = diagnostics.Kind{
		Code:     "E3007",
		Severity: protocol.DiagnosticSeverityError,
	}
	// StatefulVariable is a stateful variable in a place that does not allow one.
	StatefulVariable = diagnostics.Kind{
		Code:     "E3008",
		Severity: protocol.DiagnosticSeverityError,
	}
	// AssignmentTarget is an assignment to a target that does not accept it.
	AssignmentTarget = diagnostics.Kind{
		Code:     "E3009",
		Severity: protocol.DiagnosticSeverityError,
	}
	// CircularCall is a function that calls itself through a chain of calls.
	CircularCall = diagnostics.Kind{
		Code:     "E3010",
		Severity: protocol.DiagnosticSeverityError,
	}
	// OutputNeverAssigned is a named output that no path of its function sets.
	OutputNeverAssigned = diagnostics.Kind{
		Code:     "W3000",
		Severity: protocol.DiagnosticSeverityWarning,
	}

	// ImportFailed is an import of a module that does not exist or does not load.
	ImportFailed = diagnostics.Kind{
		Code:     "E4000",
		Severity: protocol.DiagnosticSeverityError,
	}
	// ImportDeclaration is an import that is a duplicate or is out of place.
	ImportDeclaration = diagnostics.Kind{
		Code:     "E4001",
		Severity: protocol.DiagnosticSeverityError,
	}
	// NotExported is a reference to a member that its module does not export.
	NotExported = diagnostics.Kind{
		Code:     "E4002",
		Severity: protocol.DiagnosticSeverityError,
	}
	// ModuleDeclaration is a declaration that a module, or a program, cannot hold.
	ModuleDeclaration = diagnostics.Kind{
		Code:     "E4003",
		Severity: protocol.DiagnosticSeverityError,
	}
	// SymbolUndefined is a reference to a name that no scope declares.
	SymbolUndefined = diagnostics.Kind{
		Code:     "E4004",
		Severity: protocol.DiagnosticSeverityError,
	}
	// SymbolRedefined is a declaration of a name that its scope already has.
	SymbolRedefined = diagnostics.Kind{
		Code:     "E4005",
		Severity: protocol.DiagnosticSeverityError,
	}
	// UnusedImport is an imported module that nothing references.
	UnusedImport = diagnostics.Kind{
		Code:     "W4000",
		Severity: protocol.DiagnosticSeverityWarning,
		Message:  "imported module %q is unused",
	}
	// DeprecatedSymbol is a reference to a name that has a replacement.
	DeprecatedSymbol = diagnostics.Kind{
		Code:     "W4001",
		Severity: protocol.DiagnosticSeverityWarning,
		Message:  "'%s' is deprecated, use '%s' instead",
	}

	// DeclarationAfterStep is a variable declared after a step of its scope.
	DeclarationAfterStep = diagnostics.Kind{
		Code:     "ARC4004",
		Severity: protocol.DiagnosticSeverityWarning,
		Message:  "declare variables at the top of a stage or sequence",
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
	NotCallable,
	FuncArgMissing,
	FuncArgExtra,
	FuncArgName,
	FuncArgValue,
	ParamOrder,
	ReturnValue,
	MisplacedStatement,
	StatefulVariable,
	AssignmentTarget,
	CircularCall,
	OutputNeverAssigned,
	ImportFailed,
	ImportDeclaration,
	NotExported,
	ModuleDeclaration,
	SymbolUndefined,
	SymbolRedefined,
	UnusedImport,
	DeprecatedSymbol,
	DeclarationAfterStep,
}
