// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package http

import (
	"context"
	"time"

	ws "github.com/fasthttp/websocket"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/x/encoding"
	"github.com/synnaxlabs/x/errors"
	"go.uber.org/zap"
)

// WSMessageType is used to differentiate between the different types of messages used
// to implement the websocket stream transport.
type WSMessageType string

const (
	// WSMessageTypeData is used for normal data movement between the ClientStream and
	// ServerStream implementations.
	WSMessageTypeData WSMessageType = "data"
	// WSMessageTypeClose is used to signal the end of the stream. We need to use this
	// instead of the regular websocket Close message because the 'reason' can't have
	// more than 123 bytes.
	WSMessageTypeClose WSMessageType = "close"
	// WSMessageTypeOpen is used to acknowledge the successful opening of the stream.
	// We need to do this to correctly handle the case where middleware returns an
	// error early. We can't just use the regular HTTP request/response cycle because
	// JavaScript implementations of WebSocket don't allow for accessing the response
	// body.
	WSMessageTypeOpen WSMessageType = "open"
)

// WSMessage wraps a user payload with additional information needed for the websocket
// transport to correctly implement the Stream interface. Namely, we need a custom close
// WSMessage type to correctly encode and transfer information about a closure error
// across the socket.
type WSMessage[P freighter.Payload] struct {
	// Payload is the user payload to send if the WSMessage type is WSMessageTypeData.
	Payload P `json:"payload" msgpack:"payload"`
	// Err is the error payload to send if the WSMessage type is WSMessageTypeClose.
	Err errors.Payload `json:"error" msgpack:"error"`
	// Type represents the type of WSMessage being sent. One of WSMessageTypeData or
	// WSMessageTypeClose.
	Type WSMessageType `json:"type" msgpack:"type"`
}

const (
	contextCancelledCloseCode = ws.CloseGoingAway
	closeReadWriteDeadline    = 500 * time.Millisecond
)

func newStreamCore[RQ, RS freighter.Payload](
	cfg coreConfig,
	serverShutdownSig <-chan struct{},
) streamCore[RQ, RS] {
	b := streamCore[RQ, RS]{
		serverShutdownSig:  serverShutdownSig,
		normalShutdownSig:  make(chan struct{}),
		successfulShutdown: make(chan struct{}),
		coreConfig:         cfg,
	}
	go b.listenForContextCancellation()
	return b
}

type coreConfig struct {
	codec encoding.Codec
	conn  *ws.Conn
	alamos.Instrumentation
	writeDeadline time.Duration
}

// streamCore is the common functionality implemented by both the client and server
// streams.
type streamCore[I, O freighter.Payload] struct {
	peerCloseErr       error
	serverShutdownSig  <-chan struct{}
	normalShutdownSig  chan struct{}
	successfulShutdown chan struct{}
	coreConfig
}

func (c *streamCore[I, O]) send(msg WSMessage[O]) error {
	if c.writeDeadline > 0 {
		if err := c.conn.SetWriteDeadline(time.Now().Add(c.writeDeadline)); err != nil {
			return err
		}
	}
	w, err := c.conn.NextWriter(ws.BinaryMessage)
	if err != nil {
		return err
	}
	err = c.codec.EncodeStream(context.TODO(), w, msg)
	return errors.Combine(err, w.Close())
}

func (c *streamCore[I, O]) receiveRaw() (WSMessage[I], error) {
	_, r, err := c.conn.NextReader()
	if err != nil {
		return WSMessage[I]{}, err
	}
	var msg WSMessage[I]
	return msg, c.codec.DecodeStream(context.TODO(), r, &msg)
}

func (c *streamCore[I, O]) Receive() (I, error) {
	if c.peerCloseErr != nil {
		var i I
		return i, c.peerCloseErr
	}
	msg, err := c.receiveRaw()
	if err != nil {
		if ws.IsCloseError(
			err,
			ws.CloseNormalClosure,
			ws.CloseNoStatusReceived,
			ws.CloseAbnormalClosure,
		) {
			c.peerCloseErr = freighter.EOF
		} else if ws.IsCloseError(err, contextCancelledCloseCode) {
			c.peerCloseErr = context.Canceled
		} else {
			c.peerCloseErr = freighter.ErrStreamClosed
		}
		c.peerCloseErr = errors.WithStack(c.peerCloseErr)
		var i I
		return i, c.peerCloseErr
	}
	if msg.Type == WSMessageTypeClose {
		c.peerCloseErr = errors.Decode(context.TODO(), msg.Err)
		if c.peerCloseErr == nil {
			c.peerCloseErr = freighter.EOF
		}
	}
	return msg.Payload, c.peerCloseErr
}

func (c *streamCore[I, O]) close() error {
	close(c.normalShutdownSig)
	<-c.successfulShutdown
	return c.conn.Close()
}

// listenForContextCancellation is a goroutine that listens for the context to be
// canceled and shuts down the stream forcefully if it is. We need this as the websocket
// implementation itself doesn't support context cancellation.
func (c *streamCore[I, O]) listenForContextCancellation() {
	defer close(c.successfulShutdown)
	select {
	case <-c.normalShutdownSig:
		return
	case <-c.serverShutdownSig:
		if err := c.conn.WriteControl(
			ws.CloseMessage,
			ws.FormatCloseMessage(contextCancelledCloseCode, ""),
			time.Now().Add(time.Second),
		); err != nil && !errors.Is(err, ws.ErrCloseSent) {
			c.L.Error("error sending close message: %v \n", zap.Error(err))
		}
	}
}
