// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package io_test

import (
	"bytes"
	"errors"
	"io"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	xio "github.com/synnaxlabs/x/io"
)

// testCloser is a test helper that tracks Close() calls
type testCloser struct {
	closed bool
	err    error
}

func (t *testCloser) Close() error {
	t.closed = true
	return t.err
}

var _ = Describe("CombinedReadWriteCloser", func() {
	Describe("CombineReadWriteCloser", func() {
		It("should combine reader, writer, and closer", func() {
			reader := strings.NewReader("test data")
			writer := &bytes.Buffer{}
			closer := &testCloser{}

			combined := xio.CombineReadWriteCloser(reader, writer, closer)

			// Test reading
			buf := make([]byte, 4)
			n, err := combined.Read(buf)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(Equal(4))
			Expect(string(buf)).To(Equal("test"))

			// Test writing
			n, err = combined.Write([]byte("hello"))
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(Equal(5))
			Expect(writer.String()).To(Equal("hello"))

			// Test closing
			Expect(combined.Close()).To(Succeed())
			Expect(closer.closed).To(BeTrue())
		})

		It("should handle nil closer gracefully", func() {
			reader := strings.NewReader("data")
			writer := &bytes.Buffer{}

			combined := xio.CombineReadWriteCloser(reader, writer, nil)

			// Close should not panic and should return nil
			Expect(combined.Close()).To(Succeed())
		})

		It("should propagate closer errors", func() {
			reader := strings.NewReader("data")
			writer := &bytes.Buffer{}
			expectedErr := errors.New("close failed")
			closer := &testCloser{err: expectedErr}

			combined := xio.CombineReadWriteCloser(reader, writer, closer)

			err := combined.Close()
			Expect(err).To(MatchError(expectedErr))
			Expect(closer.closed).To(BeTrue())
		})

		It("should handle EOF from reader", func() {
			reader := strings.NewReader("short")
			writer := &bytes.Buffer{}
			closer := &testCloser{}

			combined := xio.CombineReadWriteCloser(reader, writer, closer)

			// Read all data
			buf := make([]byte, 10)
			n, err := combined.Read(buf)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(Equal(5))
			Expect(string(buf[:n])).To(Equal("short"))

			// Next read should return EOF
			n, err = combined.Read(buf)
			Expect(err).To(Equal(io.EOF))
			Expect(n).To(Equal(0))
		})

		It("should handle multiple writes", func() {
			reader := strings.NewReader("")
			writer := &bytes.Buffer{}
			closer := &testCloser{}

			combined := xio.CombineReadWriteCloser(reader, writer, closer)

			// Multiple writes
			n, err := combined.Write([]byte("first "))
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(Equal(6))

			n, err = combined.Write([]byte("second "))
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(Equal(7))

			n, err = combined.Write([]byte("third"))
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(Equal(5))

			Expect(writer.String()).To(Equal("first second third"))
		})

		It("should handle empty reader and writer", func() {
			reader := strings.NewReader("")
			writer := &bytes.Buffer{}
			closer := &testCloser{}

			combined := xio.CombineReadWriteCloser(reader, writer, closer)

			// Read from empty reader
			buf := make([]byte, 10)
			n, err := combined.Read(buf)
			Expect(err).To(Equal(io.EOF))
			Expect(n).To(Equal(0))

			// Write to empty writer
			n, err = combined.Write([]byte("test"))
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(Equal(4))
			Expect(writer.String()).To(Equal("test"))
		})

		It("should handle concurrent read and write operations", func() {
			reader := strings.NewReader(strings.Repeat("a", 1000))
			writer := &bytes.Buffer{}
			closer := &testCloser{}

			combined := xio.CombineReadWriteCloser(reader, writer, closer)

			done := make(chan bool, 2)

			// Concurrent read
			go func() {
				defer GinkgoRecover()
				buf := make([]byte, 100)
				for {
					n, err := combined.Read(buf)
					if err == io.EOF {
						break
					}
					Expect(err).ToNot(HaveOccurred())
					Expect(n).To(BeNumerically(">", 0))
				}
				done <- true
			}()

			// Concurrent write
			go func() {
				defer GinkgoRecover()
				for i := 0; i < 10; i++ {
					n, err := combined.Write([]byte("test"))
					Expect(err).ToNot(HaveOccurred())
					Expect(n).To(Equal(4))
				}
				done <- true
			}()

			// Wait for both operations to complete
			Eventually(done).Should(Receive())
			Eventually(done).Should(Receive())

			// Verify write results
			Expect(writer.String()).To(Equal(strings.Repeat("test", 10)))
		})

		It("should allow multiple Close calls", func() {
			reader := strings.NewReader("data")
			writer := &bytes.Buffer{}
			closer := &testCloser{}

			combined := xio.CombineReadWriteCloser(reader, writer, closer)

			// First close
			Expect(combined.Close()).To(Succeed())
			Expect(closer.closed).To(BeTrue())

			// Reset for testing
			closer.closed = false

			// Second close should also work
			Expect(combined.Close()).To(Succeed())
			Expect(closer.closed).To(BeTrue())
		})

		It("should work with bytes.Reader and bytes.Buffer", func() {
			data := []byte("binary data\x00\x01\x02")
			reader := bytes.NewReader(data)
			writer := &bytes.Buffer{}
			closer := &testCloser{}

			combined := xio.CombineReadWriteCloser(reader, writer, closer)

			// Read binary data
			buf := make([]byte, len(data))
			n, err := combined.Read(buf)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(Equal(len(data)))
			Expect(buf).To(Equal(data))

			// Write binary data
			binaryData := []byte{0xFF, 0xFE, 0xFD}
			n, err = combined.Write(binaryData)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(Equal(3))
			Expect(writer.Bytes()).To(Equal(binaryData))
		})
	})

	Describe("Integration with io.Copy", func() {
		It("should work as source for io.Copy", func() {
			sourceData := "data to copy"
			reader := strings.NewReader(sourceData)
			writer := &bytes.Buffer{}
			closer := &testCloser{}

			combined := xio.CombineReadWriteCloser(reader, writer, closer)
			dest := &bytes.Buffer{}

			n, err := io.Copy(dest, combined)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(Equal(int64(len(sourceData))))
			Expect(dest.String()).To(Equal(sourceData))
		})

		It("should work as destination for io.Copy", func() {
			sourceData := "data to write"
			source := strings.NewReader(sourceData)
			reader := &bytes.Buffer{}
			writer := &bytes.Buffer{}
			closer := &testCloser{}

			combined := xio.CombineReadWriteCloser(reader, writer, closer)

			n, err := io.Copy(combined, source)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(Equal(int64(len(sourceData))))
			Expect(writer.String()).To(Equal(sourceData))
		})
	})
})
