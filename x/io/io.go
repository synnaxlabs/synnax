package io

import "io"

type ReaderAtCloser interface {
	io.ReaderAt
	io.Closer
}
