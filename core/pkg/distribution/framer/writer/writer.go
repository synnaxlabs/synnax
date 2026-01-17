// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package writer

import (
	"io"

	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
)

type StreamWriter = confluence.Segment[Request, Response]

var ErrClosed = errors.New("writer closed")

type Writer struct {
	cfg       Config
	requests  confluence.Inlet[Request]
	responses confluence.Outlet[Response]
	shutdown  io.Closer
	closeErr  error
}

// Write implements Writer.
func (w *Writer) Write(frame frame.Frame) (authorized bool, err error) {
	res, err := w.exec(Request{Frame: frame, Command: CommandWrite}, *w.cfg.Sync)
	if err != nil {
		return false, err
	}
	authorized = !*w.cfg.Sync || res.Authorized
	return
}

func (w *Writer) Commit() (telem.TimeStamp, error) {
	res, err := w.exec(Request{Command: CommandCommit}, true)
	return res.End, err
}

func (w *Writer) SetAuthority(cfg Config) error {
	_, err := w.exec(Request{Command: CommandSetAuthority, Config: cfg}, true)
	return err
}

func (w *Writer) exec(req Request, sync bool) (Response, error) {
	var res Response
	if w.closeErr != nil {
		return res, w.closeErr
	}
	select {
	case res = <-w.responses.Outlet():
		return res, w.close(res.Err)
	case w.requests.Inlet() <- req:
	}
	if !sync {
		return res, nil
	}
	for res = range w.responses.Outlet() {
		if res.Err != nil {
			return res, w.close(res.Err)
		}
		if res.Command == req.Command {
			return res, nil
		}
	}
	return res, w.close(nil)
}

func (w *Writer) Close() error { return w.close(nil) }

func (w *Writer) close(err error) error {
	if w.closeErr != nil {
		return errors.Skip(w.closeErr, ErrClosed)
	}
	w.closeErr = err
	w.requests.Close()
	confluence.Drain(w.responses)
	w.closeErr = errors.Combine(w.closeErr, w.shutdown.Close())
	if w.closeErr != nil {
		return w.closeErr
	}
	w.closeErr = ErrClosed
	return nil

}
