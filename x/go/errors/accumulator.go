// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package errors

// Accumulator can be used to execute a series of functions and aggregate any errors
// that they may return into a single error.
type Accumulator struct{ err error }

// Exec runs a function and catches any error that it may return.
func (a *Accumulator) Exec(fn func() error) { a.err = Join(a.err, fn()) }

// Error returns the error(s) aggregated by the Accumulator.
func (a Accumulator) Error() error { return a.err }
