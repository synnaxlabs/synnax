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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/device"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
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
		factory = MustSucceed(mqtt.NewFactory(mqtt.FactoryConfig{
			Device:          deviceSvc,
			Channel:         channelSvc,
			Framer:          framerSvc,
			Status:          statusSvc,
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
			}, "200ms").Should(Equal(status.VariantSuccess))
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
				}, "200ms").Should(Equal(status.VariantSuccess))
				Expect(stored(ctx, data[0]).Len()).To(BeEquivalentTo(1))
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
				HaveField(
					"Message",
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
				HaveField(
					"Message",
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
				HaveField(
					"Message",
					ContainSubstring("edge node plant/line1 is offline"),
				),
			))
		})

		It("Should warn about a tag that the birth does not declare",
			func(ctx SpecContext) {
				_, data := createIndexed(ctx, telem.Float64T)
				node.setTags("", tagValue{name: "other", alias: 1, value: 1.0})
				t := start(ctx, readConfig(
					dev, tagEntry("plant", "line1", "", "temperature", data[0]),
				))
				Eventually(func() task.Status { return taskStatus(ctx, t) }).Should(And(
					HaveField("Variant", status.VariantWarning),
					HaveField("Message", ContainSubstring(
						"tag plant/line1/temperature is not in the birth of its edge "+
							"node plant/line1",
					)),
				))
			},
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
					HaveField("Message", ContainSubstring(
						"rejected a value of tag plant/line1/mode",
					)),
				))
			},
		)
	})

	Describe("Configuration errors", func() {
		configureErr := func(ctx context.Context, cfg msgpack.EncodedJSON) error {
			t := newTask(rackKey, mqtt.ReadTaskType, cfg)
			_, err := factory.ConfigureTask(ctx, t, "configure")
			return err
		}

		It("Should reject an entry with no edge node", func(ctx SpecContext) {
			_, data := createIndexed(ctx, telem.Float64T)
			err := configureErr(ctx, readConfig(
				dev, tagEntry("plant", "", "", "temperature", data[0]),
			))
			Expect(err).To(MatchError(validate.ErrValidation))
			Expect(err).To(MatchError(ContainSubstring("edge_node: required")))
		})

		It("Should reject a group with a topic separator", func(ctx SpecContext) {
			_, data := createIndexed(ctx, telem.Float64T)
			err := configureErr(ctx, readConfig(
				dev, tagEntry("plant/a", "line1", "", "temperature", data[0]),
			))
			Expect(err).To(MatchError(validate.ErrValidation))
			Expect(err).To(MatchError(ContainSubstring(
				"group: plant/a must not hold /, +, or #",
			)))
		})

		It("Should reject an entry with no tag", func(ctx SpecContext) {
			_, data := createIndexed(ctx, telem.Float64T)
			err := configureErr(ctx, readConfig(
				dev, tagEntry("plant", "line1", "", "", data[0]),
			))
			Expect(err).To(MatchError(validate.ErrValidation))
			Expect(err).To(MatchError(ContainSubstring("tag: required")))
		})

		It("Should reject a tag that appears two times", func(ctx SpecContext) {
			_, first := createIndexed(ctx, telem.Float64T)
			_, second := createIndexed(ctx, telem.Float64T)
			entry := tagEntry("plant", "line1", "", "temperature", second[0])
			entry["key"] = "again"
			err := configureErr(ctx, readConfig(
				dev, tagEntry("plant", "line1", "", "temperature", first[0]), entry,
			))
			Expect(err).To(MatchError(validate.ErrValidation))
			Expect(err).To(MatchError(ContainSubstring(
				"tag plant/line1/temperature appears more than once",
			)))
		})

		It("Should reject two tags that share an index", func(ctx SpecContext) {
			_, data := createIndexed(ctx, telem.Float64T, telem.Float64T)
			err := configureErr(ctx, readConfig(
				dev,
				tagEntry("plant", "line1", "", "a", data[0]),
				tagEntry("plant", "line1", "", "b", data[1]),
			))
			Expect(err).To(MatchError(validate.ErrValidation))
			Expect(err).To(MatchError(ContainSubstring(
				"tag plant/line1/a and tag plant/line1/b write to the same channel",
			)))
		})

		It("Should reject an index channel as the channel of a tag",
			func(ctx SpecContext) {
				idx, _ := createIndexed(ctx)
				err := configureErr(ctx, readConfig(
					dev, tagEntry("plant", "line1", "", "time", idx),
				))
				Expect(err).To(MatchError(validate.ErrValidation))
				Expect(err).To(MatchError(ContainSubstring(
					"cannot take the values of a tag",
				)))
			},
		)

		It("Should reject a channel that does not exist", func(ctx SpecContext) {
			entry := tagEntry("plant", "line1", "", "temperature", channel.Channel{})
			entry["channel"] = 999999
			err := configureErr(ctx, readConfig(dev, entry))
			Expect(err).To(MatchError(validate.ErrValidation))
			Expect(
				err,
			).To(MatchError(ContainSubstring("channel 999999 does not exist")))
		})
	})
})
