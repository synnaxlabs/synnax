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
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Message", func() {
	node := sparkplug.NodeID{Group: "plant", EdgeNode: "line1"}

	Describe("Decode", func() {
		It("Should keep the topic and decode the payload", func() {
			topic := sparkplug.Topic{Type: sparkplug.NBirth, Node: node}
			m := MustSucceed(sparkplug.Decode(topic, encode(birth(0, 3,
				declared("temp", 1, sparkplug.Double, doubleValue(21.5)),
			))))
			Expect(m.Topic).To(Equal(topic))
			ev := sparkplug.NewHost(sparkplug.DefaultRebirthInterval).Handle(m)
			Expect(ev.Metrics).To(HaveLen(2))
			Expect(ev.Metrics[1].Name).To(Equal("temp"))
			Expect(ev.Metrics[1].Value).To(Equal(sparkplug.FloatValue(21.5)))
		})

		DescribeTable("Should return an error that names the topic",
			func(topic sparkplug.Topic, text string) {
				Expect(sparkplug.Decode(topic, []byte{0xFF, 0xFF, 0xFF})).Error().
					To(MatchError(ContainSubstring("invalid payload on " + text)))
			},
			Entry("NBIRTH",
				sparkplug.Topic{Type: sparkplug.NBirth, Node: node},
				"spBv1.0/plant/NBIRTH/line1",
			),
			Entry("NDATA",
				sparkplug.Topic{Type: sparkplug.NData, Node: node},
				"spBv1.0/plant/NDATA/line1",
			),
			Entry("NDEATH",
				sparkplug.Topic{Type: sparkplug.NDeath, Node: node},
				"spBv1.0/plant/NDEATH/line1",
			),
			Entry("NCMD",
				sparkplug.Topic{Type: sparkplug.NCmd, Node: node},
				"spBv1.0/plant/NCMD/line1",
			),
			Entry("DBIRTH",
				sparkplug.Topic{Type: sparkplug.DBirth, Node: node, Device: "pump"},
				"spBv1.0/plant/DBIRTH/line1/pump",
			),
			Entry("DDATA",
				sparkplug.Topic{Type: sparkplug.DData, Node: node, Device: "pump"},
				"spBv1.0/plant/DDATA/line1/pump",
			),
			Entry("DDEATH",
				sparkplug.Topic{Type: sparkplug.DDeath, Node: node, Device: "pump"},
				"spBv1.0/plant/DDEATH/line1/pump",
			),
		)
	})
})
