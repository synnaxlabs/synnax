package storage

import (
	"context"
	"github.com/synnaxlabs/cesium/internal/file"
	"github.com/synnaxlabs/cesium/internal/persist"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/telem"
	"io"
)

type WriteRequest interface {
	STarget() file.Key
	SData() []byte
}

type WriteResponse[R WriteRequest] struct {
	Request R
	Offset  telem.Offset
	Err     error
}

type Writer[R WriteRequest] interface {
	confluence.Segment[R, WriteResponse[R]]
}

type writeOperation[R WriteRequest] struct {
	req       R
	responses chan<- WriteResponse[R]
}

func (w writeOperation[R]) Target() file.Key {
	return w.req.STarget()
}

func (w writeOperation[R]) Exec(f file.File, err error) {
	if err != nil {
		w.sendRes(err, 0)
		return
	}
	off, err := f.Seek(0, io.SeekEnd)
	if err != nil {
		w.sendRes(err, 0)
		return
	}
	_, err = f.Write(w.req.SData())
	w.sendRes(err, telem.Offset(off))
}

func (w writeOperation[R]) sendRes(err error, off telem.Offset) {
	w.responses <- WriteResponse[R]{
		Request: w.req,
		Err:     err,
		Offset:  off,
	}
}

type writer[R WriteRequest] struct {
	confluence.UnarySink[R]
	confluence.AbstractUnarySource[WriteResponse[R]]
	ops confluence.Inlet[[]persist.Operation[file.Key]]
}

func (w *writer[R]) sink(ctx context.Context, req R) error {
	w.ops.Inlet() <- []persist.Operation[file.Key]{writeOperation[R]{req: req, responses: w.Out.Inlet()}}
	return nil
}
