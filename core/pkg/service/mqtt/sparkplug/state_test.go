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
	Describe("StateTopic", func() {
		It("Should return the STATE topic of the host application", func() {
			Expect(sparkplug.StateTopic("synnax")).To(Equal("spBv1.0/STATE/synnax"))
		})
		It("Should return a topic that ParseTopic reads back", func() {
			topic := MustSucceed(sparkplug.ParseTopic(sparkplug.StateTopic("synnax")))
			Expect(topic).To(Equal(sparkplug.Topic{
				Type:   sparkplug.State,
				HostID: "synnax",
			}))
		})
	})

	Describe("EncodeState", func() {
		DescribeTable(
			"Should return the exact JSON payload",
			func(online bool, timestamp telem.TimeStamp, expected string) {
				payload := sparkplug.EncodeState(online, timestamp)
				Expect(string(payload)).To(Equal(expected))
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
