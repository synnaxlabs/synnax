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
	"fmt"
	"io"
	"sync/atomic"

	"github.com/synnaxlabs/arc/lsp"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
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

var ErrContentLengthExceeded = errors.New("[transport] - content length exceeded maximum allowed size")

// streamAdapter wraps a freighter stream to implement io.ReadWriteCloser for jsonrpc2.
// Thread safety is handled by jsonrpc2.Conn which serializes writes via writeMu and
// uses a single goroutine for reads. We only need an atomic for the closed flag.
type streamAdapter struct {
	stream           freighter.ServerStream[JSONRPCMessage, JSONRPCMessage]
	buffer           []byte
	writeBuffer      []byte
	closed           atomic.Bool
	maxContentLength int
}

func (s *streamAdapter) Read(p []byte) (n int, err error) {
	if len(s.buffer) > 0 {
		n = copy(p, s.buffer)
		s.buffer = s.buffer[n:]
		return n, nil
	}
	msg, err := s.stream.Receive()
	if err != nil {
		return 0, err
	}
	var (
		contentBytes = []byte(msg.Content)
		header       = fmt.Sprintf("Content-Length: %d\r\n\r\n", len(contentBytes))
		headerBytes  = []byte(header)
		fullMessage  = make([]byte, len(headerBytes)+len(contentBytes))
	)
	copy(fullMessage, headerBytes)
	copy(fullMessage[len(headerBytes):], contentBytes)
	n = copy(p, fullMessage)
	if n < len(fullMessage) {
		s.buffer = fullMessage[n:]
	}
	return n, nil
}

func (s *streamAdapter) Write(p []byte) (int, error) {
	if s.closed.Load() {
		return 0, io.ErrClosedPipe
	}
	s.writeBuffer = append(s.writeBuffer, p...)
	for {
		var (
			headerEnd     = -1
			contentLength = -1
			data          = s.writeBuffer
		)
		if len(data) < 16 {
			break
		}
		for i := 0; i <= len(data)-16; i++ {
			if data[i] == 'C' && string(data[i:i+15]) == "Content-Length:" {
				j := i + 15
				for j < len(data) && (data[j] == ' ' || data[j] == '\t') {
					j++
				}
				lenStart := j
				for j < len(data) && data[j] >= '0' && data[j] <= '9' {
					j++
				}
				if j > lenStart {
					if _, err := fmt.Sscanf(string(data[lenStart:j]), "%d", &contentLength); err != nil {
						return 0, err
					}
				}
				for j < len(data)-3 {
					if data[j] == '\r' && data[j+1] == '\n' && data[j+2] == '\r' && data[j+3] == '\n' {
						headerEnd = j + 4
						break
					} else if data[j] == '\n' && data[j+1] == '\n' {
						headerEnd = j + 2
						break
					}
					j++
				}
				break
			}
		}
		if headerEnd == -1 || contentLength == -1 || len(data) < headerEnd+contentLength {
			break
		}
		if contentLength < 0 || contentLength > s.maxContentLength {
			return 0, ErrContentLengthExceeded
		}
		jsonContent := data[headerEnd : headerEnd+contentLength]
		msg := JSONRPCMessage{Content: string(jsonContent)}
		if err := s.stream.Send(msg); err != nil {
			return 0, err
		}
		s.writeBuffer = data[headerEnd+contentLength:]
	}
	return len(p), nil
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

var DefaultConfig = Config{MaxContentLength: DefaultMaxContentLength}

// connClient wraps a protocol.Client with access to the underlying jsonrpc2.Conn
// to support LSP methods missing from go.lsp.dev/protocol@v0.12.0.
type connClient struct {
	protocol.Client
	conn jsonrpc2.Conn
}

func (c *connClient) SemanticTokensRefresh(ctx context.Context) error {
	return protocol.Call(ctx, c.conn, protocol.MethodSemanticTokensRefresh, nil, nil)
}

func ServeFreighter(ctx context.Context, cfgs ...Config) error {
	cfg, err := config.New(DefaultConfig, cfgs...)
	if err != nil {
		return err
	}
	var (
		adapter = &streamAdapter{stream: cfg.Stream, maxContentLength: cfg.MaxContentLength}
		conn    = jsonrpc2.NewConn(jsonrpc2.NewStream(adapter))
	)
	defer func() {
		err = errors.Combine(err, conn.Close())
	}()
	cfg.Server.SetClient(&connClient{
		Client: protocol.ClientDispatcher(conn, cfg.Server.Logger()),
		conn:   conn,
	})
	conn.Go(ctx, protocol.ServerHandler(cfg.Server, nil))
	<-conn.Done()
	err = conn.Err()
	return err
}
