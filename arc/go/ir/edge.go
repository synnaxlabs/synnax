// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ir

import v2 "github.com/synnaxlabs/arc/ir/types/v2"

// Parameter naming conventions for IR nodes and functions.
const (
	// DefaultOutputParam is the parameter name for single-output functions and stages.
	// Use this for unary operations like neg, sqrt, etc.
	DefaultOutputParam = v2.DefaultOutputParam
	// DefaultInputParam is the parameter name for single-input functions and stages.
	// Use this for unary operations that take one input.
	DefaultInputParam = v2.DefaultInputParam
	// LHSInputParam is the left-hand side parameter name for binary operators.
	// Use this as the first operand name in operations like add, multiply, etc.
	LHSInputParam = v2.LHSInputParam
	// RHSInputParam is the right-hand side parameter name for binary operators.
	// Use this as the second operand name in operations like add, multiply, etc.
	RHSInputParam = v2.RHSInputParam
)
