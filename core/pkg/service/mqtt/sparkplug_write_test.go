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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/device"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/mqtt"
	"github.com/synnaxlabs/synnax/pkg/service/mqtt/sparkplug"
	"github.com/synnaxlabs/synnax/pkg/service/mqtt/sparkplug/pb"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// tagTarget returns the config of one Sparkplug B write target.
func tagTarget(
	group, edgeNode, dev, tag, sparkplugType string,
	ch channel.Channel,
) map[string]any {
	return map[string]any{
		"key":            group + "/" + edgeNode + "/" + dev + "/" + tag,
		"type":           "sparkplug",
		"group":          group,
		"edge_node":      edgeNode,
		"device":         dev,
		"tag":            tag,
		"sparkplug_type": sparkplugType,
		"channel":        ch.Key(),
	}
}

var _ = Describe("Sparkplug B write targets", func() {
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
		factory = newFactory(64)
		node = startEdgeNode(broker, "plant", "line1")
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

	// metrics returns the one metric of each command that the edge node received.
	metrics := func() []*pb.Payload_Metric {
		GinkgoHelper()
		var out []*pb.Payload_Metric
		for _, c := range node.received() {
			Expect(c.payload.Metrics).To(HaveLen(1))
			out = append(out, c.payload.Metrics[0])
		}
		return out
	}

	doubleValue := func(v float64) types.GomegaMatcher {
		return HaveField("Value", Equal(&pb.Payload_Metric_DoubleValue{DoubleValue: v}))
	}

	Describe("Commands", func() {
		It("Should send an NCMD for a tag of an edge node", func(ctx SpecContext) {
			ch := createVirtual(ctx, telem.Float64T)
			before := uint64(telem.Now() / telem.MillisecondTS)
			start(ctx, writeConfig(dev, tagTarget(
				"plant", "line1", "", "setpoint", "double", ch,
			)))
			command(ctx, ch, telem.NewSeriesV(2.5), func(g Gomega) {
				g.Expect(node.received()).ToNot(BeEmpty())
			})
			after := uint64(telem.Now() / telem.MillisecondTS)
			cmd := node.received()[0]
			Expect(cmd.topic).To(Equal("spBv1.0/plant/NCMD/line1"))
			Expect(cmd.qos).To(BeZero())
			Expect(cmd.payload.Seq).To(BeNil())
			Expect(cmd.payload.Timestamp).ToNot(BeNil())
			Expect(cmd.payload.GetTimestamp()).
				To(And(BeNumerically(">=", before), BeNumerically("<=", after)))
			Expect(cmd.payload.Metrics).To(HaveLen(1))
			m := cmd.payload.Metrics[0]
			Expect(m.GetName()).To(Equal("setpoint"))
			Expect(m.Alias).To(BeNil())
			Expect(m.GetDatatype()).To(BeEquivalentTo(sparkplug.Double))
			Expect(m).To(doubleValue(2.5))
			Expect(m.Timestamp).ToNot(BeNil())
			Expect(m.GetTimestamp()).
				To(And(BeNumerically(">=", before), BeNumerically("<=", after)))
		})

		It("Should send a DCMD for a tag of a device", func(ctx SpecContext) {
			ch := createVirtual(ctx, telem.Float64T)
			start(ctx, writeConfig(dev, tagTarget(
				"plant", "line1", "pump", "speed", "double", ch,
			)))
			command(ctx, ch, telem.NewSeriesV(40.0), func(g Gomega) {
				g.Expect(node.received()).ToNot(BeEmpty())
			})
			cmd := node.received()[0]
			Expect(cmd.topic).To(Equal("spBv1.0/plant/DCMD/line1/pump"))
			Expect(cmd.qos).To(BeZero())
			Expect(cmd.payload.Seq).To(BeNil())
			Expect(cmd.payload.Metrics).To(HaveLen(1))
			Expect(cmd.payload.Metrics[0].GetName()).To(Equal("speed"))
			Expect(cmd.payload.Metrics[0]).To(doubleValue(40))
		})

		DescribeTable("Should send the value of a channel as a Sparkplug B data type",
			func(
				ctx SpecContext,
				series telem.Series,
				sparkplugType string,
				dataType sparkplug.DataType,
				value types.GomegaMatcher,
			) {
				ch := createVirtual(ctx, series.DataType)
				start(ctx, writeConfig(dev, tagTarget(
					"plant", "line1", "", "tag", sparkplugType, ch,
				)))
				command(ctx, ch, series, func(g Gomega) {
					g.Expect(node.received()).ToNot(BeEmpty())
				})
				m := metrics()[0]
				Expect(m.GetDatatype()).To(BeEquivalentTo(dataType))
				Expect(m.GetValue()).To(value)
			},
			Entry("float64 as double",
				telem.NewSeriesV(2.5), "double", sparkplug.Double,
				Equal(&pb.Payload_Metric_DoubleValue{DoubleValue: 2.5}),
			),
			Entry("float32 as float",
				telem.NewSeriesV[float32](1.5), "float", sparkplug.Float,
				Equal(&pb.Payload_Metric_FloatValue{FloatValue: 1.5}),
			),
			// A negative value goes out as the two's complement of its 32 bits.
			Entry("int32 as int32",
				telem.NewSeriesV[int32](-5), "int32", sparkplug.Int32,
				Equal(&pb.Payload_Metric_IntValue{IntValue: 4294967291}),
			),
			Entry("uint8 as boolean",
				telem.NewSeriesV[uint8](1), "boolean", sparkplug.Boolean,
				Equal(&pb.Payload_Metric_BooleanValue{BooleanValue: true}),
			),
			Entry("int64 as int64",
				telem.NewSeriesV[int64](-9000000000), "int64", sparkplug.Int64,
				Equal(&pb.Payload_Metric_LongValue{
					LongValue: 18446744064709551616,
				}),
			),
			Entry("string as string",
				telem.NewSeriesV("auto"), "string", sparkplug.String,
				Equal(&pb.Payload_Metric_StringValue{StringValue: "auto"}),
			),
			Entry("timestamp as date_time in milliseconds",
				telem.NewSeriesV(1700000000123*telem.MillisecondTS),
				"date_time", sparkplug.DateTime,
				Equal(&pb.Payload_Metric_LongValue{LongValue: 1700000000123}),
			),
		)

		It("Should send one command for each sample of a command frame",
			func(ctx SpecContext) {
				ch := createVirtual(ctx, telem.Float64T)
				start(ctx, writeConfig(dev, tagTarget(
					"plant", "line1", "", "step", "double", ch,
				)))
				command(ctx, ch, telem.NewSeriesV(1.0, 2.0), func(g Gomega) {
					g.Expect(metrics()).
						To(ContainElements(doubleValue(1), doubleValue(2)))
				})
				Expect(
					metrics(),
				).To(HaveEach(HaveField("Name", HaveValue(Equal("step")))))
			},
		)

		It("Should skip a disabled target", func(ctx SpecContext) {
			ch := createVirtual(ctx, telem.Float64T)
			disabled := tagTarget("plant", "line1", "", "off", "double", ch)
			disabled["disabled"] = true
			start(ctx, writeConfig(dev,
				disabled, tagTarget("plant", "line1", "", "on", "double", ch),
			))
			command(ctx, ch, telem.NewSeriesV(1.0), func(g Gomega) {
				g.Expect(node.received()).ToNot(BeEmpty())
			})
			Expect(metrics()).To(HaveEach(HaveField("Name", HaveValue(Equal("on")))))
		})

		It("Should send a plain target and a Sparkplug B target of one task",
			func(ctx SpecContext) {
				broker.collect("plant/#")
				plain := createVirtual(ctx, telem.Float64T)
				tag := createVirtual(ctx, telem.Int32T)
				start(ctx, writeConfig(dev,
					plainTarget("plant/valve/set", plain, "/v", "number"),
					tagTarget("plant", "line1", "", "rpm", "int32", tag),
				))
				command(ctx, plain, telem.NewSeriesV(0.5), func(g Gomega) {
					g.Expect(broker.collected()).To(ContainElement(received{
						topic: "plant/valve/set", payload: `{"v":0.5}`,
					}))
				})
				Expect(node.received()).To(BeEmpty())
				command(ctx, tag, telem.NewSeriesV[int32](1200), func(g Gomega) {
					g.Expect(node.received()).ToNot(BeEmpty())
				})
				m := metrics()[0]
				Expect(m.GetName()).To(Equal("rpm"))
				Expect(m.GetIntValue()).To(BeEquivalentTo(1200))
			},
		)
	})

	Describe("Errors while running", func() {
		It("Should warn on a value that does not fit its data type and keep running",
			func(ctx SpecContext) {
				ch := createVirtual(ctx, telem.Int32T)
				t := start(ctx, writeConfig(dev, tagTarget(
					"plant", "line1", "", "level", "int8", ch,
				)))
				command(ctx, ch, telem.NewSeriesV[int32](300), func(g Gomega) {
					stat := taskStatus(ctx, t)
					g.Expect(stat.Variant).To(Equal(status.VariantWarning))
					g.Expect(stat.Message).To(And(
						ContainSubstring("tag level"),
						ContainSubstring("300 as Int8"),
						ContainSubstring("value does not fit the data type"),
					))
					g.Expect(stat.Details.Running).To(BeTrue())
				})
				Expect(node.received()).To(BeEmpty())
				command(ctx, ch, telem.NewSeriesV[int32](100), func(g Gomega) {
					g.Expect(node.received()).ToNot(BeEmpty())
					g.Expect(taskStatus(ctx, t).Variant).
						To(Equal(status.VariantSuccess))
				})
				Expect(metrics()).To(HaveEach(HaveField(
					"Value", Equal(&pb.Payload_Metric_IntValue{IntValue: 100}),
				)))
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
			Expect(factory.ConfigureTask(ctx, t, "configure")).Error().To(And(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(message)),
			))
			stat := taskStatus(ctx, t)
			Expect(stat.Variant).To(Equal(status.VariantError))
			Expect(stat.Message).To(ContainSubstring(message))
			Expect(stat.Details.Cmd).To(Equal("configure"))
		}

		DescribeTable("Should reject a target with an ID that is not valid",
			func(ctx SpecContext, group, edgeNode, deviceID, tag, message string) {
				ch := createVirtual(ctx, telem.Float64T)
				expectRejected(
					ctx,
					writeConfig(
						dev,
						tagTarget(group, edgeNode, deviceID, tag, "double", ch),
					),
					message,
				)
			},
			Entry("no group", "", "line1", "", "speed", "group: required"),
			Entry("no edge node", "plant", "", "", "speed", "edge_node: required"),
			Entry("no tag", "plant", "line1", "", "", "tag: required"),
			Entry("a device with a topic separator",
				"plant", "line1", "pump/1", "speed",
				"device: pump/1 must not hold /, +, or #",
			),
		)

		It("Should reject a channel that does not exist", func(ctx SpecContext) {
			target := tagTarget(
				"plant", "line1", "", "speed", "double", channel.Channel{},
			)
			target["channel"] = 999999
			expectRejected(
				ctx, writeConfig(dev, target),
				"target speed: channel 999999 does not exist",
			)
		})

		It("Should reject a Sparkplug B data type it does not know",
			func(ctx SpecContext) {
				ch := createVirtual(ctx, telem.Float64T)
				expectRejected(
					ctx,
					writeConfig(dev, tagTarget(
						"plant", "line1", "", "speed", "float128", ch,
					)),
					"tag plant/line1/speed: unknown Sparkplug B data type float128",
				)
			},
		)

		DescribeTable("Should reject a channel that its data type cannot carry",
			func(ctx SpecContext, dt telem.DataType, sparkplugType, name string) {
				ch := createVirtual(ctx, dt)
				expectRejected(
					ctx,
					writeConfig(dev, tagTarget(
						"plant", "line1", "pump", "speed", sparkplugType, ch,
					)),
					"tag plant/line1/pump/speed: channel "+ch.Name+" of type "+
						string(dt)+" cannot be sent as "+name,
				)
			},
			Entry("a string channel as double", telem.StringT, "double", "Double"),
			Entry("a float64 channel as string", telem.Float64T, "string", "String"),
			Entry("a timestamp channel as int64", telem.TimestampT, "int64", "Int64"),
		)
	})
})
