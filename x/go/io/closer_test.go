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
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/io"
)

// mockCloser is a test helper that tracks Close() calls
type mockCloser struct {
	closed bool
	err    error
}

func (m *mockCloser) Close() error {
	m.closed = true
	return m.err
}

var _ = Describe("Closer", func() {
	Describe("CloserFunc", func() {
		It("should execute the function when Close is called", func() {
			var called bool
			closer := io.CloserFunc(func() error { called = true; return nil })
			Expect(closer.Close()).To(Succeed())
			Expect(called).To(BeTrue())
		})

		It("should return the error from the function", func() {
			expectedErr := errors.New("close failed")
			closer := io.CloserFunc(func() error { return expectedErr })
			Expect(closer.Close()).To(MatchError(expectedErr))
		})

		It("should allow multiple calls to Close", func() {
			var callCount int
			closer := io.CloserFunc(func() error { callCount++; return nil })
			Expect(closer.Close()).To(Succeed())
			Expect(closer.Close()).To(Succeed())
			Expect(callCount).To(Equal(2))
		})
	})

	Describe("NoFailCloserFunc", func() {
		It("should execute the function and return nil", func() {
			var called bool
			closer := io.NoFailCloserFunc(func() { called = true })
			Expect(closer.Close()).To(Succeed())
			Expect(called).To(BeTrue())
		})
		It("should allow multiple calls to Close", func() {
			var callCount int
			closer := io.NoFailCloserFunc(func() { callCount++ })
			Expect(closer.Close()).To(Succeed())
			Expect(closer.Close()).To(Succeed())
			Expect(callCount).To(Equal(2))
		})

		It("should panic when Close is called on a nil NoFailCloserFunc", func() {
			var closer io.NoFailCloserFunc
			Expect(closer).To(BeNil())
			Expect(func() { closer.Close() }).To(Panic())
		})
	})

	Describe("MultiCloser", func() {
		It("should close all closers in reverse order", func() {
			var closeOrder []int

			closer1 := io.CloserFunc(func() error {
				closeOrder = append(closeOrder, 1)
				return nil
			})
			closer2 := io.CloserFunc(func() error {
				closeOrder = append(closeOrder, 2)
				return nil
			})
			closer3 := io.CloserFunc(func() error {
				closeOrder = append(closeOrder, 3)
				return nil
			})

			multi := io.MultiCloser{closer1, closer2, closer3}

			Expect(multi.Close()).To(Succeed())
			Expect(closeOrder).To(Equal([]int{3, 2, 1}))
		})

		It("should aggregate errors from all closers", func() {
			err1 := errors.New("error 1")
			err2 := errors.New("error 2")

			closer1 := io.CloserFunc(func() error { return err1 })
			closer2 := io.CloserFunc(func() error { return nil })
			closer3 := io.CloserFunc(func() error { return err2 })

			multi := io.MultiCloser{closer1, closer2, closer3}

			err := multi.Close()
			Expect(err).To(HaveOccurred())
			// The implementation uses errors.NewCatcher with aggregation which returns
			// only the first error from Error() method but collects all errors
			// internally
			Expect(err).To(Equal(err2)) // Since closers are closed in reverse order, err2 is encountered first
		})

		It("should handle empty MultiCloser", func() {
			multi := io.MultiCloser{}
			Expect(multi.Close()).To(Succeed())
		})

		It("should handle single closer", func() {
			called := false
			closer := io.CloserFunc(func() error { called = true; return nil })
			multi := io.MultiCloser{closer}
			Expect(multi.Close()).To(Succeed())
			Expect(called).To(BeTrue())
		})
		It("should close all closers even if some fail", func() {
			var closeOrder []int
			err := errors.New("closer 2 failed")

			closer1 := io.CloserFunc(func() error {
				closeOrder = append(closeOrder, 1)
				return nil
			})
			closer2 := io.CloserFunc(func() error {
				closeOrder = append(closeOrder, 2)
				return err
			})
			closer3 := io.CloserFunc(func() error {
				closeOrder = append(closeOrder, 3)
				return nil
			})

			multi := io.MultiCloser{closer1, closer2, closer3}

			Expect(multi.Close()).To(MatchError(err))
			// All closers should have been called despite the error
			Expect(closeOrder).To(Equal([]int{3, 2, 1}))
		})

		It("should work with mixed closer types", func() {
			// Using different types of closers
			mockCloser := &mockCloser{}
			funcCloser := io.CloserFunc(func() error { return nil })
			nopCloser := io.NoFailCloserFunc(func() {})

			multi := io.MultiCloser{mockCloser, funcCloser, nopCloser}

			Expect(multi.Close()).To(Succeed())
			Expect(mockCloser.closed).To(BeTrue())
		})

		It("should panic on nil closer", func() {
			closer1 := io.CloserFunc(func() error { return nil })
			multi := io.MultiCloser{closer1, nil}
			Expect(func() { multi.Close() }).To(Panic())
		})

		Describe("Integration tests", func() {
			It("should work with file operations", func() {
				var closedFiles []string

				// Create mock closers that track which files were closed
				file1Closer := io.CloserFunc(func() error {
					closedFiles = append(closedFiles, "file1")
					return nil
				})
				file2Closer := io.CloserFunc(func() error {
					closedFiles = append(closedFiles, "file2")
					return nil
				})
				file3Closer := io.CloserFunc(func() error {
					closedFiles = append(closedFiles, "file3")
					return nil
				})

				// Create multi closer
				multi := io.MultiCloser{file1Closer, file2Closer, file3Closer}

				// Close all files at once
				Expect(multi.Close()).To(Succeed())

				// Verify files were closed in reverse order
				Expect(closedFiles).To(Equal([]string{"file3", "file2", "file1"}))
			})

			It("should work in nested MultiCloser scenarios", func() {
				var closeOrder []string

				closer1 := io.CloserFunc(func() error {
					closeOrder = append(closeOrder, "1")
					return nil
				})
				closer2 := io.CloserFunc(func() error {
					closeOrder = append(closeOrder, "2")
					return nil
				})
				closer3 := io.CloserFunc(func() error {
					closeOrder = append(closeOrder, "3")
					return nil
				})
				closer4 := io.CloserFunc(func() error {
					closeOrder = append(closeOrder, "4")
					return nil
				})

				// Create nested multi closers
				inner := io.MultiCloser{closer1, closer2}
				outer := io.MultiCloser{inner, closer3, closer4}

				Expect(outer.Close()).To(Succeed())

				// Should close in reverse order: 4, 3, then inner (which closes 2, 1)
				Expect(closeOrder).To(Equal([]string{"4", "3", "2", "1"}))
			})
		})
	})

	Describe("NopCloser", func() {
		It("should do nothing when Close is called", func() {
			Expect(io.NopCloser.Close()).To(Succeed())
		})
	})
})
