package io

import "io"

// CloserFunc allows a function to implement the io.Closer interface.
type CloserFunc func() error

var _ io.Closer = CloserFunc(nil)

// Close implements io.Closer.
func (c CloserFunc) Close() error { return c() }
