// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package transport

import (
	"context"
	"encoding/json"
	"io"
	"sync"
	"sync/atomic"

	"github.com/synnaxlabs/arc/lsp"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	xlsp "github.com/synnaxlabs/x/lsp"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
	"go.lsp.dev/jsonrpc2"
)

type JSONRPCMessage struct {
	Content string `json:"content" msgpack:"content"`
}

func (m *JSONRPCMessage) UnmarshalJSON(data []byte) error {
	type Alias JSONRPCMessage
	aux := &struct{ *Alias }{Alias: (*Alias)(m)}
	if err := json.Unmarshal(data, aux); err == nil {
		return nil
	}
	m.Content = string(data)
	return nil
}

const DefaultMaxContentLength = 10 * 1024 * 1024 // 10MB

var ErrContentLengthExceeded = errors.New(
	"[transport] - content length exceeded maximum allowed size",
)

// streamAdapter wraps a freighter stream to implement jsonrpc2.Stream. Each freighter
// message carries one fully-encoded JSON-RPC envelope in Content. jsonrpc2 v1 requires
// Write to be safe for concurrent use (unlike v0.10, which serialized writes in the
// Conn), so writeMu serializes Send: the underlying freighter/websocket stream panics
// on concurrent writes. Reads run on the Conn's single read goroutine.
type streamAdapter struct {
	stream           freighter.ServerStream[JSONRPCMessage, JSONRPCMessage]
	closed           atomic.Bool
	maxContentLength int
	// writeMu serializes concurrent Write callers (handler responses racing async
	// diagnostic/semantic-token notifications).
	writeMu sync.Mutex
	// ready gates the first Read until the server's client dispatcher has been wired
	// via SetClient, so an eager client cannot trigger a notification handler that
	// publishes diagnostics before the client is set.
	ready chan struct{}
}

var _ jsonrpc2.Stream = (*streamAdapter)(nil)

func (s *streamAdapter) Read(ctx context.Context) (jsonrpc2.Message, int64, error) {
	select {
	case <-s.ready:
	case <-ctx.Done():
		return nil, 0, ctx.Err()
	}
	if s.closed.Load() {
		return nil, 0, io.EOF
	}
	msg, err := s.stream.Receive()
	if err != nil {
		return nil, 0, err
	}
	if len(msg.Content) > s.maxContentLength {
		return nil, 0, ErrContentLengthExceeded
	}
	decoded, err := jsonrpc2.DecodeMessage([]byte(msg.Content))
	if err != nil {
		return nil, 0, err
	}
	return decoded, int64(len(msg.Content)), nil
}

func (s *streamAdapter) Write(_ context.Context, msg jsonrpc2.Message) (int64, error) {
	if s.closed.Load() {
		return 0, io.ErrClosedPipe
	}
	data, err := jsonrpc2.EncodeMessage(msg)
	if err != nil {
		return 0, err
	}
	if len(data) > s.maxContentLength {
		return 0, ErrContentLengthExceeded
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if err := s.stream.Send(JSONRPCMessage{Content: string(data)}); err != nil {
		return 0, err
	}
	return int64(len(data)), nil
}

func (s *streamAdapter) Close() error {
	s.closed.Store(true)
	return nil
}

type Config struct {
	Server           *lsp.Server
	Stream           freighter.ServerStream[JSONRPCMessage, JSONRPCMessage]
	MaxContentLength int
}

var _ config.Config[Config] = (*Config)(nil)

// Validate implements config.Config.
func (c Config) Validate() error {
	v := validate.New("arc.lsp.transport.freighter")
	validate.NotNil(v, "server", c.Server)
	validate.NotNil(v, "stream", c.Stream)
	validate.Positive(v, "max_content_length", c.MaxContentLength)
	return v.Error()
}

// Override implements config.Config.
func (c Config) Override(other Config) Config {
	c.Stream = override.Nil(c.Stream, other.Stream)
	c.Server = override.Nil(c.Server, other.Server)
	c.MaxContentLength = override.Numeric(c.MaxContentLength, other.MaxContentLength)
	return c
}

func ServeFreighter(ctx context.Context, cfgs ...Config) (err error) {
	cfg, err := config.New(Config{MaxContentLength: DefaultMaxContentLength}, cfgs...)
	if err != nil {
		return err
	}
	adapter := &streamAdapter{
		stream:           cfg.Stream,
		maxContentLength: cfg.MaxContentLength,
		ready:            make(chan struct{}),
	}
	conn, client := xlsp.NewConn(ctx, cfg.Server, adapter)
	defer func() {
		err = errors.Combine(err, conn.Close())
	}()
	cfg.Server.SetClient(client)
	close(adapter.ready)
	<-conn.Done()
	return conn.Err()
}
