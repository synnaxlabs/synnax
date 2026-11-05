// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package confluence_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
)

type item struct{ id int }

// keepAll holds every value and reports each value that the queue delivers.
type keepAll struct{ popped chan item }

func (*keepAll) Push(held []item, v item) []item { return append(held, v) }

func (k *keepAll) Pop(v item) { k.popped <- v }

type keepNewest struct{}

func (keepNewest) Push(_ []item, v item) []item { return []item{v} }

func (keepNewest) Pop(item) {}

var _ = Describe("Queue", func() {
	var (
		input  *Stream[item]
		output *Stream[item]
		policy *keepAll
	)
	BeforeEach(func() {
		input = NewStream[item](0)
		output = NewStream[item](0)
		policy = &keepAll{popped: make(chan item, 1000)}
	})

	open := func(q *Queue[item]) {
		q.InFrom(input)
		q.OutTo(output)
		ctx, cancel := signal.Isolated()
		DeferCleanup(cancel)
		q.Flow(ctx, CloseOutputInletsOnExit())
	}

	send := func(items ...item) {
		for _, i := range items {
			Eventually(input.Inlet()).Should(BeSent(i))
		}
	}

	receive := func(n int) []item {
		received := make([]item, n)
		for i := range received {
			Eventually(output.Outlet()).Should(Receive(&received[i]))
		}
		return received
	}

	It("Should pass values through in order", func() {
		open(&Queue[item]{Policy: policy})
		send(item{0}, item{1}, item{2})
		Expect(receive(3)).To(Equal([]item{{0}, {1}, {2}}))
	})

	It("Should take every input while its output is not read", func() {
		open(&Queue[item]{Policy: policy})
		for i := range 100 {
			send(item{i})
		}
		Expect(receive(100)).To(HaveLen(100))
	})

	It("Should hold values by the injected policy", func() {
		open(&Queue[item]{Policy: keepNewest{}})
		send(item{0}, item{1}, item{2})
		Expect(receive(1)).To(Equal([]item{{2}}))
		Consistently(output.Outlet()).ShouldNot(Receive())
	})

	It("Should tell the policy about each value it delivers", func() {
		open(&Queue[item]{Policy: policy})
		send(item{0}, item{1})
		Expect(receive(2)).To(Equal([]item{{0}, {1}}))
		Eventually(policy.popped).Should(Receive(Equal(item{0})))
		Eventually(policy.popped).Should(Receive(Equal(item{1})))
	})

	It("Should deliver the values it holds after its input closes", func() {
		open(&Queue[item]{Policy: policy})
		send(item{0}, item{1}, item{2})
		input.Close()
		Expect(receive(3)).To(Equal([]item{{0}, {1}, {2}}))
		Eventually(output.Outlet()).Should(BeClosed())
	})
})
