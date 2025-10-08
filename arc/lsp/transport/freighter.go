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

// LSPMessage represents a single JSON-RPC message (request, response, or notification)
type LSPMessage struct {
	Content string `json:"content" msgpack:"content"`
}

// streamAdapter adapts a Freighter bidirectional stream to io.ReadWriteCloser
// for use with the jsonrpc2 library.
// On Read: adds Content-Length headers to raw JSON for jsonrpc2
// On Write: strips Content-Length headers before sending to Freighter
type streamAdapter struct {
	stream      freighter.ServerStream[LSPMessage, LSPMessage]
	mu          sync.Mutex
	buffer      []byte
	writeBuffer []byte
	closed      bool
}

// Read implements io.Reader
// Receives raw JSON-RPC from Freighter, adds Content-Length headers for jsonrpc2
func (s *streamAdapter) Read(p []byte) (n int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// If we have buffered data, return it first
	if len(s.buffer) > 0 {
		n = copy(p, s.buffer)
		s.buffer = s.buffer[n:]
		return n, nil
	}

	// Receive next message from stream
	msg, err := s.stream.Receive()
	if err != nil {
		if err == freighter.EOF {
			return 0, io.EOF
		}
		return 0, err
	}

	// Add Content-Length header for jsonrpc2 library
	// Format: Content-Length: <length>\r\n\r\n<json>
	contentBytes := []byte(msg.Content)
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(contentBytes))
	headerBytes := []byte(header)

	// Combine header and content
	fullMessage := make([]byte, len(headerBytes)+len(contentBytes))
	copy(fullMessage, headerBytes)
	copy(fullMessage[len(headerBytes):], contentBytes)

	// Copy to output buffer
	n = copy(p, fullMessage)

	// If we couldn't fit all the data, buffer the rest
	if n < len(fullMessage) {
		s.buffer = fullMessage[n:]
	}

	return n, nil
}

// Write implements io.Writer
// Receives Content-Length formatted messages from jsonrpc2, strips headers for Freighter
func (s *streamAdapter) Write(p []byte) (n int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return 0, io.ErrClosedPipe
	}

	// Accumulate data in write buffer
	s.writeBuffer = append(s.writeBuffer, p...)

	// Try to extract complete messages
	for {
		// Look for Content-Length header
		headerEnd := -1
		contentLength := -1
		data := s.writeBuffer

		// Find Content-Length header
		for i := 0; i < len(data)-16; i++ {
			if data[i] == 'C' && string(data[i:i+15]) == "Content-Length:" {
				// Parse the length
				j := i + 15
				for j < len(data) && (data[j] == ' ' || data[j] == '\t') {
					j++
				}
				lenStart := j
				for j < len(data) && data[j] >= '0' && data[j] <= '9' {
					j++
				}
				if j > lenStart {
					fmt.Sscanf(string(data[lenStart:j]), "%d", &contentLength)
				}
				// Find end of headers (\r\n\r\n or \n\n)
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

		// If we don't have a complete header yet, wait for more data
		if headerEnd == -1 || contentLength == -1 {
			break
		}

		// Check if we have the complete message
		if len(data) < headerEnd+contentLength {
			break
		}

		// Extract the JSON content (without headers)
		jsonContent := data[headerEnd : headerEnd+contentLength]

		// Send message through Freighter stream
		msg := LSPMessage{Content: string(jsonContent)}
		if err := s.stream.Send(msg); err != nil {
			return 0, err
		}

		// Remove processed message from buffer
		s.writeBuffer = data[headerEnd+contentLength:]
	}

	return len(p), nil
}

// Close implements io.Closer
func (s *streamAdapter) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	return nil
}

// ServeFreighter serves the Arc LSP server over a Freighter StreamServer
func ServeFreighter(
	ctx context.Context,
	server *lsp.Server,
	stream freighter.ServerStream[LSPMessage, LSPMessage],
) error {
	adapter := &streamAdapter{stream: stream}
	conn := jsonrpc2.NewConn(jsonrpc2.NewStream(adapter))

	// Create protocol client for server notifications
	client := protocol.ClientDispatcher(conn, server.Logger())
	server.SetClient(client)

	// Start handling messages
	conn.Go(ctx, protocol.ServerHandler(server, nil))

	// Wait for connection to close
	<-conn.Done()
	return conn.Err()
}

// UnmarshalJSON custom unmarshaler to handle both string and object formats
func (m *LSPMessage) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as struct first
	type Alias LSPMessage
	aux := &struct{ *Alias }{Alias: (*Alias)(m)}
	if err := json.Unmarshal(data, aux); err == nil {
		return nil
	}

	// Fall back to treating the entire payload as content
	m.Content = string(data)
	return nil
}