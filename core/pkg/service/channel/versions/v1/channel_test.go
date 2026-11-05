// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	v1 "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v1"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("Channel", func() {
	Describe("IsCalculated", func() {
		It("Should return true when the channel has an expression", func() {
			Expect(v1.Channel{Expression: "a + b"}.IsCalculated()).To(BeTrue())
		})
		It("Should return false when the channel has no expression", func() {
			Expect(v1.Channel{}.IsCalculated()).To(BeFalse())
		})
	})

	Describe("Key", func() {
		It("Should compose the key from the leaseholder and local key", func() {
			ch := v1.Channel{Leaseholder: 1, LocalKey: 2}
			Expect(ch.Key()).To(Equal(channel.NewKey(1, 2)))
		})
	})

	Describe("Index", func() {
		It("Should return the key of the channel's index channel", func() {
			ch := v1.Channel{Leaseholder: 1, LocalIndex: 5}
			Expect(ch.Index()).To(Equal(channel.NewKey(1, 5)))
		})
		It("Should return a zero key when the channel has no local index", func() {
			Expect(v1.Channel{Leaseholder: 1}.Index()).To(Equal(v1.Key(0)))
		})
	})

	Describe("GorpKey", func() {
		It("Should return the same value as Key", func() {
			ch := v1.Channel{Leaseholder: 1, LocalKey: 2}
			Expect(ch.GorpKey()).To(Equal(ch.Key()))
		})
	})

	Describe("SetOptions", func() {
		It("Should bootstrap-lease a free channel", func() {
			ch := v1.Channel{Leaseholder: node.KeyFree}
			Expect(ch.SetOptions()).To(Equal([]any{node.KeyBootstrapper}))
		})
		It("Should lease a non-free channel to its leaseholder", func() {
			ch := v1.Channel{Leaseholder: 2}
			Expect(ch.SetOptions()).To(Equal([]any{node.Key(2)}))
		})
	})

	Describe("Free", func() {
		It("Should return true for a channel with no leaseholder", func() {
			Expect(v1.Channel{Leaseholder: node.KeyFree}.Free()).To(BeTrue())
		})
		It("Should return false for a leased channel", func() {
			Expect(v1.Channel{Leaseholder: 1}.Free()).To(BeFalse())
		})
	})

	Describe("Distribution", func() {
		It(
			"Should return the distribution-layer representation of the channel",
			func() {
				ch := v1.Channel{
					Name:        "temp",
					Leaseholder: 1,
					DataType:    telem.Float32T,
					IsIndex:     true,
					LocalKey:    2,
					LocalIndex:  3,
					Virtual:     true,
					Concurrency: control.ConcurrencyShared,
					Internal:    true,
					Expression:  "a + b",
				}
				Expect(ch.Distribution()).To(Equal(channel.Channel{
					Name:        "temp",
					Leaseholder: 1,
					DataType:    telem.Float32T,
					IsIndex:     true,
					LocalKey:    2,
					LocalIndex:  3,
					Virtual:     true,
					Concurrency: control.ConcurrencyShared,
				}))
			},
		)
	})

	Describe("Equals", func() {
		It("Should return true if the two channels are equal", func() {
			c1 := v1.Channel{Leaseholder: 1, LocalKey: 1}
			c2 := v1.Channel{Leaseholder: 1, LocalKey: 1}
			Expect(c1.Equals(c2)).To(BeTrue())
		})
		DescribeTable(
			"Should ignore excluded fields",
			func(c1, c2 v1.Channel, exclude ...string) {
				Expect(c1.Equals(c2, exclude...)).To(BeTrue())
			},
			Entry(
				"Names",
				v1.Channel{Name: "name1", LocalKey: 1},
				v1.Channel{Name: "name2", LocalKey: 1},
				"Name",
			),
			Entry(
				"Leaseholders",
				v1.Channel{Leaseholder: 1, LocalKey: 1},
				v1.Channel{Leaseholder: 2, LocalKey: 1},
				"Leaseholder",
			),
			Entry(
				"LocalKeys",
				v1.Channel{Leaseholder: 1, LocalKey: 1},
				v1.Channel{Leaseholder: 1, LocalKey: 2},
				"LocalKey",
			),
			Entry(
				"Virtual",
				v1.Channel{Leaseholder: 1, LocalKey: 1},
				v1.Channel{Leaseholder: 1, LocalKey: 1, Virtual: true},
				"Virtual",
			),
			Entry(
				"DataType",
				v1.Channel{Leaseholder: 1, LocalKey: 1},
				v1.Channel{Leaseholder: 1, LocalKey: 1, DataType: "int"},
				"DataType",
			),
			Entry(
				"LocalIndex",
				v1.Channel{Leaseholder: 1, LocalKey: 1},
				v1.Channel{Leaseholder: 1, LocalKey: 1, LocalIndex: 1},
				"LocalIndex",
			),
			Entry(
				"Operations",
				v1.Channel{
					Leaseholder: 1,
					LocalKey:    1,
					Operations:  []v1.Operation{{Type: "max"}},
				},
				v1.Channel{
					Leaseholder: 1,
					LocalKey:    1,
					Operations:  []v1.Operation{{Type: "min"}},
				},
				"Operations",
			),
			Entry(
				"Retention",
				v1.Channel{Leaseholder: 1, LocalKey: 1},
				v1.Channel{Leaseholder: 1, LocalKey: 1, Retention: telem.Hour},
				"Retention",
			),
		)
		DescribeTable(
			"Should report inequality",
			func(c1, c2 v1.Channel, exclude ...string) {
				Expect(c1.Equals(c2, exclude...)).To(BeFalse())
			},
			Entry(
				"By LocalIndex",
				v1.Channel{Leaseholder: 1, LocalKey: 1, LocalIndex: 1},
				v1.Channel{Leaseholder: 1, LocalKey: 1, LocalIndex: 2},
			),
			Entry(
				"By Name",
				v1.Channel{Name: "name1", LocalKey: 1},
				v1.Channel{Name: "name2", LocalKey: 1},
			),
			Entry(
				"By Leaseholder",
				v1.Channel{Leaseholder: 1, LocalKey: 1},
				v1.Channel{Leaseholder: 2, LocalKey: 1},
			),
			Entry(
				"By LocalKey",
				v1.Channel{Leaseholder: 1, LocalKey: 1},
				v1.Channel{Leaseholder: 1, LocalKey: 2},
			),
			Entry(
				"By Data Type",
				v1.Channel{Leaseholder: 1, LocalKey: 1, DataType: "int"},
				v1.Channel{Leaseholder: 1, LocalKey: 1, DataType: "float"},
			),
			Entry(
				"By Virtual",
				v1.Channel{Leaseholder: 1, LocalKey: 1, Virtual: true},
				v1.Channel{Leaseholder: 1, LocalKey: 1},
			),
			Entry(
				"By Operations",
				v1.Channel{
					Leaseholder: 1,
					LocalKey:    1,
					Operations:  []v1.Operation{{Type: "max"}},
				},
				v1.Channel{
					Leaseholder: 1,
					LocalKey:    1,
					Operations:  []v1.Operation{{Type: "min"}},
				},
			),
			Entry(
				"By Retention",
				v1.Channel{Leaseholder: 1, LocalKey: 1, Retention: telem.Hour},
				v1.Channel{Leaseholder: 1, LocalKey: 1},
			),
		)
	})

	Describe("String", func() {
		DescribeTable("Should format the channel as a string",
			func(ch v1.Channel, expected string) {
				Expect(ch.String()).To(Equal(expected))
			},
			Entry("named channel",
				v1.Channel{Name: "temp", Leaseholder: 1, LocalKey: 2},
				"[temp]<1048578>",
			),
			Entry("unnamed channel",
				v1.Channel{Leaseholder: 1, LocalKey: 2},
				"<1048578>",
			),
			Entry("zero-value channel",
				v1.Channel{},
				"<0>",
			),
		)
	})

	Describe("OntologyID", func() {
		It("Should return the channel ontology identifier", func() {
			ch := v1.Channel{Leaseholder: 1, LocalKey: 2}
			Expect(ch.OntologyID()).To(Equal(ontology.ID{
				Type: ontology.ResourceTypeChannel, Key: ch.Key().String(),
			}))
		})
	})
})
