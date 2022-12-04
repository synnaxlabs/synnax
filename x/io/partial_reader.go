package io

import (
	"io"
)

// PartialReader is a internal around io.ReaderAt the returns a Reader over a subset
// of the underlying data.
func PartialReader(r io.ReaderAt, offset, length int64) ReaderReaderAt {
	return &partialReader{
		internal: r,
		offset:   offset,
		length:   length,
	}
}

type partialReader struct {
	offset   int64
	length   int64
	nRead    int64
	internal io.ReaderAt
}

func (r *partialReader) Read(p []byte) (n int, err error) {
	if r.nRead >= r.length {
		return 0, io.EOF
	}
	n, err = r.internal.ReadAt(p, r.offset+r.nRead)
	if err != nil {
		return
	}
	nextRead := r.nRead + int64(n)
	if nextRead >= r.length {
		err = io.EOF
		n = int(r.length - r.nRead)
	}
	r.nRead = nextRead
	return n, err
}

func (r *partialReader) ReadAt(p []byte, off int64) (n int, err error) {
	if off >= r.length {
		return 0, io.EOF
	}
	n, err = r.internal.ReadAt(p, r.offset+off)
	if err != nil {
		return
	}
	if r.offset+off+int64(len(p)) > r.length {
		p = p[:r.length-r.offset-off]
		n = len(p)
		err = io.EOF
	}
	return n, err
}
