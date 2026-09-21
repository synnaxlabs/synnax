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
	"strconv"

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

// field returns the config of one read field.
func field(key string, ch channel.Channel, pointer string) map[string]any {
	return map[string]any{
		"key":     key,
		"name":    key,
		"channel": ch.Key(),
		"pointer": pointer,
	}
}

// plainEntry returns the config of one plain read entry.
func plainEntry(topic string, fields ...map[string]any) map[string]any {
	list := make([]any, len(fields))
	for i, f := range fields {
		list[i] = f
	}
	return map[string]any{
		"key":    topic,
		"type":   "plain",
		"topic":  topic,
		"fields": list,
	}
}

func readConfig(dev device.Device, entries ...map[string]any) msgpack.EncodedJSON {
	list := make([]any, len(entries))
	for i, e := range entries {
		list[i] = e
	}
	return msgpack.EncodedJSON{"device": dev.Key, "entries": list}
}

var _ = Describe("Read task", func() {
	var (
		broker  *testBroker
		rackKey rack.Key
		dev     device.Device
		factory driver.Factory
	)

	BeforeEach(func(ctx SpecContext) {
		broker = startBroker(0)
		rackKey = createRack(ctx)
		dev = createBrokerDevice(ctx, rackKey, broker.port, nil)
		factory = newFactory(64)
	})

	// start configures and starts a read task, and waits for its subscriptions.
	start := func(ctx context.Context, cfg msgpack.EncodedJSON) task.Task {
		GinkgoHelper()
		t := newTask(rackKey, mqtt.ReadTaskType, cfg)
		configured := configure(ctx, factory, t)
		Expect(configured.Exec(ctx, task.Command{Type: "start", Key: "start"})).
			To(Succeed())
		Eventually(broker.subscriptions).
			Should(BeNumerically(">", 0))
		return t
	}

	Describe("Payload conversion", func() {
		It("Should write the fields of a JSON payload with its arrival time",
			func(ctx SpecContext) {
				idx, data := createIndexed(ctx, telem.Float64T, telem.Int32T)
				before := telem.Now()
				t := start(ctx, readConfig(dev, plainEntry(
					"plant/line1",
					field("temperature", data[0], "/temperature"),
					field("count", data[1], "/stats/count"),
				)))
				broker.publish(
					"plant/line1", `{"temperature":23.5,"stats":{"count":7}}`, false,
				)
				Eventually(func() int64 { return stored(ctx, data[0]).Len() }).
					Should(BeEquivalentTo(1))
				Expect(stored(ctx, data[0])).
					To(telem.MatchSeriesData(telem.NewSeriesV(23.5)))
				Expect(stored(ctx, data[1])).
					To(telem.MatchSeriesData(telem.NewSeriesV[int32](7)))
				stamp := stored(ctx, idx).ValueAt[telem.TimeStamp](0)
				Expect(stamp).To(BeNumerically(">=", before))
				Expect(stamp).To(BeNumerically("<=", telem.Now()))
				Expect(taskStatus(ctx, t).Variant).To(Equal(status.VariantSuccess))
			},
		)

		It("Should take the timestamp from the index field", func(ctx SpecContext) {
			idx, data := createIndexed(ctx, telem.Float64T)
			time := field("time", idx, "/ts")
			time["time_format"] = "unix_ms"
			entry := plainEntry(
				"plant/line2", time, field("pressure", data[0], "/pressure"),
			)
			entry["index"] = "time"
			start(ctx, readConfig(dev, entry))
			broker.publish("plant/line2", `{"ts":1000,"pressure":1.5}`, false)
			broker.publish("plant/line2", `{"ts":2000,"pressure":2.5}`, false)
			Eventually(func() int64 { return stored(ctx, data[0]).Len() }).
				Should(BeEquivalentTo(2))
			Expect(stored(ctx, idx)).
				To(telem.MatchSeriesData(telem.NewSeriesSecondsTSV(1, 2)))
			Expect(stored(ctx, data[0])).
				To(telem.MatchSeriesData(telem.NewSeriesV(1.5, 2.5)))
		})

		It("Should take a bare scalar and a bare enum label as the whole payload",
			func(ctx SpecContext) {
				_, number := createIndexed(ctx, telem.Float32T)
				_, state := createIndexed(ctx, telem.Uint8T)
				label := field("state", state[0], "")
				label["enum_values"] = []any{
					map[string]any{"label": "ON", "value": 1},
					map[string]any{"label": "OFF", "value": 0},
				}
				start(ctx, readConfig(
					dev,
					plainEntry("sensors/level", field("level", number[0], "")),
					plainEntry("pumps/1/state", label),
				))
				broker.publish("sensors/level", "23.4", false)
				broker.publish("pumps/1/state", "ON", false)
				Eventually(func() int64 { return stored(ctx, state[0]).Len() }).
					Should(BeEquivalentTo(1))
				Eventually(func() int64 { return stored(ctx, number[0]).Len() }).
					Should(BeEquivalentTo(1))
				Expect(stored(ctx, number[0])).
					To(telem.MatchSeriesData(telem.NewSeriesV[float32](23.4)))
				Expect(stored(ctx, state[0])).
					To(telem.MatchSeriesData(telem.NewSeriesV[uint8](1)))
			},
		)

		It("Should stream and not store when data saving is disabled",
			func(ctx SpecContext) {
				_, data := createIndexed(ctx, telem.Float64T)
				cfg := readConfig(dev, plainEntry(
					"plant/line3", field("temperature", data[0], ""),
				))
				cfg["data_saving_disabled"] = true
				t := start(ctx, cfg)
				broker.publish("plant/line3", "5", false)
				broker.publish("plant/line3", "not a number", false)
				// The second message is rejected, which proves the first was handled.
				Eventually(func() status.Variant { return taskStatus(ctx, t).Variant }).
					Should(Equal(status.VariantWarning))
				Expect(stored(ctx, data[0]).Len()).To(BeZero())
			},
		)
	})

	Describe("Rejected messages", func() {
		It("Should warn and write nothing for a payload that lacks a field",
			func(ctx SpecContext) {
				_, data := createIndexed(ctx, telem.Float64T, telem.Float64T)
				t := start(ctx, readConfig(dev, plainEntry(
					"plant/line4",
					field("a", data[0], "/a"),
					field("b", data[1], "/b"),
				)))
				broker.publish("plant/line4", `{"a":1}`, false)
				Eventually(func(g Gomega) {
					stat := taskStatus(ctx, t)
					g.Expect(stat.Variant).To(Equal(status.VariantWarning))
					g.Expect(stat.Message).To(ContainSubstring(
						"rejected a message on topic plant/line4: " +
							"field b: no value at /b",
					))
					g.Expect(stat.Details.Running).To(BeTrue())
				}).Should(Succeed())
				broker.publish("plant/line4", `{"a":1,"b":2}`, false)
				Eventually(func() int64 { return stored(ctx, data[1]).Len() }).
					Should(BeEquivalentTo(1))
				Expect(stored(ctx, data[0]).Len()).To(BeEquivalentTo(1))
			},
		)

		It("Should reject a timestamp that is not after the last one written",
			func(ctx SpecContext) {
				idx, data := createIndexed(ctx, telem.Float64T)
				time := field("time", idx, "/ts")
				time["time_format"] = "unix_sec"
				entry := plainEntry("plant/line5", time, field("v", data[0], "/v"))
				entry["index"] = "time"
				t := start(ctx, readConfig(dev, entry))
				broker.publish("plant/line5", `{"ts":10,"v":1}`, false)
				broker.publish("plant/line5", `{"ts":10,"v":2}`, false)
				broker.publish("plant/line5", `{"ts":11,"v":3}`, false)
				Eventually(func() int64 { return stored(ctx, data[0]).Len() }).
					Should(BeEquivalentTo(2))
				Expect(stored(ctx, data[0])).
					To(telem.MatchSeriesData(telem.NewSeriesV[float64](1, 3)))
				Expect(taskStatus(ctx, t).Message).
					To(ContainSubstring(
						"its timestamp is not after the last one written",
					))
			},
		)

		It("Should drop the retained message when the entry ignores it",
			func(ctx SpecContext) {
				_, data := createIndexed(ctx, telem.Float64T)
				broker.publish("plant/line6", "1", true)
				entry := plainEntry("plant/line6", field("v", data[0], ""))
				entry["retained_ignored"] = true
				start(ctx, readConfig(dev, entry))
				broker.publish("plant/line6", "2", false)
				Eventually(func() int64 { return stored(ctx, data[0]).Len() }).
					Should(BeEquivalentTo(1))
				Expect(stored(ctx, data[0])).
					To(telem.MatchSeriesData(telem.NewSeriesV[float64](2)))
			},
		)
	})

	Describe("Overload", func() {
		It("Should drop the oldest messages of a full queue and warn",
			func(ctx SpecContext) {
				factory = newFactory(1)
				_, data := createIndexed(ctx, telem.Int64T)
				t := start(ctx, readConfig(dev, plainEntry(
					"plant/burst", field("v", data[0], ""),
				)))
				const burst = 500
				for i := range burst {
					broker.publish("plant/burst", strconv.Itoa(i), false)
				}
				Eventually(func(g Gomega) {
					stat := taskStatus(ctx, t)
					g.Expect(stat.Variant).To(Equal(status.VariantWarning))
					g.Expect(stat.Message).To(ContainSubstring(
						"messages arrive faster than the task can write them",
					))
					g.Expect(stat.Details.Running).To(BeTrue())
				}).Should(Succeed())
				// The newest message survives, so the last sample is the last sent.
				Eventually(func(g Gomega) {
					values := stored(ctx, data[0])
					g.Expect(values.Len()).To(BeNumerically(">", 0))
					g.Expect(values.ValueAt[int64](-1)).
						To(BeEquivalentTo(burst - 1))
				}).Should(Succeed())
				Expect(stored(ctx, data[0]).Len()).To(BeNumerically("<", burst))
			},
		)
	})

	Describe("Connection", func() {
		It("Should share one connection between the tasks of a broker",
			func(ctx SpecContext) {
				_, a := createIndexed(ctx, telem.Float64T)
				_, b := createIndexed(ctx, telem.Float64T)
				start(
					ctx,
					readConfig(dev, plainEntry("shared/a", field("a", a[0], ""))),
				)
				start(
					ctx,
					readConfig(dev, plainEntry("shared/b", field("b", b[0], ""))),
				)
				Eventually(broker.subscriptions).
					Should(Equal(2))
				Expect(broker.clientCount()).To(Equal(1))
				broker.publish("shared/a", "1", false)
				broker.publish("shared/b", "2", false)
				Eventually(func() int64 { return stored(ctx, a[0]).Len() }).
					Should(BeEquivalentTo(1))
				Eventually(func() int64 { return stored(ctx, b[0]).Len() }).
					Should(BeEquivalentTo(1))
			},
		)

		It("Should warn while the broker is down and read again when it returns",
			func(ctx SpecContext) {
				_, data := createIndexed(ctx, telem.Float64T)
				t := start(ctx, readConfig(dev, plainEntry(
					"plant/line7", field("v", data[0], ""),
				)))
				port := broker.port
				broker.stop()
				Eventually(func(g Gomega) {
					stat := taskStatus(ctx, t)
					g.Expect(stat.Variant).To(Equal(status.VariantWarning))
					g.Expect(stat.Message).
						To(ContainSubstring("broker is not connected"))
					g.Expect(stat.Details.Running).To(BeTrue())
				}, "5s").Should(Succeed())
				broker = startBroker(port)
				Eventually(
					func() status.Variant { return taskStatus(ctx, t).Variant },
					"10s",
				).
					Should(Equal(status.VariantSuccess))
				Eventually(
					broker.subscriptions,
					"5s",
				).
					Should(Equal(1))
				broker.publish("plant/line7", "9", false)
				Eventually(func() int64 { return stored(ctx, data[0]).Len() }).
					Should(BeEquivalentTo(1))
			},
		)

		It(
			"Should start with a warning when the broker is down",
			func(ctx SpecContext) {
				_, data := createIndexed(ctx, telem.Float64T)
				port := broker.port
				broker.stop()
				t := newTask(rackKey, mqtt.ReadTaskType, readConfig(dev, plainEntry(
					"plant/line8", field("v", data[0], ""),
				)))
				configured := configure(ctx, factory, t)
				Expect(configured.Exec(ctx, task.Command{Type: "start", Key: "start"})).
					To(Succeed())
				Eventually(func(g Gomega) {
					stat := taskStatus(ctx, t)
					g.Expect(stat.Variant).To(Equal(status.VariantWarning))
					g.Expect(stat.Message).
						To(ContainSubstring("broker is not connected"))
					g.Expect(stat.Details.Running).To(BeTrue())
				}).Should(Succeed())
				broker = startBroker(port)
				Eventually(
					func() status.Variant { return taskStatus(ctx, t).Variant },
					"10s",
				).
					Should(Equal(status.VariantSuccess))
				Eventually(
					broker.subscriptions,
					"5s",
				).
					Should(Equal(1))
				broker.publish("plant/line8", "4", false)
				Eventually(func() int64 { return stored(ctx, data[0]).Len() }).
					Should(BeEquivalentTo(1))
			},
		)
	})

	Describe("Configuration errors", func() {
		expectRejected := func(
			ctx context.Context,
			cfg msgpack.EncodedJSON,
			message string,
		) {
			GinkgoHelper()
			t := newTask(rackKey, mqtt.ReadTaskType, cfg)
			Expect(factory.ConfigureTask(ctx, t, "cmd-1")).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(message)),
			))
			stat := taskStatus(ctx, t)
			Expect(stat.Variant).To(Equal(status.VariantError))
			Expect(stat.Message).To(ContainSubstring(message))
			Expect(stat.Details.Cmd).To(Equal("cmd-1"))
		}

		It("Should reject a task with no broker", func(ctx SpecContext) {
			expectRejected(ctx, msgpack.EncodedJSON{}, "device: a broker is required")
		})

		It("Should reject a broker that does not exist", func(ctx SpecContext) {
			expectRejected(
				ctx,
				msgpack.EncodedJSON{"device": "missing"},
				"device: broker missing does not exist",
			)
		})

		It("Should reject a device that is not a broker", func(ctx SpecContext) {
			other := device.Device{
				Key:      "not-a-broker",
				Rack:     rackKey,
				Location: "here",
				Name:     "Card",
				Make:     "ni",
				Model:    "9205",
			}
			Expect(deviceSvc.NewWriter(nil).Create(ctx, &other)).To(Succeed())
			expectRejected(
				ctx,
				msgpack.EncodedJSON{"device": other.Key},
				"device: Card is not an MQTT broker",
			)
		})

		It("Should reject a task with no enabled entries", func(ctx SpecContext) {
			_, data := createIndexed(ctx, telem.Float64T)
			entry := plainEntry("plant/x", field("v", data[0], ""))
			entry["disabled"] = true
			expectRejected(
				ctx, readConfig(dev, entry), "entries: the task has no enabled entries",
			)
		})

		It("Should reject a topic with a wildcard", func(ctx SpecContext) {
			_, data := createIndexed(ctx, telem.Float64T)
			expectRejected(
				ctx,
				readConfig(dev, plainEntry("plant/+/temp", field("v", data[0], ""))),
				"topic plant/+/temp: must not hold the wildcards + or #",
			)
		})

		It("Should reject a channel that does not exist", func(ctx SpecContext) {
			missing := map[string]any{
				"key": "v", "name": "v", "channel": 4000000, "pointer": "",
			}
			expectRejected(
				ctx,
				readConfig(dev, plainEntry("plant/x", missing)),
				"field v of topic plant/x: channel 4000000 does not exist",
			)
		})

		It("Should reject fields of one topic on different indexes",
			func(ctx SpecContext) {
				_, a := createIndexed(ctx, telem.Float64T)
				_, b := createIndexed(ctx, telem.Float64T)
				expectRejected(
					ctx,
					readConfig(dev, plainEntry(
						"plant/x", field("a", a[0], "/a"), field("b", b[0], "/b"),
					)),
					"every channel of a topic must share one index channel",
				)
			},
		)

		It("Should reject two topics that write to one channel", func(ctx SpecContext) {
			_, data := createIndexed(ctx, telem.Float64T)
			expectRejected(
				ctx,
				readConfig(
					dev,
					plainEntry("plant/x", field("v", data[0], "")),
					plainEntry("plant/y", field("v", data[0], "")),
				),
				"write to the same channel",
			)
		})

		It("Should reject a timestamp field with no time format",
			func(ctx SpecContext) {
				idx, _ := createIndexed(ctx)
				entry := plainEntry("plant/x", field("time", idx, "/ts"))
				entry["index"] = "time"
				expectRejected(
					ctx,
					readConfig(dev, entry),
					"a timestamp channel requires a time format",
				)
			},
		)

		It("Should reject an invalid JSON Pointer", func(ctx SpecContext) {
			_, data := createIndexed(ctx, telem.Float64T)
			expectRejected(
				ctx,
				readConfig(dev, plainEntry("plant/x", field("v", data[0], "no-slash"))),
				"field v of topic plant/x",
			)
		})

		It("Should reject a Sparkplug B entry", func(ctx SpecContext) {
			expectRejected(
				ctx,
				readConfig(dev, map[string]any{"key": "s", "type": "sparkplug"}),
				"Sparkplug B entries are not supported yet",
			)
		})

		It("Should not handle a task type of another integration",
			func(ctx SpecContext) {
				t := newTask(rackKey, "opc_read", msgpack.EncodedJSON{})
				Expect(factory.ConfigureTask(ctx, t, "cmd-1")).Error().
					To(MatchError(driver.ErrTaskNotHandled))
			},
		)
	})

	Describe("Auto start", func() {
		It("Should start a task that is configured to start", func(ctx SpecContext) {
			_, data := createIndexed(ctx, telem.Float64T)
			cfg := readConfig(dev, plainEntry("plant/auto", field("v", data[0], "")))
			cfg["auto_start"] = true
			t := newTask(rackKey, mqtt.ReadTaskType, cfg)
			configured := MustSucceed(factory.ConfigureTask(ctx, t, driver.NoCommand))
			DeferCleanup(func() { Expect(configured.Stop(false)).To(Succeed()) })
			Expect(taskStatus(ctx, t).Details.Running).To(BeTrue())
		})
	})
})
