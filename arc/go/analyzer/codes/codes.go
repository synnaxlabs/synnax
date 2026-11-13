// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package codes defines diagnostic error codes for the Arc compiler.
// Error codes follow the format ARC<category><number> where:
//   - Category 2xxx: Type system errors
//   - Category 3xxx: Function-related errors
//   - Category 4xxx: Symbol/scope errors
//   - Category 5xxx: Import errors
//   - Category 6xxx: Constant expression errors
package codes

//go:generate go run gen.go

import (
	"github.com/synnaxlabs/x/diagnostics"
	"go.lsp.dev/protocol"
)

// Reserved codes. The compiler does not report them.
const (
	// TypeMismatch indicates incompatible types in an operation or assignment.
	TypeMismatch diagnostics.ErrorCode = "ARC2001"
	// TypeConstraintViolation indicates a type doesn't satisfy a type variable's
	// constraint.
	TypeConstraintViolation diagnostics.ErrorCode = "ARC2003"
	// SymbolUndefined indicates a referenced symbol was not found.
	SymbolUndefined diagnostics.ErrorCode = "ARC4001"
	// SymbolRedefined indicates a symbol was declared more than once in the same scope.
	SymbolRedefined diagnostics.ErrorCode = "ARC4002"
)

var (
	// NaNComparison indicates a comparison with a NaN constant, which always has the
	// same result.
	NaNComparison = diagnostics.Kind{
		Code:     "ARC2004",
		Severity: protocol.DiagnosticSeverityError,
		Message:  "%s with %s is always %t, use %s instead",
	}
	// TruthyCondition is a number or a string used as a condition.
	TruthyCondition = diagnostics.Kind{
		Code:     "ARC2005",
		Severity: protocol.DiagnosticSeverityWarning,
		Message:  "non-boolean conditions are deprecated, use an explicit comparison like x != 0",
	}
	// BoolCast is a cast of a bool to a number type other than u8.
	BoolCast = diagnostics.Kind{
		Code:     "ARC2006",
		Severity: protocol.DiagnosticSeverityWarning,
		Message:  "bool to %s is experimental; a future release may only allow bool to u8",
	}

	// FuncArgMissing is a call with no argument for a required parameter.
	FuncArgMissing = diagnostics.Kind{
		Code:     "ARC3001",
		Severity: protocol.DiagnosticSeverityError,
		Message:  "missing required argument for parameter '%s' of func '%s'",
	}
	// FuncArgType indicates a function argument has the wrong type.
	FuncArgType = diagnostics.Kind{
		Code:     "ARC3002",
		Severity: protocol.DiagnosticSeverityError,
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
		Severity: protocol.DiagnosticSeverityError,
		Message:  "imported module %q is unused",
	}

	// ConstantOverflow is a constant integer that does not fit its type.
	ConstantOverflow = diagnostics.Kind{
		Code:     "ARC6001",
		Severity: protocol.DiagnosticSeverityError,
		Message:  "constant %s overflows %s",
	}
	// DivisionByZero is a constant integer division or modulo by zero.
	DivisionByZero = diagnostics.Kind{
		Code:     "ARC6002",
		Severity: protocol.DiagnosticSeverityError,
		Message:  "integer %s by zero",
	}
)

// All is each kind of the table.
var All = []diagnostics.Kind{
	NaNComparison,
	TruthyCondition,
	BoolCast,
	FuncArgMissing,
	FuncArgType,
	FuncArgExtra,
	DeprecatedSymbol,
	DeclarationAfterStep,
	UnusedImport,
	ConstantOverflow,
	DivisionByZero,
}
