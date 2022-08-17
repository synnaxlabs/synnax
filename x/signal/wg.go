package signal

// WaitGroup provides methods for detecting and waiting for the exit of goroutines
// managed by a signal.Conductor.
type WaitGroup interface {
	// Wait waits for all running goroutines to exit, then proceeds to return
	// the first non-nil error (returns nil if all errors are nil). Returns nil
	// if no goroutines are running. WaitOnAll. is NOT safe to call concurrently
	// with any other wait methods.
	Wait() error
	// Stopped returns a channel that is closed when the context is canceled AND all
	// running goroutines have exited.
	Stopped() <-chan struct{}
}

// Wait implements the WaitGroup interface.
func (c *core) Wait() error { return c.wrapped.Wait() }
