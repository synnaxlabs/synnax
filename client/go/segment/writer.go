// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package segment

import (
	"github.com/cockroachdb/errors"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/freighter/ferrors"
	"github.com/synnaxlabs/synnax/pkg/api"
	"sync"
)

type Writer interface {
	Write(segments []api.Segment) bool
	Error() error
	Close() error
}

type WriterStream = freighter.ClientStream[api.FrameWriterRequest, api.FrameWriterResponse]

type writer struct {
	stream WriterStream
	mu     struct {
		sync.Mutex
		err         error
		receivedRes chan struct{}
	}
}

func NewWriter(stream WriterStream, keys ...string) (Writer, error) {
	w := &writer{stream: stream}
	if err := w.stream.Send(api.FrameWriterRequest{OpenKeys: keys}); err != nil {
		return nil, err
	}
	w.mu.receivedRes = make(chan struct{}, 1)
	res, err := w.stream.Receive()
	if err != nil {
		return nil, err
	}
	if !res.Ack {
		return nil, errors.New("failed to open writer")
	}
	go w.receiveErrors()
	return w, nil
}

func (w *writer) Write(segments []api.Segment) bool {
	select {
	case <-w.mu.receivedRes:
		return false
	default:
		err := w.stream.Send(api.FrameWriterRequest{Segments: segments})
		w.maybeSetError(err)
		return err == nil
	}
}

func (w *writer) Error() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.mu.err != nil {
		w.mu.receivedRes = make(chan struct{}, 1)
		err := w.mu.err
		w.mu.err = nil
		return err
	}
	return nil
}

func (w *writer) Close() error {
	if err := w.stream.CloseSend(); err != nil {
		return err
	}
	<-w.mu.receivedRes
	err := w.Error()
	if errors.Is(err, freighter.EOF) {
		return nil
	}
	return err
}

func (w *writer) receiveErrors() {
	for {
		msg, err := w.stream.Receive()
		if err != nil {
			w.maybeSetError(err)
			return
		}
		w.maybeSetError(ferrors.Decode(msg.Err))
	}
}

func (w *writer) maybeSetError(err error) {
	if err == nil {
		return
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.mu.err != nil {
		return
	}
	w.mu.err = err
	close(w.mu.receivedRes)
}
