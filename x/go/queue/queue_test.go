// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package queue_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/queue"
)

var _ = Describe("Queue", func() {
	Describe("Basic tests", func() {
		It("Should perform basic functions", func() {
			q := &queue.Queue[int]{}
			q.Push(1)
			q.Push(2)
			q.Push(3)
			Expect(q.Len()).To(Equal(3))
			Expect(q.Empty()).To(BeFalse())

			front := q.Peek()
			Expect(front).To(Equal(1))
			Expect(q.Len()).To(Equal(3))

			val, err := q.Pop()
			Expect(err).ToNot(HaveOccurred())
			Expect(val).To(Equal(1))
			for i := 0; i < 2; i++ {
				x, e := q.Pop()
				Expect(e).ToNot(HaveOccurred())
				Expect(x).To(Equal(i + 2))
			}
			Expect(q.Empty()).To(BeTrue())
		})
		It("Should return an error when popping from an empty queue", func() {
			q := &queue.Queue[int]{}
			Expect(func() { q.Peek() }).To(Panic())
			_, err := q.Pop()
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(queue.EmptyQueueError))
			Expect(err.Error()).To(Equal("queue is empty"))
		})
		It("Should work with other types", func() {
			q := &queue.Queue[string]{}
			q.Push("a")
			q.Push("b")
			q.Push("c")
			Expect(q.Len()).To(Equal(3))
			Expect(q.Empty()).To(BeFalse())

			front := q.Peek()
			Expect(front).To(Equal("a"))

			val, err := q.Pop()
			Expect(err).ToNot(HaveOccurred())
			Expect(val).To(Equal("a"))
			for _, v := range []string{"b", "c"} {
				x, e := q.Pop()
				Expect(e).ToNot(HaveOccurred())
				Expect(x).To(Equal(v))
			}
		})
	})
})
