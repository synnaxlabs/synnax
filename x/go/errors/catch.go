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
type Catcher struct{ errors []error }

// NewCatcher instantiates a Catcher.
func NewCatcher() *Catcher { return &Catcher{} }

// Exec runs a CatchAction and catches any error that it may return.
func (c *Catcher) Exec(ca func() error) {
	if err := ca(); err != nil {
		c.errors = append(c.errors, err)
	}
}

// Error returns the first error caught.
func (c *Catcher) Error() error {
	if len(c.Errors()) == 0 {
		return nil
	}
	return c.Errors()[0]
}

// Errors returns all errors caught.
func (c *Catcher) Errors() []error { return c.errors }
