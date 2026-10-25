// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package mqtt_test

import (
	"context"
	"math"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/mqtt"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// tagEntry returns the config of one Sparkplug B read entry.
func tagEntry(group, edgeNode, dev, tag string, ch channel.Channel) map[string]any {
	return map[string]any{
		"key":       group + "/" + edgeNode + "/" + dev + "/" + tag,
		"type":      "sparkplug",
		"group":     group,
		"edge_node": edgeNode,
		"device":    dev,
		"tag":       tag,
		"channel":   ch.Key(),
	}
}

const (
	testRebirthInterval = 50 * time.Millisecond
	testBirthGrace      = 300 * time.Millisecond
)

var _ = Describe("Sparkplug B read entries", func() {
	var (
		broker  *testBroker
		rackKey rack.Key
		dev     device.Device
		factory driver.Factory
		node    *testEdgeNode
	)

	BeforeEach(func(ctx SpecContext) {
		broker = startBroker(0)
		rackKey = createRack(ctx)
		dev = createBrokerDevice(ctx, rackKey, broker.port, nil)
		factory = MustOpen(mqtt.OpenFactory(mqtt.FactoryConfig{
			Device:  deviceSvc,
			Channel: channelSvc,
			Framer:  framerSvc,
			DB:      db, Status: statusSvc,
			RebirthInterval: testRebirthInterval,
			BirthGrace:      testBirthGrace,
		}))
		node = startEdgeNode(broker, "plant", "line1")
	})

	// start configures and starts a read task, and waits for the birth that answers
	// its rebirth request.
	start := func(ctx context.Context, cfg msgpack.EncodedJSON) task.Task {
		GinkgoHelper()
		t := newTask(rackKey, mqtt.ReadTaskType, cfg)
		configured := configure(ctx, factory, t)
		before := node.births()
		Expect(configured.Exec(ctx, task.Command{Type: "start", Key: "start"})).
			To(Succeed())
		Eventually(node.births).Should(BeNumerically(">", before))
		return t
	}

	Describe("Values", func() {
		It("Should ask for a rebirth and write the values of the birth",
			func(ctx SpecContext) {
				idx, data := createIndexed(ctx, telem.Float64T)
				node.setTags("", tagValue{
					name:      "temperature",
					alias:     1,
					value:     23.5,
					timestamp: 1700000000000,
				})
				t := start(ctx, readConfig(
					dev, tagEntry("plant", "line1", "", "temperature", data[0]),
				))
				Eventually(func() int64 { return stored(ctx, data[0]).Len() }).
					Should(BeEquivalentTo(1))
				Expect(stored(ctx, data[0])).
					To(telem.MatchSeriesData(telem.NewSeriesV(23.5)))
				Expect(stored(ctx, idx).ValueAt[telem.TimeStamp](0)).
					To(Equal(1700000000000 * telem.MillisecondTS))
				Expect(taskStatus(ctx, t).Variant).To(Equal(status.VariantSuccess))
			},
		)

		It("Should resolve the alias of a data message", func(ctx SpecContext) {
			_, data := createIndexed(ctx, telem.Float64T)
			node.setTags("", tagValue{
				name: "temperature", alias: 7, value: 1.0, timestamp: 1700000000000,
			})
			start(ctx, readConfig(
				dev, tagEntry("plant", "line1", "", "temperature", data[0]),
			))
			node.data("", tagValue{alias: 7, value: 2.0, timestamp: 1700000001000})
			node.data("", tagValue{alias: 7, value: 3.0, timestamp: 1700000002000})
			Eventually(func() int64 { return stored(ctx, data[0]).Len() }).
				Should(BeEquivalentTo(3))
			Expect(stored(ctx, data[0])).
				To(telem.MatchSeriesData(telem.NewSeriesV(1.0, 2.0, 3.0)))
		})

		It("Should give each tag of one message its own index", func(ctx SpecContext) {
			_, first := createIndexed(ctx, telem.Float64T)
			_, second := createIndexed(ctx, telem.Int32T)
			node.setTags(
				"",
				tagValue{name: "a", alias: 1, value: 1.0, timestamp: 1700000000000},
				tagValue{
					name:      "b",
					alias:     2,
					value:     int32(-4),
					timestamp: 1700000000500,
				},
			)
			start(ctx, readConfig(
				dev,
				tagEntry("plant", "line1", "", "a", first[0]),
				tagEntry("plant", "line1", "", "b", second[0]),
			))
			Eventually(func() int64 { return stored(ctx, second[0]).Len() }).
				Should(BeEquivalentTo(1))
			Expect(stored(ctx, first[0])).
				To(telem.MatchSeriesData(telem.NewSeriesV(1.0)))
			Expect(stored(ctx, second[0])).
				To(telem.MatchSeriesData(telem.NewSeriesV[int32](-4)))
		})

		It("Should read the tags of a device", func(ctx SpecContext) {
			_, data := createIndexed(ctx, telem.Uint8T)
			node.setTags("pump", tagValue{
				name: "running", alias: 3, value: false, timestamp: 1700000000000,
			})
			start(ctx, readConfig(
				dev, tagEntry("plant", "line1", "pump", "running", data[0]),
			))
			node.data("pump", tagValue{alias: 3, value: true, timestamp: 1700000001000})
			Eventually(func() int64 { return stored(ctx, data[0]).Len() }).
				Should(BeEquivalentTo(2))
			Expect(stored(ctx, data[0])).
				To(telem.MatchSeriesData(telem.NewSeriesV[uint8](0, 1)))
		})

		It("Should write a string tag to a virtual channel", func(ctx SpecContext) {
			ch := createVirtual(ctx, telem.StringT)
			node.setTags("", tagValue{name: "mode", alias: 1, value: "auto"})
			t := start(ctx, readConfig(
				dev, tagEntry("plant", "line1", "", "mode", ch),
			))
			Consistently(func() status.Variant {
				return taskStatus(ctx, t).Variant
			}, 200*time.Millisecond).Should(Equal(status.VariantSuccess))
		})

		It("Should skip a birth value that the task already wrote",
			func(ctx SpecContext) {
				_, data := createIndexed(ctx, telem.Float64T)
				node.setTags("", tagValue{
					name: "temperature", alias: 1, value: 5.0, timestamp: 1700000000000,
				})
				t := start(ctx, readConfig(
					dev, tagEntry("plant", "line1", "", "temperature", data[0]),
				))
				Eventually(func() int64 { return stored(ctx, data[0]).Len() }).
					Should(BeEquivalentTo(1))
				before := node.births()
				node.birth()
				Expect(node.births()).To(Equal(before + 1))
				Consistently(func() status.Variant {
					return taskStatus(ctx, t).Variant
				}, 200*time.Millisecond).Should(Equal(status.VariantSuccess))
				Expect(stored(ctx, data[0]).Len()).To(BeEquivalentTo(1))
			},
		)

		It("Should warn about a historical value and not write it",
			func(ctx SpecContext) {
				_, data := createIndexed(ctx, telem.Float64T)
				node.setTags("", tagValue{
					name: "temperature", alias: 1, value: 5.0, timestamp: 1700000002000,
				})
				t := start(ctx, readConfig(
					dev, tagEntry("plant", "line1", "", "temperature", data[0]),
				))
				Eventually(func() int64 { return stored(ctx, data[0]).Len() }).
					Should(BeEquivalentTo(1))
				node.data("", tagValue{
					alias: 1, value: 4.0, timestamp: 1700000001000, historical: true,
				})
				Eventually(func() task.Status { return taskStatus(ctx, t) }).Should(And(
					HaveField("Variant", status.VariantWarning),
					HaveField("Message", Equal("Failed to read some data")),
					HaveField("Description", Equal(
						"rejected a historical value of tag plant/line1/temperature: "+
							"the task does not write values that an edge node stored "+
							"while offline",
					)),
				))
				Expect(stored(ctx, data[0])).
					To(telem.MatchSeriesData(telem.NewSeriesV(5.0)))
			},
		)
	})

	Describe("Session state", func() {
		It(
			"Should ask for a rebirth after a gap in the sequence",
			func(ctx SpecContext) {
				_, data := createIndexed(ctx, telem.Float64T)
				node.setTags("", tagValue{
					name: "temperature", alias: 1, value: 1.0, timestamp: 1700000000000,
				})
				start(ctx, readConfig(
					dev, tagEntry("plant", "line1", "", "temperature", data[0]),
				))
				before := node.births()
				node.skipSeq()
				node.data("", tagValue{alias: 1, value: 2.0, timestamp: 1700000001000})
				Eventually(node.births).Should(Equal(before + 1))
				// The birth that follows carries the value of the dropped message.
				Eventually(func() int64 { return stored(ctx, data[0]).Len() }).
					Should(BeEquivalentTo(2))
				Expect(stored(ctx, data[0])).
					To(telem.MatchSeriesData(telem.NewSeriesV(1.0, 2.0)))
			},
		)

		DescribeTable("Should warn about an edge node that gives no birth after a "+
			"sequence gap",
			func(ctx SpecContext, device string) {
				_, data := createIndexed(ctx, telem.Float64T)
				node.setTags(device, tagValue{
					name: "temperature", alias: 1, value: 1.0, timestamp: 1700000000000,
				})
				t := start(ctx, readConfig(
					dev, tagEntry("plant", "line1", device, "temperature", data[0]),
				))
				Eventually(func() int64 { return stored(ctx, data[0]).Len() }).
					Should(BeEquivalentTo(1))
				node.setSilent(true)
				node.skipSeq()
				node.data(device, tagValue{
					alias: 1, value: 2.0, timestamp: 1700000001000,
				})
				Eventually(func() task.Status { return taskStatus(ctx, t) }).Should(And(
					HaveField("Variant", status.VariantWarning),
					HaveField("Message", Equal("Failed to read some data")),
					HaveField("Description", Equal("edge node plant/line1 is offline")),
				))
			},
			Entry("in a message of the edge node", ""),
			Entry("in a message of a device", "pump"),
		)

		It("Should warn while an edge node is dead", func(ctx SpecContext) {
			_, data := createIndexed(ctx, telem.Float64T)
			node.setTags("", tagValue{
				name: "temperature", alias: 1, value: 1.0, timestamp: 1700000000000,
			})
			t := start(ctx, readConfig(
				dev, tagEntry("plant", "line1", "", "temperature", data[0]),
			))
			node.death()
			Eventually(func() task.Status { return taskStatus(ctx, t) }).Should(And(
				HaveField("Variant", status.VariantWarning),
				HaveField("Message", Equal("Failed to read some data")),
				HaveField(
					"Description",
					ContainSubstring("edge node plant/line1 is offline"),
				),
			))
			node.birth()
			Eventually(func() status.Variant { return taskStatus(ctx, t).Variant }).
				Should(Equal(status.VariantSuccess))
		})

		It("Should warn while a device is dead", func(ctx SpecContext) {
			_, data := createIndexed(ctx, telem.Float64T)
			node.setTags("pump", tagValue{
				name: "speed", alias: 1, value: 1.0, timestamp: 1700000000000,
			})
			t := start(ctx, readConfig(
				dev, tagEntry("plant", "line1", "pump", "speed", data[0]),
			))
			node.deviceDeath("pump")
			Eventually(func() task.Status { return taskStatus(ctx, t) }).Should(And(
				HaveField("Variant", status.VariantWarning),
				HaveField("Message", Equal("Failed to read some data")),
				HaveField(
					"Description",
					ContainSubstring("device plant/line1/pump is offline"),
				),
			))
			node.birth()
			Eventually(func() status.Variant { return taskStatus(ctx, t).Variant }).
				Should(Equal(status.VariantSuccess))
		})

		It("Should warn about an edge node that gives no birth", func(ctx SpecContext) {
			_, data := createIndexed(ctx, telem.Float64T)
			node.setSilent(true)
			t := newTask(rackKey, mqtt.ReadTaskType, readConfig(
				dev, tagEntry("plant", "line1", "", "temperature", data[0]),
			))
			configured := configure(ctx, factory, t)
			Expect(configured.Exec(ctx, task.Command{Type: "start", Key: "start"})).
				To(Succeed())
			Eventually(func() task.Status { return taskStatus(ctx, t) }).Should(And(
				HaveField("Variant", status.VariantWarning),
				HaveField("Message", Equal("Failed to read some data")),
				HaveField(
					"Description",
					ContainSubstring("edge node plant/line1 is offline"),
				),
			))
		})

		It("Should warn about a device that gives no birth", func(ctx SpecContext) {
			_, data := createIndexed(ctx, telem.Float64T)
			t := start(ctx, readConfig(
				dev, tagEntry("plant", "line1", "pump", "speed", data[0]),
			))
			Eventually(func() task.Status { return taskStatus(ctx, t) }).Should(And(
				HaveField("Variant", status.VariantWarning),
				HaveField("Message", Equal("Failed to read some data")),
				HaveField("Description", Equal("device plant/line1/pump is offline")),
			))
		})

		It("Should warn about a device that gives no birth after its edge node is "+
			"born again",
			func(ctx SpecContext) {
				_, data := createIndexed(ctx, telem.Float64T)
				node.setTags("pump", tagValue{
					name: "speed", alias: 1, value: 1.0, timestamp: 1700000000000,
				})
				t := start(ctx, readConfig(
					dev, tagEntry("plant", "line1", "pump", "speed", data[0]),
				))
				Eventually(func() int64 { return stored(ctx, data[0]).Len() }).
					Should(BeEquivalentTo(1))
				node.removeDevice("pump")
				node.death()
				node.birth()
				Eventually(func() task.Status { return taskStatus(ctx, t) }).Should(And(
					HaveField("Variant", status.VariantWarning),
					HaveField("Message", Equal("Failed to read some data")),
					HaveField(
						"Description",
						Equal("device plant/line1/pump is offline"),
					),
				))
			},
		)

		It("Should warn about a tag that the birth does not declare",
			func(ctx SpecContext) {
				_, data := createIndexed(ctx, telem.Float64T)
				node.setTags("", tagValue{name: "other", alias: 1, value: 1.0})
				t := start(ctx, readConfig(
					dev, tagEntry("plant", "line1", "", "temperature", data[0]),
				))
				Eventually(func() task.Status { return taskStatus(ctx, t) }).Should(And(
					HaveField("Variant", status.VariantWarning),
					HaveField("Message", Equal("Failed to read some data")),
					HaveField("Description", ContainSubstring(
						"tag plant/line1/temperature is not in the birth of its edge "+
							"node plant/line1",
					)),
				))
			},
		)

		DescribeTable("Should write a NaN or an infinity of a double tag",
			func(ctx SpecContext, value float64, expected telem.Series) {
				_, data := createIndexed(ctx, expected.DataType)
				node.setTags("", tagValue{
					name:      "temperature",
					alias:     1,
					value:     value,
					timestamp: 1700000000000,
				})
				start(ctx, readConfig(
					dev, tagEntry("plant", "line1", "", "temperature", data[0]),
				))
				Eventually(func() int64 { return stored(ctx, data[0]).Len() }).
					Should(BeEquivalentTo(1))
				Expect(stored(ctx, data[0])).To(telem.MatchSeriesData(expected))
			},
			Entry("NaN into float64", nan, telem.NewSeriesV(nan)),
			Entry("+Inf into float64", math.Inf(1), telem.NewSeriesV(math.Inf(1))),
			Entry("-Inf into float64", math.Inf(-1), telem.NewSeriesV(math.Inf(-1))),
			Entry("NaN into float32", nan, telem.NewSeriesV(float32(nan))),
			Entry("+Inf into float32",
				math.Inf(1), telem.NewSeriesV(float32(math.Inf(1))),
			),
			Entry("-Inf into float32",
				math.Inf(-1), telem.NewSeriesV(float32(math.Inf(-1))),
			),
		)

		It("Should warn about a value that does not fit its channel",
			func(ctx SpecContext) {
				_, data := createIndexed(ctx, telem.Float64T)
				node.setTags("", tagValue{name: "mode", alias: 1, value: "auto"})
				t := start(ctx, readConfig(
					dev, tagEntry("plant", "line1", "", "mode", data[0]),
				))
				Eventually(func() task.Status { return taskStatus(ctx, t) }).Should(And(
					HaveField("Variant", status.VariantWarning),
					HaveField("Message", Equal("Failed to read some data")),
					HaveField("Description", ContainSubstring(
						"rejected a value of tag plant/line1/mode",
					)),
				))
			},
		)
	})

	Describe("Configuration errors", func() {
		DescribeTable(
			"Should reject a config that is not valid",
			func(
				ctx SpecContext,
				build func(ctx context.Context) (msgpack.EncodedJSON, string),
			) {
				cfg, message := build(ctx)
				t := newTask(rackKey, mqtt.ReadTaskType, cfg)
				Expect(factory.ConfigureTask(ctx, t, "configure")).
					Error().To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring(message)),
				))
			},
			Entry("an entry with no edge node",
				func(ctx context.Context) (msgpack.EncodedJSON, string) {
					_, data := createIndexed(ctx, telem.Float64T)
					return readConfig(
						dev, tagEntry("plant", "", "", "temperature", data[0]),
					), "edge_node: required"
				},
			),
			Entry("a group with a topic separator",
				func(ctx context.Context) (msgpack.EncodedJSON, string) {
					_, data := createIndexed(ctx, telem.Float64T)
					return readConfig(
						dev, tagEntry("plant/a", "line1", "", "temperature", data[0]),
					), "group: plant/a must not hold /, +, or #"
				},
			),
			Entry("an entry with no tag",
				func(ctx context.Context) (msgpack.EncodedJSON, string) {
					_, data := createIndexed(ctx, telem.Float64T)
					return readConfig(dev, tagEntry("plant", "line1", "", "", data[0])),
						"tag: required"
				},
			),
			Entry("a tag that appears two times",
				func(ctx context.Context) (msgpack.EncodedJSON, string) {
					_, first := createIndexed(ctx, telem.Float64T)
					_, second := createIndexed(ctx, telem.Float64T)
					entry := tagEntry("plant", "line1", "", "temperature", second[0])
					entry["key"] = "again"
					return readConfig(
						dev,
						tagEntry("plant", "line1", "", "temperature", first[0]),
						entry,
					), "tag plant/line1/temperature appears more than once"
				},
			),
			Entry("two tags that share an index",
				func(ctx context.Context) (msgpack.EncodedJSON, string) {
					_, data := createIndexed(ctx, telem.Float64T, telem.Float64T)
					a := tagEntry("plant", "line1", "", "a", data[0])
					b := tagEntry("plant", "line1", "", "b", data[1])
					return readConfig(dev, a, b),
						"tag plant/line1/a and tag plant/line1/b " +
							"write to the same channel"
				},
			),
			Entry("an index channel as the channel of a tag",
				func(ctx context.Context) (msgpack.EncodedJSON, string) {
					idx, _ := createIndexed(ctx)
					return readConfig(dev, tagEntry("plant", "line1", "", "time", idx)),
						"cannot take the values of a tag"
				},
			),
			Entry("a json channel as the channel of a tag",
				func(ctx context.Context) (msgpack.EncodedJSON, string) {
					ch := createVirtual(ctx, telem.JSONT)
					return readConfig(dev, tagEntry("plant", "line1", "", "doc", ch)),
						"tag plant/line1/doc: channel " + ch.Name +
							" cannot take the values of a tag"
				},
			),
			Entry("a channel that does not exist",
				func(context.Context) (msgpack.EncodedJSON, string) {
					entry := tagEntry(
						"plant", "line1", "", "temperature", channel.Channel{},
					)
					entry["channel"] = 999999
					return readConfig(dev, entry), "channel 999999 does not exist"
				},
			),
		)
	})
})
