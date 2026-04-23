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
//   - Category 5xxx: Unused / unreachable code (warnings)
package codes

import "github.com/synnaxlabs/x/diagnostics"

const (
	// TypeMismatch indicates incompatible types in an operation or assignment.
	TypeMismatch diagnostics.ErrorCode = "ARC2001"
	// TypeConstraintViolation indicates a type doesn't satisfy a type variable's constraint.
	TypeConstraintViolation diagnostics.ErrorCode = "ARC2003"

	// FuncArgCount indicates wrong number of arguments in a function call.
	FuncArgCount diagnostics.ErrorCode = "ARC3001"
	// FuncArgType indicates a function argument has the wrong type.
	FuncArgType diagnostics.ErrorCode = "ARC3002"

	// SymbolUndefined indicates a referenced symbol was not found.
	SymbolUndefined diagnostics.ErrorCode = "ARC4001"
	// SymbolRedefined indicates a symbol was declared more than once in the same scope.
	SymbolRedefined diagnostics.ErrorCode = "ARC4002"

	// UnusedVariable indicates a local or stateful variable (including a channel alias)
	// was declared but never referenced. Prefix the name with an underscore to suppress.
	UnusedVariable diagnostics.ErrorCode = "ARC5101"
	// UncalledFunction indicates a function was declared but never invoked from any
	// call site (flow statement, stage body, routing table, or function-body
	// expression). Prefix the name with an underscore to suppress.
	UncalledFunction diagnostics.ErrorCode = "ARC5102"
	// UnusedGlobalConstant indicates a top-level constant was declared but never
	// referenced from any expression. Prefix the name with an underscore to suppress.
	UnusedGlobalConstant diagnostics.ErrorCode = "ARC5103"
)
