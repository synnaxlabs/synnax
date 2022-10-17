package storage

import (
	"context"
	"github.com/synnaxlabs/cesium/internal/file"
	"github.com/synnaxlabs/cesium/internal/persist"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/telem"
)

type ReadRequest interface {
	STarget() file.Key
	SOffset() telem.Offset
	SSize() telem.Size
}

type ReadResponse[R ReadRequest] struct {
	Request R
	Err     error
	Data    []byte
}

type Reader[R ReadRequest] interface {
	confluence.Segment[R, ReadResponse[R]]
}

type readOperation[R ReadRequest] struct {
	req       R
	responses chan<- ReadResponse[R]
}

func (r readOperation[R]) Target() file.Key {
	return r.Target()
}

func (r readOperation[R]) Exec(f file.File, err error) {
	if err != nil {
		r.sendRes(err, nil)
		return
	}
	data := make([]byte, r.req.SSize())
	_, err = f.ReadAt(data, int64(r.req.SOffset()))
	r.sendRes(err, data)
}

func (r readOperation[R]) sendRes(err error, data []byte) {
	r.responses <- ReadResponse[R]{
		Request: r.req,
		Err:     err,
		Data:    data,
	}
}

type reader[R ReadRequest] struct {
	confluence.UnarySink[R]
	confluence.AbstractUnarySource[ReadResponse[R]]
	ops confluence.Inlet[[]persist.Operation[file.Key]]
}

func (r *reader[R]) sink(ctx context.Context, req R) error {
	r.ops.Inlet() <- []persist.Operation[file.Key]{
		readOperation[R]{req: req, responses: r.Out.Inlet()},
	}
	return nil
}
