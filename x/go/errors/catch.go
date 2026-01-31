// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package errors

// Catcher can be used to catch errors from a series of function calls and aggregate
// them into a single error list.
type Catcher struct{ err error }

// Exec runs a CatchAction and catches any error that it may return.
func (c *Catcher) Exec(ca func() error) { c.err = Join(c.err, ca()) }

// Error returns the error aggregated by the Catcher.
func (c *Catcher) Error() error { return c.err }
