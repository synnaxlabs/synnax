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
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	"go.uber.org/zap"
)

type Writer struct {
	requests          confluence.Inlet[WriterRequest]
	responses         confluence.Outlet[WriterResponse]
	wg                signal.WaitGroup
	logger            *zap.Logger
	hasAccumulatedErr bool
	closed            bool
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

func (w *Writer) Write(frame Frame) bool {
	if w.closed || w.hasAccumulatedErr {
		return false
	}
	select {
	case <-w.responses.Outlet():
		w.hasAccumulatedErr = true
		return false
	case w.requests.Inlet() <- WriterRequest{Frame: frame, Command: WriterWrite}:
		return true
	}
}

func (w *Writer) Commit() (telem.TimeStamp, bool) {
	if w.closed || w.hasAccumulatedErr {
		return 0, false
	}
	select {
	case <-w.responses.Outlet():
		w.hasAccumulatedErr = true
		return 0, false
	case w.requests.Inlet() <- WriterRequest{Command: WriterCommit}:
	}
	for res := range w.responses.Outlet() {
		if res.Command == WriterCommit {
			return res.End, res.Ack
		}
	}
	w.logger.DPanic(unexpectedSteamClosure)
	return 0, false
}

// SetAuthority is synchronous
func (w *Writer) SetAuthority(cfg WriterConfig) bool {
	if w.closed || w.hasAccumulatedErr {
		return false
	}
	select {
	case <-w.responses.Outlet():
		w.hasAccumulatedErr = true
		return false
	case w.requests.Inlet() <- WriterRequest{Config: cfg, Command: WriterSetAuthority}:
	}
	for res := range w.responses.Outlet() {
		if res.Command == WriterSetAuthority {
			return res.Ack
		}
	}
	w.logger.DPanic(unexpectedSteamClosure)
	return false
}

func (w *Writer) Error() error {
	if w.closed {
		return errWriterClosed
	}
	w.requests.Inlet() <- WriterRequest{Command: WriterError}
	for res := range w.responses.Outlet() {
		if res.Command == WriterError {
			w.hasAccumulatedErr = false
			return res.Err
		}
	}
	w.logger.DPanic(unexpectedSteamClosure)
	return errors.New(unexpectedSteamClosure)
}

func (w *Writer) Close() (err error) {
	if w.closed {
		return nil
	}
	w.closed = true
	w.requests.Close()
	for range w.responses.Outlet() {
	}
	return w.wg.Wait()
}
