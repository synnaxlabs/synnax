// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package errors

// Catcher can be used to catch errors from a particular function call and
// prevent execution of subsequent functions if the error is caught.
type Catcher struct {
	errors []error
	opts   catcherOpts
}

// NewCatcher instantiates a Catcher with the provided options.
func NewCatcher(opts ...CatcherOpt) *Catcher {
	return &Catcher{opts: newCatchOpts(opts)}
}

// Exec runs a CatchAction and catches any error that it may return.
func (c *Catcher) Exec(ca func() error) {
	if !c.opts.aggregate && len(c.errors) > 0 {
		return
	}
	if err := ca(); err != nil {
		c.errors = append(c.errors, err)
	}
}

// Error returns the most recent error caught.
func (c *Catcher) Error() error {
	if len(c.Errors()) == 0 {
		return nil
	}
	return c.Errors()[0]
}

// Errors returns all errors caught. Will only have len > 1 if WithAggregation
// opt is used on instantiation.
func (c *Catcher) Errors() []error { return c.errors }

type catcherOpts struct {
	aggregate bool
}

func newCatchOpts(opts []CatcherOpt) (c catcherOpts) {
	for _, o := range opts {
		o(&c)
	}
	return c
}

// CatcherOpt is a functional option for configuring a Catcher.
type CatcherOpt func(o *catcherOpts)

// WithAggregation causes the Catcher to execute all functions and aggregate the errors caught.
// For Example:
//
//		c := errutil.NewCatcher(errutil.WithAggregation())
//		c.exec(myFunc1)  // Returns an error
//		c.exec(myFunc2)
//		fmt.Println(c.errors())
//
//	Output:
//		errors returned by myFunc1 and myFunc2
//
// In this case, if myFunc1 returns an error, the Catcher will execute and catch any errors returned by myFunc2.
func WithAggregation() CatcherOpt { return func(o *catcherOpts) { o.aggregate = true } }
