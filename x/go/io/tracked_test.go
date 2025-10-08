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
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	xio "github.com/synnaxlabs/x/io"
	xfs "github.com/synnaxlabs/x/io/fs"
)

var _ = Describe("TrackedWriteCloser", func() {
	var (
		fs   xfs.FS
		file xfs.File
	)

	BeforeEach(func() {
		fs = xfs.NewMem()
	})

	AfterEach(func() {
		if file != nil {
			Expect(file.Close()).To(Succeed())
			file = nil
		}
	})

	Describe("NewTrackedWriteCloser", func() {
		It("should create a tracked writer at the end of an empty file", func() {
			var err error
			file, err = fs.Open("test.txt", os.O_CREATE|os.O_RDWR)
			Expect(err).ToNot(HaveOccurred())

			tracked, err := xio.NewTrackedWriteCloser(file)
			Expect(err).ToNot(HaveOccurred())

			Expect(tracked.Offset()).To(Equal(int64(0)))
			Expect(tracked.Len()).To(Equal(int64(0)))
		})

		It("should create a tracked writer at the end of an existing file", func() {
			var err error
			file, err = fs.Open("test.txt", os.O_CREATE|os.O_RDWR)
			Expect(err).ToNot(HaveOccurred())

			// Write some initial data
			_, err = file.Write([]byte("Initial content"))
			Expect(err).ToNot(HaveOccurred())

			tracked, err := xio.NewTrackedWriteCloser(file)
			Expect(err).ToNot(HaveOccurred())

			Expect(tracked.Offset()).To(Equal(int64(15))) // "Initial content" is 15 bytes
			Expect(tracked.Len()).To(Equal(int64(0)))
		})
	})

	Describe("Write", func() {
		It("should track bytes written", func() {
			var err error
			file, err = fs.Open("test.txt", os.O_CREATE|os.O_RDWR)
			Expect(err).ToNot(HaveOccurred())

			tracked, err := xio.NewTrackedWriteCloser(file)
			Expect(err).ToNot(HaveOccurred())

			// First write
			n, err := tracked.Write([]byte("Hello"))
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(Equal(5))
			Expect(tracked.Len()).To(Equal(int64(5)))
			Expect(tracked.Offset()).To(Equal(int64(0)))

			// Second write
			n, err = tracked.Write([]byte(" World"))
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(Equal(6))
			Expect(tracked.Len()).To(Equal(int64(11)))
			Expect(tracked.Offset()).To(Equal(int64(0)))
		})

		It("should handle multiple sequential writes", func() {
			var err error
			file, err = fs.Open("test.txt", os.O_CREATE|os.O_RDWR)
			Expect(err).ToNot(HaveOccurred())

			tracked, err := xio.NewTrackedWriteCloser(file)
			Expect(err).ToNot(HaveOccurred())

			writes := []string{"First ", "Second ", "Third"}
			totalLen := int64(0)

			for _, data := range writes {
				n, err := tracked.Write([]byte(data))
				Expect(err).ToNot(HaveOccurred())
				Expect(n).To(Equal(len(data)))
				totalLen += int64(n)
				Expect(tracked.Len()).To(Equal(totalLen))
			}

			Expect(tracked.Offset()).To(Equal(int64(0)))
		})

		It("should handle empty writes", func() {
			var err error
			file, err = fs.Open("test.txt", os.O_CREATE|os.O_RDWR)
			Expect(err).ToNot(HaveOccurred())

			tracked, err := xio.NewTrackedWriteCloser(file)
			Expect(err).ToNot(HaveOccurred())

			n, err := tracked.Write([]byte{})
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(Equal(0))
			Expect(tracked.Len()).To(Equal(int64(0)))
		})
	})

	Describe("Reset", func() {
		It("should reset the tracked length and update offset", func() {
			var err error
			file, err = fs.Open("test.txt", os.O_CREATE|os.O_RDWR)
			Expect(err).ToNot(HaveOccurred())

			tracked, err := xio.NewTrackedWriteCloser(file)
			Expect(err).ToNot(HaveOccurred())

			// Write some data
			_, err = tracked.Write([]byte("First batch"))
			Expect(err).ToNot(HaveOccurred())
			Expect(tracked.Len()).To(Equal(int64(11)))
			Expect(tracked.Offset()).To(Equal(int64(0)))

			// Reset
			tracked.Reset()
			Expect(tracked.Len()).To(Equal(int64(0)))
			Expect(tracked.Offset()).To(Equal(int64(11)))

			// Write more data after reset
			_, err = tracked.Write([]byte("Second"))
			Expect(err).ToNot(HaveOccurred())
			Expect(tracked.Len()).To(Equal(int64(6)))
			Expect(tracked.Offset()).To(Equal(int64(11)))
		})

		It("should handle multiple resets", func() {
			var err error
			file, err = fs.Open("test.txt", os.O_CREATE|os.O_RDWR)
			Expect(err).ToNot(HaveOccurred())

			tracked, err := xio.NewTrackedWriteCloser(file)
			Expect(err).ToNot(HaveOccurred())

			// First write and reset
			_, err = tracked.Write([]byte("AAA"))
			Expect(err).ToNot(HaveOccurred())
			tracked.Reset()
			Expect(tracked.Offset()).To(Equal(int64(3)))

			// Second write and reset
			_, err = tracked.Write([]byte("BBBB"))
			Expect(err).ToNot(HaveOccurred())
			tracked.Reset()
			Expect(tracked.Offset()).To(Equal(int64(7)))

			// Third write and reset
			_, err = tracked.Write([]byte("CC"))
			Expect(err).ToNot(HaveOccurred())
			tracked.Reset()
			Expect(tracked.Offset()).To(Equal(int64(9)))

			Expect(tracked.Len()).To(Equal(int64(0)))
		})

		It("should reset without any writes", func() {
			var err error
			file, err = fs.Open("test.txt", os.O_CREATE|os.O_RDWR)
			Expect(err).ToNot(HaveOccurred())

			tracked, err := xio.NewTrackedWriteCloser(file)
			Expect(err).ToNot(HaveOccurred())

			// Reset without writing
			tracked.Reset()
			Expect(tracked.Len()).To(Equal(int64(0)))
			Expect(tracked.Offset()).To(Equal(int64(0)))
		})
	})

	Describe("Close", func() {
		It("should close the underlying file", func() {
			var err error
			file, err = fs.Open("test.txt", os.O_CREATE|os.O_RDWR)
			Expect(err).ToNot(HaveOccurred())

			tracked, err := xio.NewTrackedWriteCloser(file)
			Expect(err).ToNot(HaveOccurred())

			_, err = tracked.Write([]byte("test"))
			Expect(err).ToNot(HaveOccurred())

			err = tracked.Close()
			Expect(err).ToNot(HaveOccurred())

			// Clear the file variable to prevent AfterEach from trying to close it again
			file = nil
		})
	})

	Describe("Integration tests", func() {
		It("should handle a complete write-reset-write workflow", func() {
			var err error
			file, err = fs.Open("test.txt", os.O_CREATE|os.O_RDWR)
			Expect(err).ToNot(HaveOccurred())

			// Write initial content directly to file
			_, err = file.Write([]byte("PREFIX:"))
			Expect(err).ToNot(HaveOccurred())

			tracked, err := xio.NewTrackedWriteCloser(file)
			Expect(err).ToNot(HaveOccurred())
			Expect(tracked.Offset()).To(Equal(int64(7)))

			// First batch of writes
			_, err = tracked.Write([]byte("batch1,"))
			Expect(err).ToNot(HaveOccurred())
			Expect(tracked.Len()).To(Equal(int64(7)))

			// Reset and second batch
			tracked.Reset()
			_, err = tracked.Write([]byte("batch2,"))
			Expect(err).ToNot(HaveOccurred())
			Expect(tracked.Len()).To(Equal(int64(7)))
			Expect(tracked.Offset()).To(Equal(int64(14)))

			// Reset and third batch
			tracked.Reset()
			_, err = tracked.Write([]byte("batch3"))
			Expect(err).ToNot(HaveOccurred())
			Expect(tracked.Len()).To(Equal(int64(6)))
			Expect(tracked.Offset()).To(Equal(int64(21)))

			// Verify file contents by reading from beginning
			content := make([]byte, 27)
			n, err := file.ReadAt(content, 0)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(Equal(27))
			Expect(string(content)).To(Equal("PREFIX:batch1,batch2,batch3"))
		})

		It("should track binary data correctly", func() {
			var err error
			file, err = fs.Open("binary.dat", os.O_CREATE|os.O_RDWR)
			Expect(err).ToNot(HaveOccurred())

			tracked, err := xio.NewTrackedWriteCloser(file)
			Expect(err).ToNot(HaveOccurred())

			// Write binary data
			binaryData := []byte{0x00, 0xFF, 0xAA, 0x55, 0x12, 0x34}
			n, err := tracked.Write(binaryData)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(Equal(6))
			Expect(tracked.Len()).To(Equal(int64(6)))

			// Verify the data was written correctly
			readData := make([]byte, 6)
			n, err = file.ReadAt(readData, 0)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(Equal(6))
			Expect(readData).To(Equal(binaryData))
		})
	})
})
