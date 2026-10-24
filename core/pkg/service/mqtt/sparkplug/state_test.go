// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package sparkplug_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/mqtt/sparkplug"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("State", func() {
	Describe("EncodeState", func() {
		It(
			"Should publish retained at quality of service 1 on the STATE topic",
			func() {
				pub := sparkplug.EncodeState("synnax", true, 0)
				Expect(pub.Topic).To(Equal("spBv1.0/STATE/synnax"))
				Expect(pub.QoS).To(Equal(byte(1)))
				Expect(pub.Retained).To(BeTrue())
			},
		)
		It("Should use a topic that ParseTopic reads back", func() {
			pub := sparkplug.EncodeState("synnax", false, 0)
			Expect(MustSucceed(sparkplug.ParseTopic(pub.Topic))).To(Equal(
				sparkplug.Topic{Type: sparkplug.State, HostID: "synnax"},
			))
		})
		DescribeTable(
			"Should return the exact JSON payload",
			func(online bool, timestamp telem.TimeStamp, expected string) {
				pub := sparkplug.EncodeState("synnax", online, timestamp)
				Expect(string(pub.Payload)).To(Equal(expected))
			},
			Entry("online",
				true,
				1700000000123*telem.MillisecondTS,
				`{"online":true,"timestamp":1700000000123}`,
			),
			Entry("offline",
				false,
				1700000000123*telem.MillisecondTS,
				`{"online":false,"timestamp":1700000000123}`,
			),
			Entry("a timestamp between two milliseconds",
				true,
				1500*telem.MillisecondTS+999_999,
				`{"online":true,"timestamp":1500}`,
			),
			Entry(
				"the epoch",
				true,
				telem.TimeStamp(0),
				`{"online":true,"timestamp":0}`,
			),
		)
	})
})
