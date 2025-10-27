// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package transport_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/arc/lsp"
	"github.com/synnaxlabs/arc/lsp/transport"
	"github.com/synnaxlabs/freighter/fmock"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Freighter Transport", func() {
	Describe("streamAdapter", func() {
		var (
			serverStream *fmock.ServerStream[transport.JSONRPCMessage, transport.JSONRPCMessage]
			clientStream *fmock.ClientStream[transport.JSONRPCMessage, transport.JSONRPCMessage]
			ctx          context.Context
			cancel       context.CancelFunc
		)
		BeforeEach(func() {
			ctx, cancel = context.WithCancel(context.Background())
			clientStream, serverStream = fmock.NewStreams[transport.JSONRPCMessage, transport.JSONRPCMessage](ctx)
		})
		AfterEach(func() {
			cancel()
		})
		Describe("Read", func() {
			It("Should read JSON-RPC message and add Content-Length header", func() {
				adapter := &streamAdapter{stream: serverStream}
				jsonContent := `{"jsonrpc":"2.0","id":1,"method":"test"}`
				done := make(chan struct{})
				go func() {
					defer GinkgoRecover()
					defer close(done)
					Expect(clientStream.Send(transport.JSONRPCMessage{Content: jsonContent})).To(Succeed())
				}()
				buf := make([]byte, 1024)
				n, err := adapter.Read(buf)
				Expect(err).ToNot(HaveOccurred())
				Expect(n).To(BeNumerically(">", 0))
				expected := fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(jsonContent), jsonContent)
				Expect(string(buf[:n])).To(Equal(expected))
				Eventually(done).Should(BeClosed())
			})
			It("Should handle partial reads with buffering", func() {
				adapter := &streamAdapter{stream: serverStream}
				jsonContent := `{"jsonrpc":"2.0","id":1,"method":"test"}`
				done := make(chan struct{})
				go func() {
					defer GinkgoRecover()
					defer close(done)
					Expect(clientStream.Send(transport.JSONRPCMessage{Content: jsonContent})).To(Succeed())
				}()
				smallBuf := make([]byte, 10)
				n, err := adapter.Read(smallBuf)
				Expect(err).ToNot(HaveOccurred())
				Expect(n).To(Equal(10))
				secondBuf := make([]byte, 1024)
				n2, err := adapter.Read(secondBuf)
				Expect(err).ToNot(HaveOccurred())
				Expect(n2).To(BeNumerically(">", 0))
				Eventually(done).Should(BeClosed())
			})
			It("Should return error when stream receive fails", func() {
				adapter := &streamAdapter{stream: serverStream}
				done := make(chan struct{})
				go func() {
					defer GinkgoRecover()
					defer close(done)
					Expect(clientStream.CloseSend()).To(Succeed())
				}()
				buf := make([]byte, 1024)
				_, err := adapter.Read(buf)
				Expect(err).To(HaveOccurred())
				Eventually(done).Should(BeClosed())
			})
		})
		Describe("Write", func() {
			It("Should parse Content-Length header and send JSON-RPC message", func() {
				adapter := &streamAdapter{stream: serverStream}
				jsonContent := `{"jsonrpc":"2.0","id":1,"method":"test"}`
				message := fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(jsonContent), jsonContent)
				done := make(chan struct{})
				go func() {
					defer GinkgoRecover()
					defer close(done)
					msg := MustSucceed(clientStream.Receive())
					Expect(msg.Content).To(Equal(jsonContent))
				}()
				n, err := adapter.Write([]byte(message))
				Expect(err).ToNot(HaveOccurred())
				Expect(n).To(Equal(len(message)))
				Eventually(done).Should(BeClosed())
			})
			It("Should handle multiple messages in single write", func() {
				adapter := &streamAdapter{stream: serverStream}
				json1 := `{"jsonrpc":"2.0","id":1}`
				json2 := `{"jsonrpc":"2.0","id":2}`
				msg1 := fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(json1), json1)
				msg2 := fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(json2), json2)
				combined := msg1 + msg2
				var receivedCount atomic.Int32
				go func() {
					defer GinkgoRecover()
					for i := 0; i < 2; i++ {
						msg := MustSucceed(clientStream.Receive())
						if msg.Content == json1 || msg.Content == json2 {
							receivedCount.Add(1)
						}
					}
				}()
				n, err := adapter.Write([]byte(combined))
				Expect(err).ToNot(HaveOccurred())
				Expect(n).To(Equal(len(combined)))
				Eventually(func() int32 { return receivedCount.Load() }).Should(Equal(int32(2)))
			})
			It("Should handle partial writes with buffering", func() {
				adapter := &streamAdapter{stream: serverStream}
				jsonContent := `{"jsonrpc":"2.0","id":1}`
				fullMessage := fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(jsonContent), jsonContent)
				part1 := fullMessage[:20]
				part2 := fullMessage[20:]
				n1, err := adapter.Write([]byte(part1))
				Expect(err).ToNot(HaveOccurred())
				Expect(n1).To(Equal(len(part1)))
				done := make(chan struct{})
				go func() {
					defer GinkgoRecover()
					defer close(done)
					msg := MustSucceed(clientStream.Receive())
					Expect(msg.Content).To(Equal(jsonContent))
				}()
				n2, err := adapter.Write([]byte(part2))
				Expect(err).ToNot(HaveOccurred())
				Expect(n2).To(Equal(len(part2)))
				Eventually(done).Should(BeClosed())
			})
			It("Should handle headers with LF instead of CRLF", func() {
				adapter := &streamAdapter{stream: serverStream}
				jsonContent := `{"jsonrpc":"2.0","id":1}`
				message := fmt.Sprintf("Content-Length: %d\n\n%s", len(jsonContent), jsonContent)
				done := make(chan struct{})
				go func() {
					defer GinkgoRecover()
					defer close(done)
					msg := MustSucceed(clientStream.Receive())
					Expect(msg.Content).To(Equal(jsonContent))
				}()
				n, err := adapter.Write([]byte(message))
				Expect(err).ToNot(HaveOccurred())
				Expect(n).To(Equal(len(message)))
				Eventually(done).Should(BeClosed())
			})
			It("Should return error when closed", func() {
				adapter := &streamAdapter{stream: serverStream, closed: true}
				_, err := adapter.Write([]byte("test"))
				Expect(err).To(Equal(io.ErrClosedPipe))
			})
		})
		Describe("Close", func() {
			It("Should mark adapter as closed", func() {
				adapter := &streamAdapter{stream: serverStream}
				Expect(adapter.Close()).To(Succeed())
				_, err := adapter.Write([]byte("test"))
				Expect(err).To(Equal(io.ErrClosedPipe))
			})
		})
	})
	Describe("ServeFreighter", func() {
		It("Should create LSP connection and start serving", func() {
			ins := alamos.New("test")
			server := MustSucceed(lsp.New(lsp.Config{Instrumentation: ins}))
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			clientStream, serverStream := fmock.NewStreams[transport.JSONRPCMessage, transport.JSONRPCMessage](ctx)
			errChan := make(chan error, 1)
			go func() {
				errChan <- transport.ServeFreighter(ctx, server, serverStream)
			}()
			Expect(clientStream.CloseSend()).To(Succeed())
			Eventually(errChan, time.Second).Should(Receive())
		})
	})
	Describe("JSONRPCMessage", func() {
		Describe("UnmarshalJSON", func() {
			It("Should unmarshal valid JSON object", func() {
				var msg transport.JSONRPCMessage
				input := `{"content":"test message"}`
				Expect(json.Unmarshal([]byte(input), &msg)).To(Succeed())
				Expect(msg.Content).To(Equal("test message"))
			})
			It("Should handle raw string as content", func() {
				var msg transport.JSONRPCMessage
				input := `"raw string content"`
				Expect(json.Unmarshal([]byte(input), &msg)).To(Succeed())
				Expect(msg.Content).To(Equal(input))
			})
			It("Should handle empty object", func() {
				var msg transport.JSONRPCMessage
				input := `{}`
				Expect(json.Unmarshal([]byte(input), &msg)).To(Succeed())
				Expect(msg.Content).To(Equal(""))
			})
		})
	})
})

type streamAdapter struct {
	stream      *fmock.ServerStream[transport.JSONRPCMessage, transport.JSONRPCMessage]
	buffer      []byte
	writeBuffer []byte
	closed      bool
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
	contentBytes := []byte(msg.Content)
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(contentBytes))
	headerBytes := []byte(header)
	fullMessage := make([]byte, len(headerBytes)+len(contentBytes))
	copy(fullMessage, headerBytes)
	copy(fullMessage[len(headerBytes):], contentBytes)
	n = copy(p, fullMessage)
	if n < len(fullMessage) {
		s.buffer = fullMessage[n:]
	}
	return n, nil
}

func (s *streamAdapter) Write(p []byte) (int, error) {
	if s.closed {
		return 0, io.ErrClosedPipe
	}
	s.writeBuffer = append(s.writeBuffer, p...)
	for {
		headerEnd := -1
		contentLength := -1
		data := s.writeBuffer
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
		msg := transport.JSONRPCMessage{Content: string(jsonContent)}
		if err := s.stream.Send(msg); err != nil {
			return 0, err
		}
		s.writeBuffer = data[headerEnd+contentLength:]
	}
	return len(p), nil
}

func (s *streamAdapter) Close() error {
	s.closed = true
	return nil
}
