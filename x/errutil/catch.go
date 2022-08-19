// Package errutil contains utilities for working with errors.
package errutil

// Catch can be used to catch errors from a particular function call and
// prevent execution of subsequent functions if the error is caught.
type Catch struct {
	errors []error
	opts   catchOpts
}

// NewCatch instantiates a Catch with the provided options.
func NewCatch(opts ...CatchOpt) *Catch {
	return &Catch{opts: newCatchOpts(opts)}
}

// Exec runs a CatchAction and catches any errors that it may return.
// See Catch interface for more info on how this works.
func (c *Catch) Exec(ca func() error) {
	if !c.opts.aggregate && len(c.errors) > 0 {
		return
	}
	if err := ca(); err != nil {
		c.errors = append(c.errors, err)
	}
}

// Reset resets the Catch so it becomes error free.
func (c *Catch) Reset() { c.errors = []error{} }

// Error returns the most recent error caught.
func (c *Catch) Error() error {
	if len(c.Errors()) == 0 {
		return nil
	}
	return c.Errors()[0]
}

// Errors returns all errors caught. Will only have len > 1 if WithAggregation
// opt is used on instantiation.
func (c *Catch) Errors() []error { return c.errors }

// |||| OPTS ||||

type catchOpts struct {
	aggregate bool
}

func newCatchOpts(opts []CatchOpt) (c catchOpts) {
	for _, o := range opts {
		o(&c)
	}
	return c
}

type CatchOpt func(o *catchOpts)

// WithAggregation causes the Catch to execute all functions and aggregate the errors caught.
// For Example:
//
//		c := errutil.NewCatch(errutil.WithAggregation())
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
