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
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	xio "github.com/synnaxlabs/x/io"
	xfs "github.com/synnaxlabs/x/io/fs"
	. "github.com/synnaxlabs/x/testutil"
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
			file = MustSucceed(fs.Open("test.txt", os.O_CREATE|os.O_RDWR))

			tracked := MustSucceed(xio.NewTrackedWriteCloser(file))

			Expect(tracked.Offset()).To(Equal(int64(0)))
			Expect(tracked.Len()).To(Equal(int64(0)))
		})

		It("should create a tracked writer at the end of an existing file", func() {
			file = MustSucceed(fs.Open("test.txt", os.O_CREATE|os.O_RDWR))

			// Write some initial data
			MustSucceed(file.Write([]byte("Initial content")))

			tracked := MustSucceed(xio.NewTrackedWriteCloser(file))

			Expect(tracked.Offset()).To(Equal(int64(15))) // "Initial content" is 15 bytes
			Expect(tracked.Len()).To(Equal(int64(0)))
		})
	})

	Describe("Write", func() {
		It("should track bytes written", func() {
			file = MustSucceed(fs.Open("test.txt", os.O_CREATE|os.O_RDWR))

			tracked := MustSucceed(xio.NewTrackedWriteCloser(file))

			// First write
			n := MustSucceed(tracked.Write([]byte("Hello")))
			Expect(n).To(Equal(5))
			Expect(tracked.Len()).To(Equal(int64(5)))
			Expect(tracked.Offset()).To(Equal(int64(0)))

			// Second write
			n = MustSucceed(tracked.Write([]byte(" World")))
			Expect(n).To(Equal(6))
			Expect(tracked.Len()).To(Equal(int64(11)))
			Expect(tracked.Offset()).To(Equal(int64(0)))
		})

		It("should handle multiple sequential writes", func() {
			file = MustSucceed(fs.Open("test.txt", os.O_CREATE|os.O_RDWR))

			tracked := MustSucceed(xio.NewTrackedWriteCloser(file))

			writes := []string{"First ", "Second ", "Third"}
			totalLen := int64(0)

			for _, data := range writes {
				n := MustSucceed(tracked.Write([]byte(data)))
				Expect(n).To(Equal(len(data)))
				totalLen += int64(n)
				Expect(tracked.Len()).To(Equal(totalLen))
			}

			Expect(tracked.Offset()).To(Equal(int64(0)))
		})

		It("should handle empty writes", func() {
			file = MustSucceed(fs.Open("test.txt", os.O_CREATE|os.O_RDWR))

			tracked := MustSucceed(xio.NewTrackedWriteCloser(file))

			n := MustSucceed(tracked.Write([]byte{}))
			Expect(n).To(Equal(0))
			Expect(tracked.Len()).To(Equal(int64(0)))
		})
	})

	Describe("Reset", func() {
		It("should reset the tracked length and update offset", func() {
			file = MustSucceed(fs.Open("test.txt", os.O_CREATE|os.O_RDWR))

			tracked := MustSucceed(xio.NewTrackedWriteCloser(file))

			// Write some data
			MustSucceed(tracked.Write([]byte("First batch")))
			Expect(tracked.Len()).To(Equal(int64(11)))
			Expect(tracked.Offset()).To(Equal(int64(0)))

			// Reset
			tracked.Reset()
			Expect(tracked.Len()).To(Equal(int64(0)))
			Expect(tracked.Offset()).To(Equal(int64(11)))

			// Write more data after reset
			MustSucceed(tracked.Write([]byte("Second")))
			Expect(tracked.Len()).To(Equal(int64(6)))
			Expect(tracked.Offset()).To(Equal(int64(11)))
		})

		It("should handle multiple resets", func() {
			file = MustSucceed(fs.Open("test.txt", os.O_CREATE|os.O_RDWR))

			tracked := MustSucceed(xio.NewTrackedWriteCloser(file))

			// First write and reset
			MustSucceed(tracked.Write([]byte("AAA")))
			tracked.Reset()
			Expect(tracked.Offset()).To(Equal(int64(3)))

			// Second write and reset
			MustSucceed(tracked.Write([]byte("BBBB")))
			tracked.Reset()
			Expect(tracked.Offset()).To(Equal(int64(7)))

			// Third write and reset
			MustSucceed(tracked.Write([]byte("CC")))
			tracked.Reset()
			Expect(tracked.Offset()).To(Equal(int64(9)))

			Expect(tracked.Len()).To(Equal(int64(0)))
		})

		It("should reset without any writes", func() {
			file = MustSucceed(fs.Open("test.txt", os.O_CREATE|os.O_RDWR))

			tracked := MustSucceed(xio.NewTrackedWriteCloser(file))

			// Reset without writing
			tracked.Reset()
			Expect(tracked.Len()).To(Equal(int64(0)))
			Expect(tracked.Offset()).To(Equal(int64(0)))
		})
	})

	Describe("Close", func() {
		It("should close the underlying file", func() {
			file = MustSucceed(fs.Open("test.txt", os.O_CREATE|os.O_RDWR))

			tracked := MustSucceed(xio.NewTrackedWriteCloser(file))

			MustSucceed(tracked.Write([]byte("test")))

			Expect(tracked.Close()).To(Succeed())

			// Clear the file variable to prevent AfterEach from trying to close it again
			file = nil
		})
	})

	Describe("Integration tests", func() {
		It("should handle a complete write-reset-write workflow", func() {
			file = MustSucceed(fs.Open("test.txt", os.O_CREATE|os.O_RDWR))

			// Write initial content directly to file
			MustSucceed(file.Write([]byte("PREFIX:")))

			tracked := MustSucceed(xio.NewTrackedWriteCloser(file))
			Expect(tracked.Offset()).To(Equal(int64(7)))

			// First batch of writes
			MustSucceed(tracked.Write([]byte("batch1,")))
			Expect(tracked.Len()).To(Equal(int64(7)))

			// Reset and second batch
			tracked.Reset()
			MustSucceed(tracked.Write([]byte("batch2,")))
			Expect(tracked.Len()).To(Equal(int64(7)))
			Expect(tracked.Offset()).To(Equal(int64(14)))

			// Reset and third batch
			tracked.Reset()
			MustSucceed(tracked.Write([]byte("batch3")))
			Expect(tracked.Len()).To(Equal(int64(6)))
			Expect(tracked.Offset()).To(Equal(int64(21)))

			// Verify file contents by reading from beginning
			content := make([]byte, 27)
			n := MustSucceed(file.ReadAt(content, 0))
			Expect(n).To(Equal(27))
			Expect(string(content)).To(Equal("PREFIX:batch1,batch2,batch3"))
		})

		It("should track binary data correctly", func() {
			file = MustSucceed(fs.Open("binary.dat", os.O_CREATE|os.O_RDWR))

			tracked := MustSucceed(xio.NewTrackedWriteCloser(file))

			// Write binary data
			binaryData := []byte{0x00, 0xFF, 0xAA, 0x55, 0x12, 0x34}
			n := MustSucceed(tracked.Write(binaryData))
			Expect(n).To(Equal(6))
			Expect(tracked.Len()).To(Equal(int64(6)))

			// Verify the data was written correctly
			readData := make([]byte, 6)
			n = MustSucceed(file.ReadAt(readData, 0))
			Expect(n).To(Equal(6))
			Expect(readData).To(Equal(binaryData))
		})
	})
})
