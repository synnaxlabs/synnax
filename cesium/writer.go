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
	"github.com/synnaxlabs/cesium/internal/resource"
	"github.com/synnaxlabs/x/confluence"
	xcontrol "github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/errors"
)

// Writer writes frames to a set of channels and commits them. A Writer is not safe for
// concurrent use.
//
// The first error that is not control.ErrUnauthorized breaks the writer: every later
// call returns that error, and Close returns it after it releases the writer. An error
// that wraps control.ErrUnauthorized leaves the writer open, so that it can regain
// control.
type Writer struct {
	WriterConfig
	internal []*idxWriter
	virtual  *virtualWriter
	// keyToIdx maps every channel key the writer is responsible for to its owning
	// idxWriter.
	keyToIdx           map[ChannelKey]*idxWriter
	relay              confluence.Inlet[relayResponse]
	relayStopped       <-chan struct{}
	updateDBControl    func(u ControlUpdate)
	authorizeEndDomain func(index ChannelKey, own map[ChannelKey]*unaryWriterState) error
	// err is the error that broke the writer.
	err    error
	closed bool
}

// ErrWriterClosed is returned by a call to a Writer that is closed.
var ErrWriterClosed = resource.NewClosedError("cesium.writer")

// check returns the error that a call to the writer must return before it runs.
func (w *Writer) check() error {
	if w.err != nil {
		return w.err
	}
	if w.closed {
		return ErrWriterClosed
	}
	return nil
}

// fail breaks the writer with err unless err wraps control.ErrUnauthorized, and
// returns err.
func (w *Writer) fail(err error) error {
	if err != nil && !errors.Is(err, xcontrol.ErrUnauthorized) && w.err == nil {
		w.err = err
	}
	return err
}

// Close releases the writer. It does not commit. It returns the error that broke the
// writer, if any, joined with any error from the release. A second Close returns the
// same error.
func (w *Writer) Close() error {
	if !w.closed {
		w.err = errors.Combine(w.err, w.release())
	}
	return w.err
}

// release closes the internal writers and returns only the errors from closing them.
func (w *Writer) release() error {
	w.closed = true
	var (
		err    error
		update = ControlUpdate{
			Transfers: make([]xcontrol.Transfer[ChannelKey], 0, len(w.internal)),
		}
	)
	for _, idx := range w.internal {
		u, errClose := idx.Close()
		if errClose != nil {
			err = errors.Join(err, errClose)
		} else {
			update.Transfers = append(update.Transfers, u.Transfers...)
		}
	}
	if w.virtual.internal != nil {
		u, errClose := w.virtual.Close()
		if errClose != nil {
			err = errors.Join(err, errClose)
		} else {
			update.Transfers = append(update.Transfers, u.Transfers...)
		}
	}
	if len(update.Transfers) > 0 {
		w.updateDBControl(update)
	}
	return err
}
