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
	requests  confluence.Inlet[WriterRequest]
	responses confluence.Outlet[WriterResponse]
	wg        signal.WaitGroup
	logger    *zap.Logger
	closed    bool
}

const unexpectedSteamClosure = "unexpected early closure of response stream"

var errWriterClosed = core.EntityClosed("cesium.writer")

func wrapStreamWriter(internal StreamWriter) *Writer {
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
	return &Writer{requests: req, responses: res, wg: sCtx}
}

func (w *Writer) Write(frame Frame) error {
	if w.closed {
		return errWriterClosed
	}
	select {
	case <-w.responses.Outlet():
		return w.Close()
	case w.requests.Inlet() <- WriterRequest{Frame: frame, Command: WriterWrite}:
	}
	return nil
}

func (w *Writer) Commit() (telem.TimeStamp, error) {
	if w.closed {
		return 0, errWriterClosed
	}
	select {
	case <-w.responses.Outlet():
		return 0, w.Close()
	case w.requests.Inlet() <- WriterRequest{Command: WriterCommit}:
	}
	for res := range w.responses.Outlet() {
		if res.Command == WriterCommit {
			return res.End, nil
		}
	}
	return 0, w.Close()
}

// SetAuthority is synchronous
func (w *Writer) SetAuthority(cfg WriterConfig) error {
	if w.closed {
		return errWriterClosed
	}
	select {
	case <-w.responses.Outlet():
		return w.Close()
	case w.requests.Inlet() <- WriterRequest{Config: cfg, Command: WriterSetAuthority}:
	}
	for res := range w.responses.Outlet() {
		if res.Command == WriterSetAuthority {
			return nil
		}
	}
	return w.Close()
}

func (w *Writer) Close() error {
	w.closed = true
	w.requests.Close()
	confluence.Drain(w.responses)
	return w.wg.Wait()
}
