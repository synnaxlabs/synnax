package ranger

import (
	"github.com/cockroachdb/errors"
	"github.com/synnaxlabs/x/buffalo"
	xio "github.com/synnaxlabs/x/io"
	xfs "github.com/synnaxlabs/x/io/fs"
	"io"
	"os"
	"strconv"
)

type OffsetWriteCloser interface {
	io.WriteCloser
	Offset() int64
	Length() int64
	FileKey() uint16
}

type fs struct {
	wrapped xfs.FS
	counter uint16
	pool    *buffalo.Pool
}

func newFS(wrapped xfs.FS) *fs {
	return &fs{
		wrapped: wrapped,
		pool:    buffalo.NewPool(4096),
	}
}

func (f *fs) newReader(key uint16) (xio.ReaderAtCloser, error) {
	return f.wrapped.OpenFile(
		strconv.Itoa(int(key))+".ranger",
		os.O_RDONLY,
		0644,
	)
}

func (f *fs) newOffsetWriteCloser() (OffsetWriteCloser, error) {
	// open file in append mode
	file, err := f.wrapped.OpenFile(
		strconv.Itoa(int(f.counter))+".ranger",
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		0644,
	)
	if err != nil {
		return nil, err
	}
	offset, err := file.Seek(0, io.SeekEnd)
	if err != nil {
		return nil, err
	}
	buf := f.pool.Acquire()
	return &offsetWriteCloser{
		fileKey: f.counter,
		offset:  offset,
		writer: &buffalo.BufferedWriter{
			Buffer: buf,
			Writer: file,
		},
		closer: file,
		buffer: buf,
	}, nil
}

type offsetWriteCloser struct {
	offset  int64
	length  int64
	closer  io.Closer
	writer  *buffalo.BufferedWriter
	buffer  *buffalo.Buffer
	fileKey uint16
}

func (o *offsetWriteCloser) Offset() int64 {
	return o.offset
}

func (o *offsetWriteCloser) Length() int64 {
	return o.length
}

func (o *offsetWriteCloser) FileKey() uint16 {
	return o.fileKey
}

func (o *offsetWriteCloser) Write(p []byte) (n int, err error) {
	n, err = o.writer.Write(p)
	o.length += int64(n)
	return
}

func (o *offsetWriteCloser) Close() error {
	err := o.writer.Flush()
	o.buffer.Release()
	return errors.CombineErrors(o.closer.Close(), err)
}
