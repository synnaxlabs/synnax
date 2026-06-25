// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package types

// Zero returns the zero value for T. It is a convenience for returning the zero value
// inline, such as in an error-return path of a generic function, without declaring a
// named variable.
func Zero[T any]() T {
	var zero T
	return zero
}
