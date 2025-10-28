// Copyright 2025 Synnax Labs, Inc.
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
	"sync"

	"github.com/synnaxlabs/arc/lsp"
	"github.com/synnaxlabs/freighter"
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
)

type JSONRPCMessage struct {
	Content string `json:"content" msgpack:"content"`
}

type streamAdapter struct {
	stream      freighter.ServerStream[JSONRPCMessage, JSONRPCMessage]
	mu          sync.Mutex
	buffer      []byte
	writeBuffer []byte
	closed      bool
}

func (s *streamAdapter) Read(p []byte) (n int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
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
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return 0, io.ErrClosedPipe
	}
	s.writeBuffer = append(s.writeBuffer, p...)
	for {
		var (
			headerEnd     = -1
			contentLength = -1
			data          = s.writeBuffer
		)
		for i := 0; i < len(data)-16; i++ {
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
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	return nil
}

func ServeFreighter(
	ctx context.Context,
	server *lsp.Server,
	stream freighter.ServerStream[JSONRPCMessage, JSONRPCMessage],
) error {
	var (
		adapter = &streamAdapter{stream: stream}
		conn    = jsonrpc2.NewConn(jsonrpc2.NewStream(adapter))
		client  = protocol.ClientDispatcher(conn, server.Logger())
	)
	server.SetClient(client)
	conn.Go(ctx, protocol.ServerHandler(server, nil))
	<-conn.Done()
	return conn.Err()
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
