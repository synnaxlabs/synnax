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
	"encoding/json"
	"math"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/device"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/mqtt"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// plainTarget returns the config of one plain write target, whose command channel
// value goes to pointer as jsonType.
func plainTarget(
	topic string,
	ch channel.Channel,
	pointer, jsonType string,
	fields ...map[string]any,
) map[string]any {
	list := make([]any, len(fields))
	for i, f := range fields {
		list[i] = f
	}
	return map[string]any{
		"key":   topic,
		"type":  "plain",
		"topic": topic,
		"channel": map[string]any{
			"channel":   ch.Key(),
			"name":      ch.Name,
			"data_type": string(ch.DataType),
			"pointer":   pointer,
			"json_type": jsonType,
		},
		"fields": list,
	}
}

func staticField(pointer, jsonType string, value any) map[string]any {
	return map[string]any{
		"key":       pointer,
		"type":      "static",
		"pointer":   pointer,
		"json_type": jsonType,
		"value":     value,
	}
}

func generatedField(pointer, generator string) map[string]any {
	return map[string]any{
		"key":       pointer,
		"type":      "generated",
		"pointer":   pointer,
		"generator": generator,
	}
}

func writeConfig(dev device.Device, targets ...map[string]any) msgpack.EncodedJSON {
	list := make([]any, len(targets))
	for i, t := range targets {
		list[i] = t
	}
	return msgpack.EncodedJSON{"device": dev.Key, "targets": list}
}

