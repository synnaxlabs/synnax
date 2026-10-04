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
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Topic", func() {
	node := sparkplug.NodeID{Group: "plant", EdgeNode: "line1"}

	Describe("Namespace", func() {
		It("Should be the Sparkplug B namespace", func() {
			Expect(sparkplug.Namespace).To(Equal("spBv1.0"))
		})
	})

	Describe("MessageType", func() {
		DescribeTable("Should hold the topic level text",
			func(t sparkplug.MessageType, expected string) {
				Expect(string(t)).To(Equal(expected))
			},
			Entry("NBirth", sparkplug.NBirth, "NBIRTH"),
			Entry("NDeath", sparkplug.NDeath, "NDEATH"),
			Entry("DBirth", sparkplug.DBirth, "DBIRTH"),
			Entry("DDeath", sparkplug.DDeath, "DDEATH"),
			Entry("NData", sparkplug.NData, "NDATA"),
			Entry("DData", sparkplug.DData, "DDATA"),
			Entry("NCmd", sparkplug.NCmd, "NCMD"),
			Entry("DCmd", sparkplug.DCmd, "DCMD"),
			Entry("State", sparkplug.State, "STATE"),
		)
		DescribeTable("Session should be true for births, deaths, and data",
			func(t sparkplug.MessageType, expected bool) {
				Expect(t.Session()).To(Equal(expected))
			},
			Entry("NBirth", sparkplug.NBirth, true),
			Entry("NDeath", sparkplug.NDeath, true),
			Entry("DBirth", sparkplug.DBirth, true),
			Entry("DDeath", sparkplug.DDeath, true),
			Entry("NData", sparkplug.NData, true),
			Entry("DData", sparkplug.DData, true),
			Entry("NCmd", sparkplug.NCmd, false),
			Entry("DCmd", sparkplug.DCmd, false),
			Entry("State", sparkplug.State, false),
			Entry("no type", sparkplug.MessageType(""), false),
		)
	})

	Describe("ParseTopic", func() {
		DescribeTable("Should parse a topic and String should return its text",
			func(text string, expected sparkplug.Topic) {
				topic := MustSucceed(sparkplug.ParseTopic(text))
				Expect(topic).To(Equal(expected))
				Expect(topic.String()).To(Equal(text))
			},
			Entry("NBIRTH",
				"spBv1.0/plant/NBIRTH/line1",
				sparkplug.Topic{Type: sparkplug.NBirth, Node: node},
			),
			Entry("NDEATH",
				"spBv1.0/plant/NDEATH/line1",
				sparkplug.Topic{Type: sparkplug.NDeath, Node: node},
			),
			Entry("NDATA",
				"spBv1.0/plant/NDATA/line1",
				sparkplug.Topic{Type: sparkplug.NData, Node: node},
			),
			Entry("NCMD",
				"spBv1.0/plant/NCMD/line1",
				sparkplug.Topic{Type: sparkplug.NCmd, Node: node},
			),
			Entry("DBIRTH",
				"spBv1.0/plant/DBIRTH/line1/pump",
				sparkplug.Topic{Type: sparkplug.DBirth, Node: node, Device: "pump"},
			),
			Entry("DDEATH",
				"spBv1.0/plant/DDEATH/line1/pump",
				sparkplug.Topic{Type: sparkplug.DDeath, Node: node, Device: "pump"},
			),
			Entry("DDATA",
				"spBv1.0/plant/DDATA/line1/pump",
				sparkplug.Topic{Type: sparkplug.DData, Node: node, Device: "pump"},
			),
			Entry("DCMD",
				"spBv1.0/plant/DCMD/line1/pump",
				sparkplug.Topic{Type: sparkplug.DCmd, Node: node, Device: "pump"},
			),
			Entry("STATE",
				"spBv1.0/STATE/synnax",
				sparkplug.Topic{Type: sparkplug.State, HostID: "synnax"},
			),
		)

		DescribeTable("Should reject a malformed topic",
			func(text string) {
				Expect(sparkplug.ParseTopic(text)).Error().To(SatisfyAll(
					MatchError(sparkplug.ErrTopic),
					MatchError(ContainSubstring(text)),
					MatchError(ContainSubstring("not a Sparkplug B topic")),
				))
			},
			Entry("an empty topic", ""),
			Entry("the wrong namespace", "spAv1.0/plant/NBIRTH/line1"),
			Entry("no namespace", "plant/NBIRTH/line1"),
			Entry("the namespace only", "spBv1.0"),
			Entry("two levels", "spBv1.0/plant"),
			Entry("no edge node level", "spBv1.0/plant/NDATA"),
			Entry("STATE with no host ID", "spBv1.0/STATE"),
			Entry("STATE with a level after the host ID", "spBv1.0/STATE/synnax/x"),
			Entry("a device level on NBIRTH", "spBv1.0/plant/NBIRTH/line1/pump"),
			Entry("a device level on NDEATH", "spBv1.0/plant/NDEATH/line1/pump"),
			Entry("a device level on NDATA", "spBv1.0/plant/NDATA/line1/pump"),
			Entry("a device level on NCMD", "spBv1.0/plant/NCMD/line1/pump"),
			Entry("no device level on DBIRTH", "spBv1.0/plant/DBIRTH/line1"),
			Entry("no device level on DDEATH", "spBv1.0/plant/DDEATH/line1"),
			Entry("no device level on DDATA", "spBv1.0/plant/DDATA/line1"),
			Entry("no device level on DCMD", "spBv1.0/plant/DCMD/line1"),
			Entry("a level after the device", "spBv1.0/plant/DDATA/line1/pump/x"),
			Entry("an unknown type", "spBv1.0/plant/NFOO/line1"),
			Entry("a type in lower case", "spBv1.0/plant/ndata/line1"),
			Entry("STATE at the type level", "spBv1.0/plant/STATE/line1"),
		)
	})

	Describe("Topic.String", func() {
		It("Should leave the device level off a topic of the edge node", func() {
			topic := sparkplug.Topic{Type: sparkplug.NData, Node: node}
			Expect(topic.String()).To(Equal("spBv1.0/plant/NDATA/line1"))
		})
		It("Should end a device topic with the device ID", func() {
			topic := sparkplug.Topic{Type: sparkplug.DData, Node: node, Device: "pump"}
			Expect(topic.String()).To(Equal("spBv1.0/plant/DDATA/line1/pump"))
		})
		It("Should use only the host ID for a STATE topic", func() {
			topic := sparkplug.Topic{
				Type:   sparkplug.State,
				Node:   node,
				HostID: "synnax",
			}
			Expect(topic.String()).To(Equal("spBv1.0/STATE/synnax"))
		})
	})

	Describe("CommandTopic", func() {
		It("Should return an NCMD topic for a tag of the edge node", func() {
			topic := sparkplug.CommandTopic(node, "")
			Expect(topic).To(Equal(sparkplug.Topic{Type: sparkplug.NCmd, Node: node}))
			Expect(topic.String()).To(Equal("spBv1.0/plant/NCMD/line1"))
		})
		It("Should return a DCMD topic for a tag of a device", func() {
			topic := sparkplug.CommandTopic(node, "pump")
			Expect(topic).To(Equal(sparkplug.Topic{
				Type:   sparkplug.DCmd,
				Node:   node,
				Device: "pump",
			}))
			Expect(topic.String()).To(Equal("spBv1.0/plant/DCMD/line1/pump"))
		})
	})

	Describe("NodeID", func() {
		Describe("String", func() {
			It("Should join the group and the edge node with a slash", func() {
				Expect(node.String()).To(Equal("plant/line1"))
			})
		})

		Describe("Filters", func() {
			It("Should match the messages of the node and of its devices", func() {
				Expect(node.Filters()).To(Equal([]string{
					"spBv1.0/plant/+/line1", "spBv1.0/plant/+/line1/+",
				}))
			})
		})

		Describe("Validate", func() {
			It("Should accept two valid IDs", func() {
				Expect(node.Validate()).To(Succeed())
			})
			It("Should reject an empty group", func() {
				Expect(sparkplug.NodeID{EdgeNode: "line1"}.Validate()).To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring("group: required")),
				))
			})
			It("Should reject an empty edge node", func() {
				Expect(sparkplug.NodeID{Group: "plant"}.Validate()).To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring("edge_node: required")),
				))
			})
			It("Should reject a group with a wildcard", func() {
				id := sparkplug.NodeID{Group: "pl+nt", EdgeNode: "line1"}
				Expect(id.Validate()).To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(
						ContainSubstring("group: pl+nt must not hold /, +, or #"),
					),
				))
			})
			It("Should reject an edge node with a slash", func() {
				id := sparkplug.NodeID{Group: "plant", EdgeNode: "line/1"}
				Expect(id.Validate()).To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring(
						"edge_node: line/1 must not hold /, +, or #",
					)),
				))
			})
		})
	})

	Describe("ValidateID", func() {
		It("Should accept an ID with no reserved character", func() {
			Expect(sparkplug.ValidateID("device", "pump-1_A.b c")).To(Succeed())
		})
		It("Should reject an empty ID", func() {
			Expect(sparkplug.ValidateID("device", "")).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("device: required")),
			))
		})
		DescribeTable("Should reject a reserved character",
			func(id string) {
				Expect(sparkplug.ValidateID("device", id)).To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring(
						"device: "+id+" must not hold /, +, or #",
					)),
				))
			},
			Entry("a slash", "pump/1"),
			Entry("a plus", "pump+1"),
			Entry("a number sign", "pump#1"),
			Entry("a lone wildcard", "#"),
		)
	})
})
