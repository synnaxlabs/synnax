// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cesium

import (
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	"go.uber.org/zap"
)

type Writer struct {
	cfg       WriterConfig
	requests  confluence.Inlet[WriterRequest]
	responses confluence.Outlet[WriterResponse]
	wg        signal.WaitGroup
	logger    *zap.Logger
	closed    bool
}

const unexpectedSteamClosure = "unexpected early closure of response stream"

var errWriterClosed = core.EntityClosed("cesium.writer")

func wrapStreamWriter(cfg WriterConfig, internal StreamWriter) *Writer {
	sCtx, _ := signal.Isolated()
	req := confluence.NewStream[WriterRequest](1)
	res := confluence.NewStream[WriterResponse](1)
	internal.InFrom(req)
	internal.OutTo(res)
	internal.Flow(
		sCtx,
		confluence.CloseOutputInletsOnExit(),
		confluence.RecoverWithErrOnPanic(),
	)
	return &Writer{cfg: cfg, requests: req, responses: res, wg: sCtx}
}

func (w *Writer) Write(frame Frame) (bool, error) {
	res, err := w.exec(WriterRequest{Frame: frame, Command: WriterWrite}, *w.cfg.Sync)
	if err != nil {
		return false, err
	}
	if *w.cfg.Sync {
		return res.Authorized, nil
	}
	return true, nil
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
	if w.closed {
		return res, errWriterClosed
	}
	select {
	case <-w.responses.Outlet():
		return res, w.Close()
	case w.requests.Inlet() <- req:
	}
	if !sync {
		return
	}
	for res := range w.responses.Outlet() {
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
	return w.wg.Wait()
}
