// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package diagnostics

// ErrorCode identifies a diagnostic category for Arc compiler errors.
// Error codes follow the format ARC<category><number> where:
//   - Category 2xxx: Type system errors
//   - Category 3xxx: Function-related errors
//   - Category 4xxx: Symbol/scope errors
type ErrorCode string

const (
	// Type system errors (2xxx)

	// ErrorCodeTypeMismatch indicates incompatible types in an operation or assignment.
	ErrorCodeTypeMismatch ErrorCode = "ARC2001"
	// ErrorCodeTypeConstraintViolation indicates a type doesn't satisfy a type variable's constraint.
	ErrorCodeTypeConstraintViolation ErrorCode = "ARC2003"

	// Function errors (3xxx)

	// ErrorCodeFuncArgCount indicates wrong number of arguments in a function call.
	ErrorCodeFuncArgCount ErrorCode = "ARC3001"
	// ErrorCodeFuncArgType indicates a function argument has the wrong type.
	ErrorCodeFuncArgType ErrorCode = "ARC3002"

	// Symbol errors (4xxx)

	// ErrorCodeSymbolUndefined indicates a referenced symbol was not found.
	ErrorCodeSymbolUndefined ErrorCode = "ARC4001"
	// ErrorCodeSymbolRedefined indicates a symbol was declared more than once in the same scope.
	ErrorCodeSymbolRedefined ErrorCode = "ARC4002"
)