var _ = Describe("Write task", func() {
	var (
		broker  *testBroker
		rackKey rack.Key
		dev     device.Device
		factory driver.Factory
	)

	BeforeEach(func(ctx SpecContext) {
		broker = startBroker(0)
		broker.collect("#")
		rackKey = createRack(ctx)
		dev = createBrokerDevice(ctx, rackKey, broker.port, nil)
		factory = newFactory(64)
	})

	start := func(ctx context.Context, cfg msgpack.EncodedJSON) task.Task {
		GinkgoHelper()
		t := newTask(rackKey, mqtt.WriteTaskType, cfg)
		configured := configure(ctx, factory, t)
		Expect(configured.Exec(ctx, task.Command{Type: "start", Key: "start"})).
			To(Succeed())
		return t
	}

	// command writes series to ch until check passes, because a streamer misses the
	// frames written before it connects.
	command := func(
		ctx context.Context,
		ch channel.Channel,
		series telem.Series,
		check func(g Gomega),
	) {
		GinkgoHelper()
		w := MustSucceed(framerSvc.OpenWriter(ctx, framer.WriterConfig{
			Keys:  channel.Keys{ch.Key()},
			Start: telem.Now(),
		}))
		defer func() { Expect(w.Close()).To(Succeed()) }()
		Eventually(func(g Gomega) {
			g.Expect(w.Write(frame.NewUnary(ch.Key(), series))).To(BeTrue())
			check(g)
		}).Should(Succeed())
	}

	// payloads returns the payloads that the broker received on topic.
	payloads := func(topic string) []string {
		var out []string
		for _, m := range broker.collected() {
			if m.topic == topic {
				out = append(out, m.payload)
			}
		}
		return out
	}

	Describe("Payloads", func() {
		It("Should publish a command value at its pointer", func(ctx SpecContext) {
			ch := createVirtual(ctx, telem.Float32T)
			start(ctx, writeConfig(dev, plainTarget(
				"plant/valve1/set", ch, "/setpoint/value", "number",
			)))
			command(ctx, ch, telem.NewSeriesV[float32](1.5), func(g Gomega) {
				g.Expect(payloads("plant/valve1/set")).
					To(ContainElement(`{"setpoint":{"value":1.5}}`))
			})
		})

		It("Should publish a value at the root pointer as the whole payload",
			func(ctx SpecContext) {
				ch := createVirtual(ctx, telem.Uint8T)
				start(ctx, writeConfig(dev, plainTarget(
					"plant/valve2/set", ch, "", "boolean",
				)))
				command(ctx, ch, telem.NewSeriesV[uint8](1), func(g Gomega) {
					g.Expect(payloads("plant/valve2/set")).To(ContainElement("true"))
				})
			},
		)

		It("Should publish the label of an enum value", func(ctx SpecContext) {
			ch := createVirtual(ctx, telem.Uint8T)
			target := plainTarget("plant/valve3/set", ch, "/state", "string")
			target["channel"].(map[string]any)["enum_values"] = []any{
				map[string]any{"label": "closed", "value": 0},
				map[string]any{"label": "open", "value": 1},
			}
			start(ctx, writeConfig(dev, target))
			command(ctx, ch, telem.NewSeriesV[uint8](1), func(g Gomega) {
				g.Expect(payloads("plant/valve3/set")).
					To(ContainElement(`{"state":"open"}`))
			})
		})

		It("Should add static and generated fields to each payload",
			func(ctx SpecContext) {
				ch := createVirtual(ctx, telem.Int32T)
				start(ctx, writeConfig(dev, plainTarget(
					"plant/pump/set", ch, "/rpm", "number",
					staticField("/unit", "string", "rpm"),
					staticField("/priority", "number", 3),
					generatedField("/id", "uuid"),
				)))
				command(ctx, ch, telem.NewSeriesV[int32](1200), func(g Gomega) {
					g.Expect(payloads("plant/pump/set")).ToNot(BeEmpty())
				})
				var ids []string
				for _, payload := range payloads("plant/pump/set") {
					var doc map[string]any
					Expect(json.Unmarshal([]byte(payload), &doc)).To(Succeed())
					Expect(doc).To(HaveKeyWithValue("rpm", BeEquivalentTo(1200)))
					Expect(doc).To(HaveKeyWithValue("unit", "rpm"))
					Expect(doc).To(HaveKeyWithValue("priority", BeEquivalentTo(3)))
					id := doc["id"].(string)
					Expect(uuid.Parse(id)).Error().ToNot(HaveOccurred())
					ids = append(ids, id)
				}
				Expect(lo.Uniq(ids)).To(HaveLen(len(ids)))
			},
		)

		It("Should publish one message for each sample of a command frame",
			func(ctx SpecContext) {
				ch := createVirtual(ctx, telem.Int32T)
				start(ctx, writeConfig(dev, plainTarget(
					"plant/steps", ch, "/step", "number",
				)))
				command(ctx, ch, telem.NewSeriesV[int32](1, 2), func(g Gomega) {
					g.Expect(payloads("plant/steps")).
						To(ContainElements(`{"step":1}`, `{"step":2}`))
				})
			},
		)

		It("Should publish every target of a command channel", func(ctx SpecContext) {
			ch := createVirtual(ctx, telem.Float64T)
			start(ctx, writeConfig(dev,
				plainTarget("plant/a/set", ch, "/v", "number"),
				plainTarget("plant/b/set", ch, "", "string"),
			))
			command(ctx, ch, telem.NewSeriesV(2.5), func(g Gomega) {
				g.Expect(payloads("plant/a/set")).To(ContainElement(`{"v":2.5}`))
				g.Expect(payloads("plant/b/set")).To(ContainElement(`"2.5"`))
			})
		})

		It("Should set the retained flag of a retained target", func(ctx SpecContext) {
			ch := createVirtual(ctx, telem.Float64T)
			target := plainTarget("plant/mode", ch, "", "number")
			target["retained"] = true
			start(ctx, writeConfig(dev, target))
			command(ctx, ch, telem.NewSeriesV(4.0), func(g Gomega) {
				g.Expect(payloads("plant/mode")).To(ContainElement("4"))
			})
			Expect(broker.retained("plant/mode")).To(Equal([]string{"4"}))
		})

		It("Should skip a disabled target", func(ctx SpecContext) {
			ch := createVirtual(ctx, telem.Float64T)
			disabled := plainTarget("plant/off/set", ch, "", "number")
			disabled["disabled"] = true
			start(ctx, writeConfig(dev,
				disabled, plainTarget("plant/on/set", ch, "", "number"),
			))
			command(ctx, ch, telem.NewSeriesV(1.0), func(g Gomega) {
				g.Expect(payloads("plant/on/set")).To(ContainElement("1"))
			})
			Expect(payloads("plant/off/set")).To(BeEmpty())
		})
	})

	Describe("Errors while running", func() {
		It("Should warn on a value with no JSON form and keep running",
			func(ctx SpecContext) {
				ch := createVirtual(ctx, telem.Float64T)
				t := start(ctx, writeConfig(dev, plainTarget(
					"plant/nan", ch, "/v", "number",
				)))
				command(ctx, ch, telem.NewSeriesV(math.NaN()), func(g Gomega) {
					stat := taskStatus(ctx, t)
					g.Expect(stat.Variant).To(Equal(status.VariantWarning))
					g.Expect(stat.Message).To(ContainSubstring("value is not finite"))
					g.Expect(stat.Details.Running).To(BeTrue())
				})
				command(ctx, ch, telem.NewSeriesV(7.0), func(g Gomega) {
					g.Expect(payloads("plant/nan")).To(ContainElement(`{"v":7}`))
					g.Expect(taskStatus(ctx, t).Variant).
						To(Equal(status.VariantSuccess))
				})
			},
		)

		It("Should warn while the broker is down and drop the commands of the outage",
			func(ctx SpecContext) {
				ch := createVirtual(ctx, telem.Float64T)
				t := start(ctx, writeConfig(dev, plainTarget(
					"plant/outage", ch, "/v", "number",
				)))
				Eventually(broker.clientCount).Should(Equal(1))
				port := broker.port
				broker.stop()
				Eventually(func(g Gomega) {
					stat := taskStatus(ctx, t)
					g.Expect(stat.Variant).To(Equal(status.VariantWarning))
					g.Expect(stat.Message).
						To(ContainSubstring("broker is not connected"))
					g.Expect(stat.Details.Running).To(BeTrue())
				}, 5*time.Second).Should(Succeed())
				w := MustSucceed(framerSvc.OpenWriter(ctx, framer.WriterConfig{
					Keys:  channel.Keys{ch.Key()},
					Start: telem.Now(),
				}))
				Expect(w.Write(frame.NewUnary(ch.Key(), telem.NewSeriesV(1.0)))).
					To(BeTrue())
				Expect(w.Close()).To(Succeed())

				broker = startBroker(port)
				broker.collect("#")
				Eventually(
					func() status.Variant { return taskStatus(ctx, t).Variant },
					10*time.Second,
				).
					Should(Equal(status.VariantSuccess))
				command(ctx, ch, telem.NewSeriesV(2.0), func(g Gomega) {
					g.Expect(payloads("plant/outage")).To(ContainElement(`{"v":2}`))
				})
				Expect(payloads("plant/outage")).ToNot(ContainElement(`{"v":1}`))
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
			t := newTask(rackKey, mqtt.WriteTaskType, cfg)
			Expect(factory.ConfigureTask(ctx, t, "configure")).Error().
				To(MatchError(ContainSubstring(message)))
			stat := taskStatus(ctx, t)
			Expect(stat.Variant).To(Equal(status.VariantError))
			Expect(stat.Message).To(ContainSubstring(message))
			Expect(stat.Details.Cmd).To(Equal("configure"))
		}

		It("Should reject a task with no enabled targets", func(ctx SpecContext) {
			ch := createVirtual(ctx, telem.Float64T)
			disabled := plainTarget("plant/x", ch, "", "number")
			disabled["disabled"] = true
			expectRejected(
				ctx, writeConfig(dev, disabled),
				"targets: the task has no enabled targets",
			)
		})

		It("Should reject a topic that holds a wildcard", func(ctx SpecContext) {
			ch := createVirtual(ctx, telem.Float64T)
			expectRejected(
				ctx, writeConfig(dev, plainTarget("plant/+/set", ch, "", "number")),
				"topic plant/+/set: must not hold the wildcards + or #",
			)
		})

		It("Should reject a channel that does not exist", func(ctx SpecContext) {
			ch := channel.Channel{Name: "missing", DataType: telem.Float64T}
			expectRejected(
				ctx, writeConfig(dev, plainTarget("plant/x", ch, "", "number")),
				"target plant/x: channel 0 does not exist",
			)
		})

		It("Should reject a JSON type the channel cannot convert to",
			func(ctx SpecContext) {
				ch := createVirtual(ctx, telem.StringT)
				expectRejected(
					ctx, writeConfig(dev, plainTarget("plant/x", ch, "", "number")),
					"target plant/x: channel "+ch.Name+
						": cannot convert string to a JSON number",
				)
			},
		)

		It("Should reject enum values for a JSON type that is not string",
			func(ctx SpecContext) {
				ch := createVirtual(ctx, telem.Uint8T)
				target := plainTarget("plant/x", ch, "/v", "number")
				target["channel"].(map[string]any)["enum_values"] = []any{
					map[string]any{"label": "open", "value": 1},
				}
				expectRejected(
					ctx, writeConfig(dev, target),
					"target plant/x: enum values require the string JSON type",
				)
			},
		)

		It("Should reject extra fields next to a root pointer", func(ctx SpecContext) {
			ch := createVirtual(ctx, telem.Float64T)
			expectRejected(
				ctx,
				writeConfig(dev, plainTarget(
					"plant/x", ch, "", "number", staticField("/unit", "string", "rpm"),
				)),
				"a channel value that is the whole payload allows no other fields",
			)
		})

		It("Should reject a static value that is not of its JSON type",
			func(ctx SpecContext) {
				ch := createVirtual(ctx, telem.Float64T)
				expectRejected(
					ctx,
					writeConfig(dev, plainTarget(
						"plant/x", ch, "/v", "number",
						staticField("/unit", "number", "rpm"),
					)),
					"target plant/x: static value at /unit is not a number",
				)
			},
		)

		It("Should reject fields whose pointers block each other",
			func(ctx SpecContext) {
				ch := createVirtual(ctx, telem.Float64T)
				expectRejected(
					ctx,
					writeConfig(dev, plainTarget(
						"plant/x", ch, "/unit/value", "number",
						staticField("/unit", "string", "rpm"),
					)),
					"target plant/x: cannot set /unit/value: "+
						"a scalar value blocks the path",
				)
			},
		)

		It("Should reject a timestamp channel with no time format",
			func(ctx SpecContext) {
				ch := createVirtual(ctx, telem.TimestampT)
				expectRejected(
					ctx, writeConfig(dev, plainTarget("plant/x", ch, "/t", "number")),
					"target plant/x: a timestamp channel requires a time format",
				)
			},
		)
	})
})
