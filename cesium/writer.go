// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cesium

import (
	"io"

	"github.com/synnaxlabs/cesium/internal/resource"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
)

type Writer struct {
	cfg       WriterConfig
	requests  confluence.Inlet[WriterRequest]
	responses confluence.Outlet[WriterResponse]
	shutdown  io.Closer
	closeErr  error
}

const unexpectedSteamClosure = "unexpected early closure of response stream"

var errWriterClosed = resource.NewErrClosed("cesium.writer")

func wrapStreamWriter(cfg WriterConfig, internal StreamWriter) *Writer {
	sCtx, cancel := signal.Isolated()
	req := confluence.NewStream[WriterRequest](1)
	res := confluence.NewStream[WriterResponse](1)
	internal.InFrom(req)
	internal.OutTo(res)
	internal.Flow(
		sCtx,
		confluence.CloseOutputInletsOnExit(),
		confluence.RecoverWithErrOnPanic(),
	)
	return &Writer{
		cfg:       cfg,
		requests:  req,
		responses: res,
		shutdown:  signal.NewHardShutdown(sCtx, cancel),
	}
}

func (w *Writer) Write(frame Frame) (authorized bool, err error) {
	res, err := w.exec(WriterRequest{Frame: frame, Command: WriterWrite}, *w.cfg.Sync)
	if err != nil {
		return false, err
	}
	authorized = !*w.cfg.Sync || res.Authorized
	return
}

func (w *Writer) Commit() (telem.TimeStamp, error) {
	res, err := w.exec(WriterRequest{Command: WriterCommit}, true)
	return res.End, err
}

// SetAuthority is synchronous
func (w *Writer) SetAuthority(cfg WriterConfig) error {
	_, err := w.exec(WriterRequest{Config: cfg, Command: WriterSetAuthority}, true)
	return err
}

func (w *Writer) exec(req WriterRequest, sync bool) (res WriterResponse, err error) {
	if w.closeErr != nil {
		return res, w.closeErr
	}
	select {
	case res := <-w.responses.Outlet():
		return res, w.close(res.Err)
	case w.requests.Inlet() <- req:
	}
	if !sync {
		return
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
		return errors.Skip(w.closeErr, errWriterClosed)
	}
	w.closeErr = err
	w.requests.Close()
	confluence.Drain(w.responses)
	w.closeErr = errors.Combine(w.closeErr, w.shutdown.Close())
	if w.closeErr != nil {
		return w.closeErr
	}
	w.closeErr = errWriterClosed
	return nil
}
