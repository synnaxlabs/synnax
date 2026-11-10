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

	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
)

type StreamWriter = confluence.Segment[Request, Response]

var ErrClosed = errors.New("writer closed")

type Writer struct {
	requests  confluence.Inlet[Request]
	responses confluence.Outlet[Response]
	shutdown  io.Closer
	closeErr  error
	cfg       Config
}

// Write implements Writer.
func (w *Writer) Write(frame frame.Frame) (authorized bool, err error) {
	res, err := w.exec(Request{Frame: frame, Command: CommandWrite}, *w.cfg.Sync)
	if err != nil {
		return false, err
	}
	authorized = !*w.cfg.Sync || res.Authorized
	return authorized, err
}

// Commit commits all writes since the last commit and returns the end of the commit.
// authorized is false if the writer lacks control of a channel with samples to commit,
// and the end is then zero. authorized is also false if a write since the last commit
// was refused for no control, but the commit applies and returns its end. The writer
// stays open so that it can regain control.
func (w *Writer) Commit() (end telem.TimeStamp, authorized bool, err error) {
	res, err := w.exec(Request{Command: CommandCommit}, true)
	if err != nil {
		return 0, false, err
	}
	return res.End, res.Authorized, nil
}

func (w *Writer) SetAuthority(cfg Config) error {
	_, err := w.exec(Request{Command: CommandSetAuthority, Config: cfg}, true)
	return err
}

// EndDomain commits the writer, then ends the domains of the given indexes and of the
// data channels they index. It returns false if another writer controls an index or one
// of its data channels. The index then keeps its domain, and some of its data channels
// may end theirs.
func (w *Writer) EndDomain(indexes channel.Keys) (authorized bool, err error) {
	res, err := w.exec(
		Request{Command: CommandEndDomain, Config: Config{Keys: indexes}},
		true,
	)
	if err != nil {
		return false, err
	}
	return res.Authorized, nil
}

func (w *Writer) exec(req Request, sync bool) (Response, error) {
	var res Response
	if w.closeErr != nil {
		return res, w.closeErr
	}
	select {
	case res = <-w.responses.Outlet():
		if res.Err != nil {
			return Response{}, w.close(res.Err)
		}
	case w.requests.Inlet() <- req:
		if !sync {
			return res, nil
		}
		for res = range w.responses.Outlet() {
			if res.Err != nil {
				return Response{}, w.close(res.Err)
			}
			if res.Command == req.Command {
				return res, nil
			}
		}
	}
	if err := w.close(nil); err != nil {
		return Response{}, err
	}
	return Response{}, errors.Wrap(ErrClosed, "writer exited without a response")
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
