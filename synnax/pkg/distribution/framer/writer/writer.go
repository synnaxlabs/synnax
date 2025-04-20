// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package writer

import (
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/core"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	"io"
)

type StreamWriter = confluence.Segment[Request, Response]

var ErrClosed = errors.New("writer closed")

type Writer struct {
	cfg       Config
	requests  confluence.Inlet[Request]
	responses confluence.Outlet[Response]
	shutdown  io.Closer
	closed    bool
}

// Write implements Writer.
func (w *Writer) Write(frame core.Frame) (authorized bool, err error) {
	res, err := w.exec(Request{Frame: frame, Command: Data}, *w.cfg.Sync)
	if err != nil {
		return false, err
	}
	authorized = !*w.cfg.Sync || res.Authorized
	return
}

func (w *Writer) Commit() (telem.TimeStamp, error) {
	res, err := w.exec(Request{Command: Commit}, true)
	return res.End, err
}

func (w *Writer) SetAuthority(cfg Config) error {
	_, err := w.exec(Request{Command: SetAuthority, Config: cfg}, true)
	return err
}

func (w *Writer) exec(req Request, sync bool) (res Response, err error) {
	if w.closed {
		return res, ErrClosed
	}
	select {
	case <-w.responses.Outlet():
		return res, w.Close()
	case w.requests.Inlet() <- req:
	}
	if !sync {
		return
	}
	for res = range w.responses.Outlet() {
		if res.Command == req.Command {
			return res, nil
		}
	}
	return res, w.Close()
}

func (w *Writer) Close() error {
	w.closed = true
	w.requests.Close()
	confluence.Drain(w.responses)
	return w.shutdown.Close()
}
