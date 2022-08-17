// Package errutil contains utilities for error catching, handling, inspection, conversion, etc.
//
// Catch -> Interface and implementations for catching errors returned by a set of similar operations.
// Convert -> Converts errors from one type to another.
// Inspect -> Inspects and prints debugging information about an error.
package errutil

import (
	"context"
	"github.com/arya-analytics/x/binary"
	"io"
)

// Catch provides an interface for catching errors returned by a set of operations.
//
// c := errutil.NewCatchSimple()
// c.exec(myFunc1)  // Returns an error
// c.exec(myFunc2)
// fmt.Println(c.ResponseErrors())
// Output:
//
//	error returned by func1
//
// The Catch will execute all functions until a function returns a non-nil error. In this case, if myFunc1 returns an
// error, myFunc2 will not execute.
//
// Types that implement the Catch interface should also implement a set of functions that
// execute actions (such as CatchSimple.Exec). They are excluded from this interface
// as to support variable interfaces.
type Catch interface {
	// Error returns the most recent error caught.
	Error() error
	// Errors returns all errors caught. Will only have len > 0 if WithAggregation opt is used on instantiation.
	Errors() []error
	// Reset resets the Catch so it becomes error free.
	Reset()
}

// |||| SIMPLE ||||

// CatchSimple is a simple implementation of Catch,
// Direct instantiation should be avoided, and
// should instead be constructed using NewCatchSimple.
type CatchSimple struct {
	errors []error
	opts   *catchOpts
}

// NewCatchSimple instantiates a CatchSimple with the provided options.
func NewCatchSimple(opts ...CatchOpt) *CatchSimple {
	c := &CatchSimple{opts: &catchOpts{}}
	c.bindOpts(opts...)
	return c
}

type CatchAction func() error

// Exec runs a CatchAction and catches any errors that it may return.
// See Catch interface for more info on how this works.
func (c *CatchSimple) Exec(ca CatchAction) {
	if !c.opts.aggregate && len(c.errors) > 0 {
		return
	}
	if err := ca(); err != nil {
		c.runHooks(err)
		c.errors = append(c.errors, err)
	}
}

// Reset implements Catch.
func (c *CatchSimple) Reset() {
	c.errors = []error{}
}

// Error implements Catch.
func (c *CatchSimple) Error() error {
	if len(c.Errors()) == 0 {
		return nil
	}
	return c.Errors()[0]
}

// Errors implements Catch.
func (c *CatchSimple) Errors() []error {
	return c.errors
}

func (c *CatchSimple) runHooks(err error) {
	for _, h := range c.opts.hooks {
		h(err)
	}
}

func (c *CatchSimple) bindOpts(opts ...CatchOpt) {
	for _, o := range opts {
		o(c.opts)
	}
}

// |||| CONTEXT ||||

// CatchContext extends CatchSimple by receiving a context.Context as an argument
// and running an action that requires that ctx (CatchActionCtx).
type CatchContext struct {
	CatchSimple
	ctx context.Context
}

// NewCatchContext creates a new CatchContext with the provided context and options.
func NewCatchContext(ctx context.Context, opts ...CatchOpt) *CatchContext {
	return &CatchContext{CatchSimple: *NewCatchSimple(opts...), ctx: ctx}
}

type CatchActionCtx func(ctx context.Context) error

// Exec runs a CatchActionCtx and catches any errors that it may return.
// See Catch interface for more info on how this works.
func (c *CatchContext) Exec(ca CatchActionCtx) {
	c.CatchSimple.Exec(func() error { return ca(c.ctx) })
}

// |||| OPTS ||||

type catchOpts struct {
	aggregate bool
	hooks     []CatchHook
}

type CatchOpt func(o *catchOpts)

// WithAggregation causes the Catch to execute all functions and aggregate the errors caught.
// For Example:
//
//		c := errutil.NewCatchSimple(errutil.WithAggregation())
//		c.exec(myFunc1)  // Returns an error
//		c.exec(myFunc2)
//		fmt.Println(c.errors())
//
//	Output:
//		errors returned by myFunc1 and myFunc2
//
// In this case, if myFunc1 returns an error, the Catch will execute and catch any errors returned by myFunc2.
func WithAggregation() CatchOpt {
	return func(o *catchOpts) {
		o.aggregate = true
	}
}

// CatchHook is executed any time an error is caught by the Catch. Receives the caught error as first argument.
type CatchHook func(err error)

// WithHooks binds a set of CatchHook to the Catch.
func WithHooks(hooks ...CatchHook) CatchOpt {
	return func(o *catchOpts) {
		o.hooks = hooks
	}
}

// |||| HOOKS ||||

// || PIPE ||

// NewPipeHook pipes errors caught by the catch into the provided pipe.
func NewPipeHook(pipe chan error) func(err error) {
	return func(err error) {
		pipe <- err
	}
}

type CatchWrite struct {
	w io.Writer
	e error
}

func NewCatchWrite(w io.Writer) *CatchWrite {
	return &CatchWrite{w: w}
}

func (c *CatchWrite) Write(data interface{}) {
	if c.e != nil {
		return
	}
	if err := binary.Write(c.w, data); err != nil {
		c.e = err
	}
}

func (c *CatchWrite) Error() error {
	return c.e
}

type CatchRead struct {
	CatchSimple
	w io.Reader
	e error
}

func NewCatchRead(r io.Reader, opts ...CatchOpt) *CatchRead {
	return &CatchRead{w: r, CatchSimple: *NewCatchSimple(opts...)}
}

func (c *CatchRead) Read(data interface{}) {
	if c.e != nil {
		return
	}
	if err := binary.Read(c.w, data); err != nil {
		c.e = err
	}
}

func (c *CatchRead) Error() error {
	return c.e
}

type CatchReadWriteSeek struct {
	CatchSimple
	w io.ReadWriteSeeker
	e error
}

func NewCatchReadWriteSeek(r io.ReadWriteSeeker, opts ...CatchOpt) *CatchReadWriteSeek {
	return &CatchReadWriteSeek{w: r, CatchSimple: *NewCatchSimple(opts...)}
}

func (c *CatchReadWriteSeek) Read(data interface{}) {
	if c.e != nil {
		return
	}
	if err := binary.Read(c.w, data); err != nil {
		c.e = err
	}
}

func (c *CatchReadWriteSeek) Seek(offset int64, whence int) int64 {
	if c.e != nil {
		return 0
	}
	ret, err := c.w.Seek(offset, whence)
	if err != nil {
		c.e = err
	}
	return ret
}

func (c *CatchReadWriteSeek) Write(data interface{}) {
	if c.e != nil {
		return
	}
	if err := binary.Write(c.w, data); err != nil {
		c.e = err
	}
}

func (c *CatchReadWriteSeek) Exec(action func() error) {
	if c.e != nil {
		return
	}
	if err := action(); err != nil {
		c.e = err
	}
}

func (c *CatchReadWriteSeek) Error() error {
	return c.e
}
