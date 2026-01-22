// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package io_test

import (
	"io"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/errors"
	xio "github.com/synnaxlabs/x/io"
)

// mockReaderAtCloser is a test helper for SectionReaderAtCloser tests
type mockReaderAtCloser struct {
	err    error
	data   []byte
	closed bool
}

func (m *mockReaderAtCloser) ReadAt(p []byte, off int64) (n int, err error) {
	if off < 0 || off >= int64(len(m.data)) {
		return 0, io.EOF
	}
	n = copy(p, m.data[off:])
	if n < len(p) {
		err = io.EOF
	}
	return n, err
}

func (m *mockReaderAtCloser) Close() error {
	m.closed = true
	return m.err
}

var _ = Describe("SectionReaderAtCloser", func() {
	Describe("NewSectionReaderAtCloser", func() {
		It("should create a section reader with specified offset and length", func() {
			data := []byte("Hello, World! This is a test.")
			reader := &mockReaderAtCloser{data: data}

			// Create a section from offset 7, length 5 ("World")
			section := xio.NewSectionReaderAtCloser(reader, 7, 5)

			// Read from the section
			buf := make([]byte, 5)
			n, err := section.ReadAt(buf, 0)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(Equal(5))
			Expect(string(buf)).To(Equal("World"))
		})

		It("should handle reading at different offsets within the section", func() {
			data := []byte("0123456789ABCDEF")
			reader := &mockReaderAtCloser{data: data}

			// Create a section from offset 5, length 6 ("56789A")
			section := xio.NewSectionReaderAtCloser(reader, 5, 6)

			// Read from offset 2 within the section
			buf := make([]byte, 3)
			n, err := section.ReadAt(buf, 2)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(Equal(3))
			Expect(string(buf)).To(Equal("789"))
		})

		It("should return EOF when reading beyond section bounds", func() {
			data := []byte("Hello, World!")
			reader := &mockReaderAtCloser{data: data}

			// Create a small section
			section := xio.NewSectionReaderAtCloser(reader, 0, 5)

			// Try to read beyond the section
			buf := make([]byte, 10)
			n, err := section.ReadAt(buf, 0)
			Expect(err).To(Equal(io.EOF))
			Expect(n).To(Equal(5))
			Expect(string(buf[:n])).To(Equal("Hello"))
		})

		It("should handle zero-length sections", func() {
			data := []byte("Hello, World!")
			reader := &mockReaderAtCloser{data: data}

			// Create a zero-length section
			section := xio.NewSectionReaderAtCloser(reader, 5, 0)

			buf := make([]byte, 5)
			n, err := section.ReadAt(buf, 0)
			Expect(err).To(Equal(io.EOF))
			Expect(n).To(Equal(0))
		})

		It("should close the underlying reader", func() {
			data := []byte("Test data")
			reader := &mockReaderAtCloser{data: data}

			section := xio.NewSectionReaderAtCloser(reader, 0, 4)

			Expect(reader.closed).To(BeFalse())
			Expect(section.Close()).To(Succeed())
			Expect(reader.closed).To(BeTrue())
		})

		It("should propagate close errors", func() {
			data := []byte("Test data")
			expectedErr := errors.New("close failed")
			reader := &mockReaderAtCloser{data: data, err: expectedErr}

			section := xio.NewSectionReaderAtCloser(reader, 0, 4)

			err := section.Close()
			Expect(err).To(MatchError(expectedErr))
			Expect(reader.closed).To(BeTrue())
		})

		It("should handle multiple reads from the same section", func() {
			data := []byte("ABCDEFGHIJKLMNOP")
			reader := &mockReaderAtCloser{data: data}

			// Create a section from offset 4, length 8 ("EFGHIJKL")
			section := xio.NewSectionReaderAtCloser(reader, 4, 8)

			// First read
			buf1 := make([]byte, 3)
			n, err := section.ReadAt(buf1, 0)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(Equal(3))
			Expect(string(buf1)).To(Equal("EFG"))

			// Second read from different offset
			buf2 := make([]byte, 4)
			n, err = section.ReadAt(buf2, 3)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(Equal(4))
			Expect(string(buf2)).To(Equal("HIJK"))

			// Third read overlapping
			buf3 := make([]byte, 5)
			n, err = section.ReadAt(buf3, 2)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(Equal(5))
			Expect(string(buf3)).To(Equal("GHIJK"))
		})

		It("should work with sections at the end of data", func() {
			data := []byte("Hello, World!")
			reader := &mockReaderAtCloser{data: data}

			// Create a section at the end
			section := xio.NewSectionReaderAtCloser(reader, 7, 6)

			buf := make([]byte, 10)
			n, err := section.ReadAt(buf, 0)
			Expect(err).To(Equal(io.EOF))
			Expect(n).To(Equal(6))
			Expect(string(buf[:n])).To(Equal("World!"))
		})
	})
})
