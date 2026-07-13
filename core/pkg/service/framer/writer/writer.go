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
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
)

// These aliases and constants re-export the distribution-layer writer types so
// service-layer callers can refer to them through this package without importing the
// distribution layer directly.
type (
	// Command is an operation that can be executed on a Writer. Re-exported from
	// [writer.Command].
	Command = writer.Command
	// Mode configures the persistence and streaming behavior of a writer. Re-exported
	// from [writer.Mode].
	Mode = writer.Mode
	// Request represents a streaming call to a Writer. Re-exported from
	// [writer.Request].
	Request = writer.Request
	// Response represents a response to a streaming call to a Writer. Re-exported from
	// [writer.Response].
	Response = writer.Response
	// StreamWriter is a channel-based writer for telemetry frames. Re-exported from
	// [writer.StreamWriter].
	StreamWriter = writer.StreamWriter
)

const (
	// CommandOpen represents opening the writer. Re-exported from [writer.CommandOpen].
	CommandOpen = writer.CommandOpen
	// CommandWrite represents a call to Writer.Write. Re-exported from
	// [writer.CommandWrite].
	CommandWrite = writer.CommandWrite
	// CommandCommit represents a call to Writer.Commit. Re-exported from
	// [writer.CommandCommit].
	CommandCommit = writer.CommandCommit
	// CommandSetAuthority represents a call to Writer.SetAuthority. Re-exported from
	// [writer.CommandSetAuthority].
	CommandSetAuthority = writer.CommandSetAuthority
)

// ErrClosed is returned when a method is called on a Writer after it has been closed.
// Re-exported from [writer.ErrClosed].
var ErrClosed = writer.ErrClosed

// Writer is a buffered, synchronous wrapper around a StreamWriter that exposes a
// method-based interface for writing frames. Writer is not safe for concurrent use.
type Writer struct {
	requests  confluence.Inlet[Request]
	responses confluence.Outlet[Response]
	shutdown  io.Closer
	closeErr  error
	cfg       Config
}

// Write writes the given frame to the cluster, returning whether the write was
// authorized. When the writer was opened with Sync set to false, authorization failures
// are only surfaced on Commit or Close.
func (w *Writer) Write(frame frame.Frame) (authorized bool, err error) {
	res, err := w.exec(Request{Frame: frame, Command: CommandWrite}, *w.cfg.Sync)
	if err != nil {
		return false, err
	}
	authorized = !*w.cfg.Sync || res.Authorized
	return
}

// Commit commits all writes since the last commit, returning the end timestamp of the
// committed region.
func (w *Writer) Commit() (telem.TimeStamp, error) {
	res, err := w.exec(Request{Command: CommandCommit}, true)
	return res.End, err
}

// SetAuthority updates the control authorities the writer holds over its channels.
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

// Close closes the writer, releasing its control over the written region. Close returns
// any accumulated write error.
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
