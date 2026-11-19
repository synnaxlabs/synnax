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
	"encoding/json/v2"
	"math"
	"time"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/mqtt"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/payload"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// messageTarget returns the config of a plain write target that publishes m of ic to
// topic, with bindings.
func messageTarget(
	topic string,
	ic icd.ICD,
	m icd.Message,
	bindings ...map[string]any,
) map[string]any {
	return map[string]any{
		"key":   topic,
		"type":  "plain",
		"topic": topic,
		"icd":   ic.Key,
		"message": map[string]any{
			"message":  m.Key,
			"bindings": lo.ToAnySlice(bindings),
		},
	}
}

// fieldTarget returns the config of a plain write target that publishes to topic a
// message of fields in a new ICD, with the first field bound to ch.
func fieldTarget(
	ctx context.Context,
	topic string,
	ch channel.Channel,
	fields ...icd.JSONField,
) map[string]any {
	GinkgoHelper()
	m := jsonMessage("Command", nil, fields...)
	return messageTarget(topic, createICD(ctx, m), m, binding(m, 0, ch))
}

func writeConfig(dev device.Device, targets ...map[string]any) msgpack.EncodedJSON {
	return msgpack.EncodedJSON{"device": dev.Key, "targets": lo.ToAnySlice(targets)}
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

	start := func(ctx context.Context, cfg msgpack.EncodedJSON) (task.Task, driver.Task) {
		GinkgoHelper()
		t := newTask(ctx, rackKey, mqtt.WriteTaskType, cfg)
		configured := configure(ctx, factory, t)
		Expect(configured.Exec(ctx, task.Command{Type: "start", Key: "start"})).
			To(Succeed())
		return t, configured
	}

	// send writes fr to keys until check passes, because a streamer misses the frames
	// written before it connects.
	send := func(
		ctx context.Context,
		keys channel.Keys,
		fr frame.Frame,
		check func(g Gomega),
		intervals ...any,
	) {
		GinkgoHelper()
		w := MustSucceed(framerSvc.OpenWriter(ctx, framer.WriterConfig{
			Keys:  keys,
			Start: telem.Now(),
		}))
		defer func() { Expect(w.Close()).To(Succeed()) }()
		Eventually(func(g Gomega) {
			g.Expect(w.Write(fr)).To(BeTrue())
			check(g)
		}, intervals...).Should(Succeed())
	}

	command := func(
		ctx context.Context,
		ch channel.Channel,
		series telem.Series,
		check func(g Gomega),
	) {
		GinkgoHelper()
		send(ctx, channel.Keys{ch.Key()}, frame.NewUnary(ch.Key(), series), check)
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
			start(ctx, writeConfig(dev, fieldTarget(
				ctx, "plant/valve1/set", ch,
				jsonField("setpoint", "/setpoint/value", icd.FloatJSONField{}),
			)))
			command(ctx, ch, telem.NewSeriesV[float32](1.5), func(g Gomega) {
				g.Expect(payloads("plant/valve1/set")).
					To(ContainElement(`{"setpoint":{"value":1.5}}`))
			})
		})

		It("Should publish a value at the root pointer as the whole payload",
			func(ctx SpecContext) {
				ch := createVirtual(ctx, telem.Uint8T)
				start(ctx, writeConfig(dev, fieldTarget(
					ctx, "plant/valve2/set", ch,
					jsonField("open", "", icd.BooleanJSONField{}),
				)))
				command(ctx, ch, telem.NewSeriesV[uint8](1), func(g Gomega) {
					g.Expect(payloads("plant/valve2/set")).To(ContainElement("true"))
				})
			},
		)

		It("Should publish bytes as base64", func(ctx SpecContext) {
			ch := createVirtual(ctx, telem.BytesT)
			start(ctx, writeConfig(dev, fieldTarget(
				ctx, "plant/blob", ch, jsonField("b", "/b", icd.BytesJSONField{}),
			)))
			value := telem.NewSeriesV([]byte{0x00, 0xff, 0x3f})
			command(ctx, ch, value, func(g Gomega) {
				g.Expect(payloads("plant/blob")).To(ContainElement(`{"b":"AP8/"}`))
			})
		})

		It("Should publish a timestamp in the time format of its field",
			func(ctx SpecContext) {
				ch := createVirtual(ctx, telem.TimestampT)
				start(ctx, writeConfig(dev, fieldTarget(
					ctx, "plant/clock/set", ch, jsonField("t", "/t",
						icd.TimestampJSONField{TimeFormat: payload.TimeFormatISO8601},
					),
				)))
				command(ctx, ch, telem.NewSeriesSecondsTSV(1), func(g Gomega) {
					g.Expect(payloads("plant/clock/set")).
						To(ContainElement(`{"t":"1970-01-01T00:00:01Z"}`))
				})
			},
		)

		It("Should publish the label of an enum value, and a value outside it as is",
			func(ctx SpecContext) {
				ch := createVirtual(ctx, telem.Uint8T)
				start(ctx, writeConfig(dev, fieldTarget(
					ctx, "plant/valve3/set", ch,
					jsonField("state", "/state", icd.UnsignedJSONField{
						EnumValues: []payload.UnsignedEntry{
							{Label: "closed", Value: 0}, {Label: "open", Value: 1},
						},
					}),
				)))
				command(ctx, ch, telem.NewSeriesV[uint8](1, 5), func(g Gomega) {
					g.Expect(payloads("plant/valve3/set")).
						To(ContainElements(`{"state":"open"}`, `{"state":5}`))
				})
			},
		)

		It("Should add the default of each unbound field to each payload",
			func(ctx SpecContext) {
				ch := createVirtual(ctx, telem.Int32T)
				start(ctx, writeConfig(dev, fieldTarget(
					ctx,
					"plant/pump/set",
					ch,
					jsonField("rpm", "/rpm", icd.IntegerJSONField{}),
					jsonField(
						"unit",
						"/unit",
						icd.StringJSONField{Default: new("rpm")},
					),
					jsonField("priority", "/priority", icd.IntegerJSONField{
						Default: new(payload.Integer(3)),
					}),
					jsonField("id", "/id", icd.UuidJSONField{
						Default: &icd.UUIDDefault{Variant: icd.RandomUUIDDefault{}},
					}),
				)))
				command(ctx, ch, telem.NewSeriesV[int32](1200), func(g Gomega) {
					g.Expect(payloads("plant/pump/set")).ToNot(BeEmpty())
				})
				published := payloads("plant/pump/set")
				ids := make([]string, 0, len(published))
				for _, payload := range published {
					var doc map[string]any
					Expect(json.Unmarshal([]byte(payload), &doc)).To(Succeed())
					Expect(doc).To(HaveKeyWithValue("rpm", BeEquivalentTo(1200)))
					Expect(doc).To(HaveKeyWithValue("unit", "rpm"))
					Expect(doc).To(HaveKeyWithValue("priority", BeEquivalentTo(3)))
					id, ok := doc["id"].(string)
					Expect(ok).To(BeTrue())
					MustSucceed(uuid.Parse(id))
					ids = append(ids, id)
				}
				Expect(lo.Uniq(ids)).To(HaveLen(len(ids)))
			},
		)

		It(
			"Should write the members of a payload in field order",
			func(ctx SpecContext) {
				ch := createVirtual(ctx, telem.Int32T)
				start(ctx, writeConfig(dev, fieldTarget(
					ctx,
					"plant/order/set",
					ch,
					jsonField("value", "/reading/value", icd.IntegerJSONField{}),
					jsonField(
						"source",
						"/source",
						icd.StringJSONField{Default: new("plc")},
					),
					jsonField("unit", "/reading/unit", icd.StringJSONField{
						Default: new("rpm"),
					}),
				)))
				command(ctx, ch, telem.NewSeriesV[int32](1200), func(g Gomega) {
					g.Expect(payloads("plant/order/set")).To(ContainElement(
						`{"reading":{"value":1200,"unit":"rpm"},"source":"plc"}`,
					))
				})
			},
		)

		It("Should publish one message for each sample of a command frame",
			func(ctx SpecContext) {
				ch := createVirtual(ctx, telem.Int32T)
				start(ctx, writeConfig(dev, fieldTarget(
					ctx,
					"plant/steps",
					ch,
					jsonField("step", "/step", icd.IntegerJSONField{}),
				)))
				command(ctx, ch, telem.NewSeriesV[int32](1, 2), func(g Gomega) {
					g.Expect(payloads("plant/steps")).
						To(ContainElements(`{"step":1}`, `{"step":2}`))
				})
			},
		)

		It("Should publish every target of a command channel", func(ctx SpecContext) {
			ch := createVirtual(ctx, telem.Float64T)
			start(ctx, writeConfig(
				dev,
				fieldTarget(
					ctx,
					"plant/a/set",
					ch,
					jsonField("v", "/v", icd.FloatJSONField{}),
				),
				fieldTarget(
					ctx,
					"plant/b/set",
					ch,
					jsonField("v", "", icd.FloatJSONField{}),
				),
			))
			command(ctx, ch, telem.NewSeriesV(2.5), func(g Gomega) {
				g.Expect(payloads("plant/a/set")).To(ContainElement(`{"v":2.5}`))
				g.Expect(payloads("plant/b/set")).To(ContainElement(`2.5`))
			})
		})

		It("Should set the retained flag of a retained target", func(ctx SpecContext) {
			ch := createVirtual(ctx, telem.Float64T)
			target := fieldTarget(
				ctx, "plant/mode", ch, jsonField("v", "", icd.FloatJSONField{}),
			)
			target["retained"] = true
			start(ctx, writeConfig(dev, target))
			command(ctx, ch, telem.NewSeriesV(4.0), func(g Gomega) {
				g.Expect(payloads("plant/mode")).To(ContainElement("4"))
			})
			Expect(broker.retained("plant/mode")).To(Equal([]string{"4"}))
		})

		It("Should skip a disabled target", func(ctx SpecContext) {
			ch := createVirtual(ctx, telem.Float64T)
			disabled := fieldTarget(
				ctx, "plant/off/set", ch, jsonField("v", "", icd.FloatJSONField{}),
			)
			disabled["disabled"] = true
			start(ctx, writeConfig(dev, disabled, fieldTarget(
				ctx, "plant/on/set", ch, jsonField("v", "", icd.FloatJSONField{}),
			)))
			command(ctx, ch, telem.NewSeriesV(1.0), func(g Gomega) {
				g.Expect(payloads("plant/on/set")).To(ContainElement("1"))
			})
			Expect(payloads("plant/off/set")).To(BeEmpty())
		})
	})

	Describe("Bound fields", func() {
		var (
			x, y channel.Channel
			m    icd.Message
			ic   icd.ICD
		)

		// pointTarget returns the config of a target that sends x and y, where y has
		// the fields of yField.
		pointTarget := func(ctx context.Context, yField icd.JSONFieldVariant) map[string]any {
			GinkgoHelper()
			x, y = createVirtual(
				ctx,
				telem.Float64T,
			), createVirtual(
				ctx,
				telem.Float64T,
			)
			m = jsonMessage("Point", nil,
				jsonField("x", "/x", icd.FloatJSONField{}),
				jsonField("y", "/y", yField),
			)
			ic = createICD(ctx, m)
			return messageTarget(
				"plant/point",
				ic,
				m,
				binding(m, 0, x),
				binding(m, 1, y),
			)
		}

		both := func(xs, ys telem.Series) frame.Frame {
			return frame.NewMulti(
				[]channel.Key{x.Key(), y.Key()}, []telem.Series{xs, ys},
			)
		}

		It("Should publish the commands of one frame as one message",
			func(ctx SpecContext) {
				start(ctx, writeConfig(dev, pointTarget(ctx, icd.FloatJSONField{})))
				send(ctx, channel.Keys{x.Key(), y.Key()},
					both(telem.NewSeriesV(1.0), telem.NewSeriesV(2.0)),
					func(g Gomega) {
						g.Expect(payloads("plant/point")).ToNot(BeEmpty())
					},
				)
				Expect(payloads("plant/point")).To(HaveEach(`{"x":1,"y":2}`))
			},
		)

		It("Should send the last command of a bound channel with no new one",
			func(ctx SpecContext) {
				start(ctx, writeConfig(dev, pointTarget(ctx, icd.FloatJSONField{})))
				send(ctx, channel.Keys{x.Key(), y.Key()},
					both(telem.NewSeriesV(1.0), telem.NewSeriesV(2.0)),
					func(g Gomega) {
						g.Expect(payloads("plant/point")).ToNot(BeEmpty())
					},
				)
				command(ctx, x, telem.NewSeriesV(3.0), func(g Gomega) {
					g.Expect(payloads("plant/point")).
						To(ContainElement(`{"x":3,"y":2}`))
				})
			},
		)

		It("Should send the default of a bound field with no command yet",
			func(ctx SpecContext) {
				start(ctx, writeConfig(dev, pointTarget(
					ctx, icd.FloatJSONField{Default: new(payload.Float(5))},
				)))
				command(ctx, x, telem.NewSeriesV(1.0), func(g Gomega) {
					g.Expect(payloads("plant/point")).
						To(ContainElement(`{"x":1,"y":5}`))
				})
			},
		)

		It("Should wait for the first command of a bound field with no default",
			func(ctx SpecContext) {
				start(ctx, writeConfig(dev, pointTarget(ctx, icd.FloatJSONField{})))
				w := MustSucceed(framerSvc.OpenWriter(ctx, framer.WriterConfig{
					Keys:  channel.Keys{x.Key()},
					Start: telem.Now(),
				}))
				Consistently(func(g Gomega) {
					g.Expect(w.Write(frame.NewUnary(x.Key(), telem.NewSeriesV(1.0)))).
						To(BeTrue())
					g.Expect(payloads("plant/point")).To(BeEmpty())
				}, 500*time.Millisecond).Should(Succeed())
				Expect(w.Close()).To(Succeed())
				command(ctx, y, telem.NewSeriesV(2.0), func(g Gomega) {
					g.Expect(payloads("plant/point")).
						To(ContainElement(`{"x":1,"y":2}`))
				})
			},
		)

		It("Should forget the commands of an earlier run", func(ctx SpecContext) {
			_, configured := start(ctx, writeConfig(dev, pointTarget(
				ctx, icd.FloatJSONField{},
			)))
			send(ctx, channel.Keys{x.Key(), y.Key()},
				both(telem.NewSeriesV(1.0), telem.NewSeriesV(2.0)),
				func(g Gomega) {
					g.Expect(payloads("plant/point")).ToNot(BeEmpty())
				},
			)
			Expect(configured.Exec(ctx, task.Command{Type: "stop", Key: "stop"})).
				To(Succeed())
			Expect(configured.Exec(ctx, task.Command{Type: "start", Key: "again"})).
				To(Succeed())
			w := MustSucceed(framerSvc.OpenWriter(ctx, framer.WriterConfig{
				Keys:  channel.Keys{x.Key()},
				Start: telem.Now(),
			}))
			Consistently(func(g Gomega) {
				g.Expect(w.Write(frame.NewUnary(x.Key(), telem.NewSeriesV(3.0)))).
					To(BeTrue())
				g.Expect(payloads("plant/point")).ToNot(ContainElement(`{"x":3,"y":2}`))
			}, 500*time.Millisecond).Should(Succeed())
			Expect(w.Close()).To(Succeed())
			command(ctx, y, telem.NewSeriesV(4.0), func(g Gomega) {
				g.Expect(payloads("plant/point")).To(ContainElement(`{"x":3,"y":4}`))
			})
		})

		It("Should keep the last command of a field after a command it cannot hold",
			func(ctx SpecContext) {
				start(ctx, writeConfig(dev, pointTarget(ctx, icd.FloatJSONField{})))
				send(ctx, channel.Keys{x.Key(), y.Key()},
					both(telem.NewSeriesV(1.0), telem.NewSeriesV(2.0)),
					func(g Gomega) {
						g.Expect(payloads("plant/point")).ToNot(BeEmpty())
					},
				)
				command(ctx, x, telem.NewSeriesV(math.NaN()), func(g Gomega) {
					g.Expect(broker.collected()).ToNot(BeEmpty())
				})
				command(ctx, y, telem.NewSeriesV(4.0), func(g Gomega) {
					g.Expect(payloads("plant/point")).
						To(ContainElement(`{"x":1,"y":4}`))
				})
			},
		)
	})

	Describe("Errors while running", func() {
		It("Should warn with a count of the commands it drops and keep running",
			func(ctx SpecContext) {
				ch := createVirtual(ctx, telem.Float64T)
				t, _ := start(ctx, writeConfig(dev, fieldTarget(
					ctx, "plant/nan", ch, jsonField("v", "/v", icd.FloatJSONField{}),
				)))
				command(ctx, ch, telem.NewSeriesV(math.NaN()), func(g Gomega) {
					stat := taskStatus(ctx, t)
					g.Expect(stat.Variant).To(Equal(status.VariantWarning))
					g.Expect(stat.Message).
						To(MatchRegexp(`^Skipped [\d,]+ of [\d,]+ commands?$`))
					g.Expect(stat.Description).To(Equal(
						"Command for target plant/nan: message Command: field v: " +
							"NaN is not a JSON number: validation error",
					))
					g.Expect(stat.Details.Running).To(BeTrue())
				})
				send(ctx, channel.Keys{ch.Key()},
					frame.NewUnary(ch.Key(), telem.NewSeriesV(7.0)),
					func(g Gomega) {
						g.Expect(payloads("plant/nan")).To(ContainElement(`{"v":7}`))
						g.Expect(taskStatus(ctx, t).Variant).
							To(Equal(status.VariantSuccess))
					},
					2*driver.SkipHold, 100*time.Millisecond,
				)
			},
		)

		It("Should warn while the broker is down and drop the commands of the outage",
			func(ctx SpecContext) {
				ch := createVirtual(ctx, telem.Float64T)
				t, _ := start(ctx, writeConfig(dev, fieldTarget(
					ctx, "plant/outage", ch, jsonField("v", "/v", icd.FloatJSONField{}),
				)))
				Eventually(broker.clientCount).Should(Equal(1))
				port := broker.port
				broker.stop()
				Eventually(func(g Gomega) {
					stat := taskStatus(ctx, t)
					g.Expect(stat.Variant).To(Equal(status.VariantWarning))
					g.Expect(stat.Message).To(Equal("Failed to write data"))
					g.Expect(stat.Description).To(Equal("broker is not connected"))
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

		It("Should warn and keep running when the connection drops under a publish",
			func(ctx SpecContext) {
				ch := createVirtual(ctx, telem.Float64T)
				target := fieldTarget(
					ctx,
					"plant/unacked",
					ch,
					jsonField("v", "/v", icd.FloatJSONField{}),
				)
				target["qos"] = "at_least_once"
				unacked := &unackedHook{topic: "plant/unacked"}
				Expect(broker.server.AddHook(unacked, nil)).To(Succeed())
				t, _ := start(ctx, writeConfig(dev, target))
				command(ctx, ch, telem.NewSeriesV(1.0), func(g Gomega) {
					g.Expect(unacked.held.Load()).To(BeNumerically(">", 0))
				})
				unacked.released.Store(true)
				broker.dropClients()
				Eventually(func(g Gomega) {
					stat := taskStatus(ctx, t)
					g.Expect(stat.Variant).To(Equal(status.VariantWarning))
					g.Expect(stat.Message).To(Equal("Failed to write data"))
					g.Expect(stat.Details.Running).To(BeTrue())
				}, 5*time.Second).Should(Succeed())
				Eventually(
					func() status.Variant { return taskStatus(ctx, t).Variant },
					10*time.Second,
				).
					Should(Equal(status.VariantSuccess))
				command(ctx, ch, telem.NewSeriesV(2.0), func(g Gomega) {
					g.Expect(payloads("plant/unacked")).To(ContainElement(`{"v":2}`))
				})
			},
		)
	})

	DescribeTable("Should report a drop that strands a publish in the client",
		func(ctx SpecContext, blocks bool) {
			stall := newPublishStall(blocks)
			factory = stall.factory()
			ch := createVirtual(ctx, telem.Float64T)
			t, _ := start(ctx, writeConfig(dev, fieldTarget(
				ctx, "plant/stalled", ch, jsonField("v", "/v", icd.FloatJSONField{}),
			)))
			DeferCleanup(stall.release)
			stall.armed.Store(true)
			command(ctx, ch, telem.NewSeriesV(1.0), func(g Gomega) {
				g.Expect(stall.stalled.Load()).To(BeTrue())
			})
			broker.dropClients()
			Eventually(func(g Gomega) {
				stat := taskStatus(ctx, t)
				g.Expect(stat.Variant).To(Equal(status.VariantWarning))
				g.Expect(stat.Description).To(ContainSubstring("not connected"))
				g.Expect(stat.Details.Running).To(BeTrue())
			}, 3*time.Second).Should(Succeed())
			stall.release()
			Eventually(
				func() status.Variant { return taskStatus(ctx, t).Variant },
				10*time.Second,
			).
				Should(Equal(status.VariantSuccess))
		},
		Entry("with a token that never completes", false),
		Entry("with a call that blocks", true),
	)

	Describe("Configuration errors", func() {
		expectRejected := func(
			ctx context.Context,
			cfg msgpack.EncodedJSON,
			message string,
		) {
			GinkgoHelper()
			t := newTask(ctx, rackKey, mqtt.WriteTaskType, cfg)
			Expect(factory.ConfigureTask(ctx, t, "configure")).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(message)),
			))
			stat := taskStatus(ctx, t)
			Expect(stat.Variant).To(Equal(status.VariantError))
			Expect(stat.Message).To(Equal("Failed to configure task"))
			Expect(stat.Description).To(ContainSubstring(message))
			Expect(stat.Details.Cmd).To(Equal("configure"))
		}

		// valueTarget returns the config of a target that publishes to topic the value
		// of ch at /v.
		valueTarget := func(
			ctx context.Context,
			topic string,
			ch channel.Channel,
		) map[string]any {
			return fieldTarget(
				ctx,
				topic,
				ch,
				jsonField("v", "/v", icd.FloatJSONField{}),
			)
		}

		DescribeTable(
			"Should reject a config that is not valid",
			func(
				ctx SpecContext,
				build func(ctx context.Context) (msgpack.EncodedJSON, string),
			) {
				cfg, message := build(ctx)
				expectRejected(ctx, cfg, message)
			},
			Entry("no enabled targets",
				func(ctx context.Context) (msgpack.EncodedJSON, string) {
					disabled := valueTarget(
						ctx,
						"plant/x",
						createVirtual(ctx, telem.Float64T),
					)
					disabled["disabled"] = true
					return writeConfig(dev, disabled),
						"targets: the task has no enabled targets"
				},
			),
			Entry("a null target", func(context.Context) (msgpack.EncodedJSON, string) {
				return msgpack.EncodedJSON{"device": dev.Key, "targets": []any{nil}},
					"targets: a target is null"
			}),
			Entry("a topic that holds a wildcard",
				func(ctx context.Context) (msgpack.EncodedJSON, string) {
					ch := createVirtual(ctx, telem.Float64T)
					return writeConfig(dev, valueTarget(ctx, "plant/+/set", ch)),
						"topic plant/+/set: must not hold the wildcards + or #"
				},
			),
			Entry("an unknown QoS",
				func(ctx context.Context) (msgpack.EncodedJSON, string) {
					target := valueTarget(
						ctx,
						"plant/x",
						createVirtual(ctx, telem.Float64T),
					)
					target["qos"] = "bogus"
					return writeConfig(dev, target),
						`target plant/x: qos: unknown value "bogus"`
				},
			),
			Entry("a channel that does not exist",
				func(ctx context.Context) (msgpack.EncodedJSON, string) {
					missing := channel.Channel{
						Name:     "missing",
						DataType: telem.Float64T,
					}
					return writeConfig(dev, valueTarget(ctx, "plant/x", missing)),
						"target plant/x: channel 0 does not exist"
				},
			),
			Entry("an ICD that does not exist",
				func(ctx context.Context) (msgpack.EncodedJSON, string) {
					m := jsonMessage(
						"Command",
						nil,
						jsonField("v", "/v", icd.FloatJSONField{}),
					)
					missing := icd.ICD{Key: uuid.New()}
					ch := createVirtual(ctx, telem.Float64T)
					return writeConfig(dev, messageTarget(
						"plant/x", missing, m, binding(m, 0, ch),
					)), "icd: ICD " + missing.Key.String() + " does not exist"
				},
			),
			Entry("a message that the ICD lacks",
				func(ctx context.Context) (msgpack.EncodedJSON, string) {
					ic := createICD(ctx)
					m := jsonMessage(
						"Command",
						nil,
						jsonField("v", "/v", icd.FloatJSONField{}),
					)
					ch := createVirtual(ctx, telem.Float64T)
					return writeConfig(dev, messageTarget(
							"plant/x", ic, m, binding(m, 0, ch),
						)), "target plant/x: message " + m.Key.String() +
							" does not exist in ICD Plant"
				},
			),
			Entry("a message that is not JSON",
				func(ctx context.Context) (msgpack.EncodedJSON, string) {
					m := icd.Message{
						Key:     uuid.New(),
						Name:    "Frame",
						Payload: icd.Payload{Variant: icd.BinaryPayload{}},
					}
					ch := createVirtual(ctx, telem.Float64T)
					return writeConfig(dev, messageTarget(
						"plant/x", createICD(ctx, m), m,
						map[string]any{"field": uuid.New(), "channel": ch.Key()},
					)), "target plant/x: message Frame is not a JSON message"
				},
			),
			Entry("a message that binds no fields",
				func(ctx context.Context) (msgpack.EncodedJSON, string) {
					m := jsonMessage(
						"Command",
						nil,
						jsonField("v", "/v", icd.FloatJSONField{
							Default: new(payload.Float(1)),
						}),
					)
					return writeConfig(
							dev,
							messageTarget("plant/x", createICD(ctx, m), m),
						),
						"target plant/x: message Command binds no fields"
				},
			),
			Entry("a field with no binding and no default",
				func(ctx context.Context) (msgpack.EncodedJSON, string) {
					ch := createVirtual(ctx, telem.Float64T)
					return writeConfig(dev, fieldTarget(
							ctx, "plant/x", ch,
							jsonField("v", "/v", icd.FloatJSONField{}),
							jsonField("unit", "/unit", icd.StringJSONField{}),
						)), "target plant/x: message Command: field unit: no channel and " +
							"no default"
				},
			),
			Entry("a channel that does not write to its field",
				func(ctx context.Context) (msgpack.EncodedJSON, string) {
					ch := createVirtual(ctx, telem.StringT)
					return writeConfig(dev, valueTarget(ctx, "plant/x", ch)),
						"target plant/x: message Command: field v: cannot write " +
							"string channels to float fields"
				},
			),
			Entry("a field at the parent of another",
				func(ctx context.Context) (msgpack.EncodedJSON, string) {
					ch := createVirtual(ctx, telem.Float64T)
					return writeConfig(dev, fieldTarget(
							ctx, "plant/x", ch,
							jsonField("value", "/reading/value", icd.FloatJSONField{}),
							jsonField("reading", "/reading", icd.StringJSONField{
								Default: new("rpm"),
							}),
						)), `target plant/x: message Command: field value "/reading/value" ` +
							`and field reading "/reading" overlap`
				},
			),
		)
	})
})
