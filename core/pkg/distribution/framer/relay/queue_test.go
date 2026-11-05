// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package relay_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/relay"
	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("DropOldest", func() {
	var d *relay.DropOldest
	BeforeEach(func() { d = &relay.DropOldest{Capacity: 8} })
	res := func(key channel.Key, value int32) relay.Response {
		return relay.Response{Frame: frame.NewUnary(key, telem.NewSeriesV(value))}
	}
	push := func(responses ...relay.Response) []relay.Response {
		var held []relay.Response
		for _, r := range responses {
			held = d.Push(held, r)
		}
		return held
	}

	It("Should hold responses up to its capacity", func() {
		Expect(push(res(1, 1), res(2, 2))).To(Equal([]relay.Response{
			res(1, 1),
			res(2, 2),
		}))
	})

	It("Should replace dropped responses with a notice of their channels", func() {
		Expect(push(res(2, 1), res(1, 2), res(2, 3))).To(Equal(
			[]relay.Response{{Dropped: channel.Keys{2}}, res(1, 2), res(2, 3)},
		))
	})

	It("Should merge adjacent notices", func() {
		Expect(push(res(2, 1), res(1, 2), res(2, 3), res(3, 4))).To(Equal(
			[]relay.Response{
				{Dropped: channel.Keys{1, 2}},
				res(2, 3),
				res(3, 4),
			},
		))
	})

	It("Should keep responses with no frame in front of the notice", func() {
		ack := relay.Response{Ack: true}
		Expect(push(res(1, 1), ack, res(1, 2), res(1, 3))).To(Equal(
			[]relay.Response{ack, {Dropped: channel.Keys{1}}, res(1, 2), res(1, 3)},
		))
	})

	It("Should hold one notice for a consumer that never reads", func() {
		d.Capacity = 4
		var held []relay.Response
		for i := range 100 {
			held = d.Push(held, res(channel.Key(i%3+1), int32(i)))
		}
		Expect(held).To(Equal([]relay.Response{
			{Dropped: channel.Keys{1, 2, 3}},
			res(1, 99),
		}))
	})

	It("Should never drop the newest response", func() {
		big := relay.Response{
			Frame: frame.NewUnary(1, telem.NewSeriesV[int64](1, 2, 3)),
		}
		Expect(push(res(2, 1), big)).To(Equal([]relay.Response{
			{Dropped: channel.Keys{2}},
			big,
		}))
	})

	It("Should free the size of popped responses", func() {
		d.Pop(push(res(1, 1), res(1, 2))[0])
		Expect(push(res(1, 3))).To(Equal([]relay.Response{res(1, 3)}))
	})
})
