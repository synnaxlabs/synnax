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
	"github.com/synnaxlabs/x/signal"
	"io"
)

type StreamWriter = confluence.Segment[Request, Response]

type Writer struct {
	requests          confluence.Inlet[Request]
	responses         confluence.Outlet[Response]
	wg                signal.WaitGroup
	shutdown          io.Closer
	hasAccumulatedErr bool
}

// Write implements Writer.
func (w *Writer) Write(frame core.Frame) bool {
	if w.hasAccumulatedErr {
		return false
	}
	select {
	case <-w.wg.Stopped():
		return false
	case <-w.responses.Outlet():
		w.hasAccumulatedErr = true
		return false
	case w.requests.Inlet() <- Request{Command: Data, Frame: frame}:
		return true
	}
}

func (w *Writer) Commit() bool {
	if w.hasAccumulatedErr {
		return false
	}
	select {
	case <-w.wg.Stopped():
		return false
	case <-w.responses.Outlet():
		w.hasAccumulatedErr = true
		return false
	case w.requests.Inlet() <- Request{Command: Commit}:
	}
	for res := range w.responses.Outlet() {
		if res.Command == Commit {
			return res.Ack
		}
	}
	return false
}

func (w *Writer) Error() error {
	w.requests.Inlet() <- Request{Command: Error}
	for res := range w.responses.Outlet() {
		if res.Command == Error {
			return res.Error
		}
	}
	return nil
}

func (w *Writer) Close() error {
	w.requests.Close()
	for range w.responses.Outlet() {
	}
	return w.shutdown.Close()
}
