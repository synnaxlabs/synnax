// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package stack_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/stack"
)

var _ = Describe("Stack", func() {
	Describe("Basic tests", func() {
		It("Should perform basic functions", func() {
			s := &stack.Stack[int]{}
			s.Push(1)
			s.Push(2)
			s.Push(3)
			Expect(s.Len()).To(Equal(3))
			Expect(s.Empty()).To(BeFalse())

			top := s.Peek()
			Expect(*top).To(Equal(3))
			Expect(s.Len()).To(Equal(3))
			*top = 4
			Expect(*s.Peek()).To(Equal(4))

			val, err := s.Pop()
			Expect(err).ToNot(HaveOccurred())
			Expect(val).To(Equal(4))
			for i := 2; i > 0; i-- {
				x, e := s.Pop()
				Expect(e).ToNot(HaveOccurred())
				Expect(x).To(Equal(i))
			}
			Expect(s.Empty()).To(BeTrue())
		})
		It("Should return an error when popping from an empty queue", func() {
			s := &stack.Stack[int]{}
			Expect(s.Peek()).To(BeNil())
			_, err := s.Pop()
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(stack.EmptyStackError))
			Expect(err.Error()).To(Equal("stack is empty"))
		})
		It("Should work with other types", func() {
			s := &stack.Stack[string]{}
			s.Push("a")
			s.Push("b")
			s.Push("c")
			Expect(s.Len()).To(Equal(3))
			Expect(s.Empty()).To(BeFalse())

			top := s.Peek()
			Expect(*top).To(Equal("c"))

			val, err := s.Pop()
			Expect(err).ToNot(HaveOccurred())
			Expect(val).To(Equal("c"))
			for _, v := range []string{"b", "a"} {
				x, e := s.Pop()
				Expect(e).ToNot(HaveOccurred())
				Expect(x).To(Equal(v))
			}
		})
	})
})
