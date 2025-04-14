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
	requests  confluence.Inlet[Request]
	responses confluence.Outlet[Response]
	wg        signal.WaitGroup
	shutdown  io.Closer
}

// Write implements Writer.
func (w *Writer) Write(frame core.Frame) error {
	select {
	case <-w.responses.Outlet():
		return w.Close()
	case w.requests.Inlet() <- Request{Command: Data, Frame: frame}:
	}
	return nil
}

func (w *Writer) Commit() error {
	select {
	case <-w.responses.Outlet():
		return w.Close()
	case w.requests.Inlet() <- Request{Command: Commit}:
	}
	for res := range w.responses.Outlet() {
		if res.Command == Commit {
			return nil
		}
	}
	return w.Close()
}

func (w *Writer) Close() error {
	w.requests.Close()
	for range w.responses.Outlet() {
	}
	return w.shutdown.Close()
}
