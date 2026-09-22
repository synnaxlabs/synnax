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
	"slices"
	"time"

	"github.com/google/uuid"
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
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// edgeTag returns the config of one tag of an edge node task.
func edgeTag(name string, ch channel.Channel, sparkplugType string) map[string]any {
	return map[string]any{
		"key":            name,
		"name":           name,
		"channel":        ch.Key(),
		"sparkplug_type": sparkplugType,
	}
}

// edgeConfig returns the config of an edge node task for plant/line1 whose commands
// write with authority 200.
func edgeConfig(dev device.Device, tags ...map[string]any) msgpack.EncodedJSON {
	list := make([]any, len(tags))
	for i, t := range tags {
		list[i] = t
	}
	return msgpack.EncodedJSON{
		"device":    dev.Key,
		"group":     "plant",
		"edge_node": "line1",
		"authority": 200,
		"tags":      list,
	}
}

func millis(ts telem.TimeStamp) uint64 { return uint64(ts / telem.MillisecondTS) }

var _ = Describe("Edge task", func() {
	var (
		broker  *testBroker
		rackKey rack.Key
		dev     device.Device
		factory driver.Factory
		host    *testHost
	)

	BeforeEach(func(ctx SpecContext) {
		broker = startBroker(0)
		rackKey = createRack(ctx)
		dev = createBrokerDevice(ctx, rackKey, broker.port, nil)
		factory = newFactory(64)
		host = startHost(broker, "plant", "line1")
	})

	// start configures and starts an edge task, and waits for its birth.
	start := func(ctx context.Context, cfg msgpack.EncodedJSON) (driver.Task, task.Task) {
		GinkgoHelper()
		t := newTask(rackKey, mqtt.EdgeTaskType, cfg)
		configured := configure(ctx, factory, t)
		Expect(configured.Exec(ctx, task.Command{Type: "start", Key: "start"})).
			To(Succeed())
		Eventually(host.births).Should(HaveLen(1))
		return configured, t
	}

	// nextStamp returns the first of samples timestamps one millisecond apart, after
	// every sample written so far. A coarse platform clock repeats values, and Cesium
	// takes only data that comes after what it holds.
	var lastStamp telem.TimeStamp
	nextStamp := func(samples int64) telem.TimeStamp {
		stamp := telem.Now()
		if stamp <= lastStamp {
			stamp = lastStamp + 1
		}
		lastStamp = stamp + telem.TimeStamp(samples-1)*telem.MillisecondTS
		return stamp
	}

	// stream writes series on ch, stamped at idx when idx is a channel, until check
	// passes, because a streamer misses the frames written before it connects. Each
	// attempt writes at a later time, and check receives the timestamp of the first
	// sample of every attempt so far, because the NDATA of an attempt can arrive after
	// its check ran.
	stream := func(
		ctx context.Context,
		idx, ch channel.Channel,
		series telem.Series,
		check func(g Gomega, written []telem.TimeStamp),
	) {
		GinkgoHelper()
		keys := channel.Keys{ch.Key()}
		if idx.Key() != 0 {
			keys = append(keys, idx.Key())
		}
		w := MustSucceed(framerSvc.OpenWriter(ctx, framer.WriterConfig{
			Keys:  keys,
			Start: nextStamp(1),
		}))
		defer func() { Expect(w.Close()).To(Succeed()) }()
		var written []telem.TimeStamp
		Eventually(func(g Gomega) {
			stamp := nextStamp(series.Len())
			fr := frame.NewUnary(ch.Key(), series)
			if idx.Key() != 0 {
				stamps := make([]telem.TimeStamp, series.Len())
				for i := range stamps {
					stamps[i] = stamp + telem.TimeStamp(i)*telem.MillisecondTS
				}
				fr = fr.Append(idx.Key(), telem.NewSeriesV(stamps...))
			}
			g.Expect(w.Write(fr)).To(BeTrue())
			written = append(written, stamp)
			check(g, written)
		}).Should(Succeed())
	}

	// streamValue writes series on ch until the host sees an NDATA with it. For an
	// indexed channel it returns the timestamp of the attempt that the host saw.
	streamValue := func(
		ctx context.Context,
		idx, ch channel.Channel,
		series telem.Series,
	) telem.TimeStamp {
		GinkgoHelper()
		if idx.Key() == 0 {
			before := len(host.data())
			stream(ctx, idx, ch, series, func(g Gomega, _ []telem.TimeStamp) {
				g.Expect(len(host.data())).To(BeNumerically(">", before))
			})
			return 0
		}
		var seen telem.TimeStamp
		stream(ctx, idx, ch, series, func(g Gomega, written []telem.TimeStamp) {
			for _, stamp := range written {
				if len(host.dataAt(stamp)) > 0 {
					seen = stamp
					return
				}
			}
			g.Expect(written).To(BeEmpty(), "no NDATA for a written stamp yet")
		})
		return seen
	}

	expectSequential := func(seqs []uint64) {
		GinkgoHelper()
		for i, seq := range seqs {
			Expect(seq).To(Equal(uint64(i)))
		}
	}

	expectStatus := func(
		ctx context.Context,
		t task.Task,
		variant status.Variant,
		message string,
	) {
		GinkgoHelper()
		Eventually(func(g Gomega) {
			stat := taskStatus(ctx, t)
			g.Expect(stat.Variant).To(Equal(variant))
			g.Expect(stat.Message).To(ContainSubstring(message))
			g.Expect(stat.Details.Running).To(BeTrue())
		}, 10*time.Second).Should(Succeed())
	}

	Describe("Birth", func() {
		It("Should publish an NBIRTH on start that declares every tag",
			func(ctx SpecContext) {
				_, data := createIndexed(ctx, telem.Float64T)
				count := createVirtual(ctx, telem.Int32T)
				before := millis(telem.Now())
				_, t := start(ctx, edgeConfig(dev,
					edgeTag("temp", data[0], "double"),
					edgeTag("count", count, "int32"),
				))
				after := millis(telem.Now())
				birth := host.births()[0]
				Expect(birth.topic).To(Equal("spBv1.0/plant/NBIRTH/line1"))
				Expect(birth.qos).To(BeZero())
				Expect(birth.retained).To(BeFalse())
				Expect(birth.payload.Seq).To(HaveValue(Equal(uint64(0))))
				Expect(birth.payload.GetTimestamp()).To(And(
					BeNumerically(">=", before), BeNumerically("<=", after),
				))
				Expect(birth.payload.Metrics).To(HaveLen(4))
				bdSeq := birth.payload.Metrics[0]
				Expect(bdSeq.GetName()).To(Equal(sparkplug.BdSeqMetric))
				Expect(bdSeq.GetDatatype()).To(BeEquivalentTo(sparkplug.Int64))
				Expect(bdSeq.GetValue()).To(Equal(
					&pb.Payload_Metric_LongValue{LongValue: 0},
				))
				rebirth := birth.payload.Metrics[1]
				Expect(rebirth.GetName()).To(Equal(sparkplug.RebirthMetric))
				Expect(rebirth.GetDatatype()).To(BeEquivalentTo(sparkplug.Boolean))
				Expect(rebirth.GetValue()).To(Equal(
					&pb.Payload_Metric_BooleanValue{BooleanValue: false},
				))
				temp := birth.payload.Metrics[2]
				Expect(temp.GetName()).To(Equal("temp"))
				Expect(temp.Alias).To(HaveValue(Equal(uint64(1))))
				Expect(temp.GetDatatype()).To(BeEquivalentTo(sparkplug.Double))
				Expect(temp.GetIsNull()).To(BeTrue())
				Expect(temp.GetValue()).To(BeNil())
				countMetric := birth.payload.Metrics[3]
				Expect(countMetric.GetName()).To(Equal("count"))
				Expect(countMetric.Alias).To(HaveValue(Equal(uint64(2))))
				Expect(countMetric.GetDatatype()).To(BeEquivalentTo(sparkplug.Int32))
				Expect(countMetric.GetIsNull()).To(BeTrue())
				// The edge node listens for commands on its own client.
				Expect(broker.clientCount()).To(Equal(1))
				Expect(broker.subscriptions()).To(Equal(1))
				stat := taskStatus(ctx, t)
				Expect(stat.Variant).To(Equal(status.VariantSuccess))
				Expect(stat.Details.Running).To(BeTrue())
			},
		)

		It("Should skip a disabled tag and alias the enabled ones in order",
			func(ctx SpecContext) {
				ch := createVirtual(ctx, telem.Float64T)
				disabled := edgeTag("off", ch, "double")
				disabled["disabled"] = true
				start(ctx, edgeConfig(dev,
					disabled, edgeTag("a", ch, "double"), edgeTag("b", ch, "double"),
				))
				birth := host.births()[0]
				Expect(birth.payload.Metrics).To(HaveLen(4))
				Expect(birth.payload.Metrics[2].GetName()).To(Equal("a"))
				Expect(birth.payload.Metrics[2].Alias).To(HaveValue(Equal(uint64(1))))
				Expect(birth.payload.Metrics[3].GetName()).To(Equal("b"))
				Expect(birth.payload.Metrics[3].Alias).To(HaveValue(Equal(uint64(2))))
			},
		)

		It("Should publish the birth again with the last values on a rebirth request",
			func(ctx SpecContext) {
				idx, data := createIndexed(ctx, telem.Float64T)
				count := createVirtual(ctx, telem.Int32T)
				start(ctx, edgeConfig(dev,
					edgeTag("temp", data[0], "double"),
					edgeTag("count", count, "int32"),
				))
				streamValue(ctx, idx, data[0], telem.NewSeriesV(21.5))
				host.sendRebirth()
				Eventually(host.births).Should(HaveLen(2))
				birth := host.births()[1]
				Expect(birth.payload.Seq).To(HaveValue(Equal(uint64(0))))
				Expect(birth.bdSeq()).To(Equal(uint64(0)))
				Expect(birth.metric(sparkplug.RebirthMetric).GetBooleanValue()).
					To(BeFalse())
				temp := birth.metric("temp")
				Expect(temp.Alias).To(HaveValue(Equal(uint64(1))))
				Expect(temp.GetIsNull()).To(BeFalse())
				Expect(temp.GetValue()).To(Equal(
					&pb.Payload_Metric_DoubleValue{DoubleValue: 21.5},
				))
				// The value keeps the timestamp of its index.
				var published []uint64
				for _, m := range host.data() {
					published = append(published, m.payload.Metrics[0].GetTimestamp())
				}
				Expect(temp.GetTimestamp()).To(BeElementOf(published))
				Expect(birth.metric("count").GetIsNull()).To(BeTrue())
				// The rebirth starts the sequence again.
				streamValue(ctx, idx, data[0], telem.NewSeriesV(22.5))
				seqs := host.sequence()
				Expect(seqs).To(HaveLen(len(host.data()) + 2))
				expectSequential(seqs[slices.Index(seqs[1:], 0)+1:])
			},
		)
	})

	Describe("Data", func() {
		It("Should publish the samples of a frame as one NDATA stamped by the index",
			func(ctx SpecContext) {
				idx, data := createIndexed(ctx, telem.Float64T)
				start(ctx, edgeConfig(dev, edgeTag("temp", data[0], "double")))
				stamp := streamValue(ctx, idx, data[0], telem.NewSeriesV(1.5, 2.5))
				m := host.dataAt(stamp)[0]
				Expect(m.topic).To(Equal("spBv1.0/plant/NDATA/line1"))
				Expect(m.qos).To(BeZero())
				Expect(m.retained).To(BeFalse())
				Expect(m.payload.GetTimestamp()).To(BeNumerically(">=", millis(stamp)))
				Expect(m.payload.Metrics).To(HaveLen(2))
				for i, expected := range []float64{1.5, 2.5} {
					pm := m.payload.Metrics[i]
					Expect(pm.Name).To(BeNil())
					Expect(pm.Alias).To(HaveValue(Equal(uint64(1))))
					Expect(pm.GetDatatype()).To(BeEquivalentTo(sparkplug.Double))
					Expect(pm.GetValue()).To(Equal(
						&pb.Payload_Metric_DoubleValue{DoubleValue: expected},
					))
					Expect(pm.GetTimestamp()).To(Equal(millis(stamp) + uint64(i)))
				}
				expectSequential(host.sequence())
			},
		)

		It("Should count the sequence up across NDATA messages", func(ctx SpecContext) {
			idx, data := createIndexed(ctx, telem.Float64T)
			start(ctx, edgeConfig(dev, edgeTag("temp", data[0], "double")))
			for _, v := range []float64{1, 2, 3} {
				streamValue(ctx, idx, data[0], telem.NewSeriesV(v))
			}
			seqs := host.sequence()
			Expect(len(seqs)).To(BeNumerically(">=", 4))
			expectSequential(seqs)
		})

		It("Should stamp the samples of a virtual channel with the publish time",
			func(ctx SpecContext) {
				ch := createVirtual(ctx, telem.Float64T)
				start(ctx, edgeConfig(dev, edgeTag("temp", ch, "double")))
				before := millis(telem.Now())
				streamValue(ctx, channel.Channel{}, ch, telem.NewSeriesV(4.5))
				after := millis(telem.Now())
				m := host.data()[0]
				Expect(m.payload.Metrics).To(HaveLen(1))
				Expect(m.payload.Metrics[0].GetValue()).To(Equal(
					&pb.Payload_Metric_DoubleValue{DoubleValue: 4.5},
				))
				Expect(m.payload.Metrics[0].GetTimestamp()).To(And(
					BeNumerically(">=", before), BeNumerically("<=", after),
				))
				Expect(m.payload.Metrics[0].GetTimestamp()).
					To(Equal(m.payload.GetTimestamp()))
			},
		)

		DescribeTable("Should encode a sample as the Sparkplug B data type of its tag",
			func(
				ctx SpecContext,
				series telem.Series,
				sparkplugType string,
				dataType sparkplug.DataType,
				value types.GomegaMatcher,
			) {
				ch := createVirtual(ctx, series.DataType)
				start(ctx, edgeConfig(dev, edgeTag("tag", ch, sparkplugType)))
				Expect(host.births()[0].metric("tag").GetDatatype()).
					To(BeEquivalentTo(dataType))
				streamValue(ctx, channel.Channel{}, ch, series)
				m := host.data()[0].metricByAlias(1)
				Expect(m.GetDatatype()).To(BeEquivalentTo(dataType))
				Expect(m.GetValue()).To(value)
			},
			Entry("float64 as double",
				telem.NewSeriesV(2.5), "double", sparkplug.Double,
				Equal(&pb.Payload_Metric_DoubleValue{DoubleValue: 2.5}),
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

		It("Should publish every tag of one channel", func(ctx SpecContext) {
			ch := createVirtual(ctx, telem.Float64T)
			start(ctx, edgeConfig(dev,
				edgeTag("celsius", ch, "double"), edgeTag("rounded", ch, "int32"),
			))
			streamValue(ctx, channel.Channel{}, ch, telem.NewSeriesV(3.0))
			m := host.data()[0]
			Expect(m.payload.Metrics).To(HaveLen(2))
			Expect(m.metricByAlias(1).GetValue()).To(Equal(
				&pb.Payload_Metric_DoubleValue{DoubleValue: 3},
			))
			Expect(m.metricByAlias(2).GetValue()).To(Equal(
				&pb.Payload_Metric_IntValue{IntValue: 3},
			))
		})
	})

	Describe("Commands", func() {
		// commandTag returns the config of a tag that takes commands on cmd.
		commandTag := func(name string, ch, cmd channel.Channel) map[string]any {
			tag := edgeTag(name, ch, "double")
			tag["command_channel"] = cmd.Key()
			return tag
		}

		It("Should write a command by name to the command channel at its arrival time",
			func(ctx SpecContext) {
				cmdIdx, cmdData := createIndexed(ctx, telem.Float64T)
				start(ctx, edgeConfig(dev, commandTag(
					"setpoint", createVirtual(ctx, telem.Float64T), cmdData[0],
				)))
				before := telem.Now()
				host.sendCommand(commandMetric("setpoint", 0, 2.5))
				Eventually(func() int64 { return stored(ctx, cmdData[0]).Len() }).
					Should(BeEquivalentTo(1))
				after := telem.Now()
				Expect(stored(ctx, cmdData[0])).
					To(telem.MatchSeriesData(telem.NewSeriesV(2.5)))
				Expect(stored(ctx, cmdIdx).ValueAt[telem.TimeStamp](0)).To(And(
					BeNumerically(">=", before), BeNumerically("<=", after),
				))
			},
		)

		It("Should write a command by alias to the command channel",
			func(ctx SpecContext) {
				_, cmdData := createIndexed(ctx, telem.Float64T)
				start(ctx, edgeConfig(
					dev,
					edgeTag("temp", createVirtual(ctx, telem.Float64T), "double"),
					commandTag(
						"setpoint",
						createVirtual(ctx, telem.Float64T),
						cmdData[0],
					),
				))
				host.sendCommand(commandMetric("", 2, 3.5))
				Eventually(func() int64 { return stored(ctx, cmdData[0]).Len() }).
					Should(BeEquivalentTo(1))
				Expect(stored(ctx, cmdData[0])).
					To(telem.MatchSeriesData(telem.NewSeriesV(3.5)))
			},
		)

		It("Should write nothing for a tag that has no command channel",
			func(ctx SpecContext) {
				_, cmdData := createIndexed(ctx, telem.Float64T)
				_, t := start(ctx, edgeConfig(
					dev,
					edgeTag("temp", createVirtual(ctx, telem.Float64T), "double"),
					commandTag(
						"setpoint",
						createVirtual(ctx, telem.Float64T),
						cmdData[0],
					),
				))
				host.sendCommand(
					commandMetric("temp", 0, 9.0),
					commandMetric("setpoint", 0, 1.5),
				)
				Eventually(func() int64 { return stored(ctx, cmdData[0]).Len() }).
					Should(BeEquivalentTo(1))
				Expect(stored(ctx, cmdData[0])).
					To(telem.MatchSeriesData(telem.NewSeriesV(1.5)))
				Consistently(func() status.Variant {
					return taskStatus(ctx, t).Variant
				}, 200*time.Millisecond).Should(Equal(status.VariantSuccess))
			},
		)

		DescribeTable("Should convert a command value to the type of the channel",
			func(
				ctx SpecContext,
				dt telem.DataType,
				value any,
				expected telem.Series,
			) {
				_, cmdData := createIndexed(ctx, dt)
				start(ctx, edgeConfig(dev, commandTag(
					"setpoint", createVirtual(ctx, telem.Float64T), cmdData[0],
				)))
				host.sendCommand(commandMetric("setpoint", 0, value))
				Eventually(func() int64 { return stored(ctx, cmdData[0]).Len() }).
					Should(BeEquivalentTo(1))
				Expect(stored(ctx, cmdData[0])).To(telem.MatchSeriesData(expected))
			},
			Entry("a double into an int32 channel",
				telem.Int32T, 7.0, telem.NewSeriesV[int32](7),
			),
			Entry("a boolean into a uint8 channel",
				telem.Uint8T, true, telem.NewSeriesV[uint8](1),
			),
			Entry("an int32 into a float64 channel",
				telem.Float64T, int32(-3), telem.NewSeriesV(-3.0),
			),
			Entry("a numeric string into a float64 channel",
				telem.Float64T, "2.5", telem.NewSeriesV(2.5),
			),
		)

		It("Should write with the authority of the config", func(ctx SpecContext) {
			cmdIdx, cmdData := createIndexed(ctx, telem.Float64T)
			keys := channel.Keys{cmdData[0].Key(), cmdIdx.Key()}
			_, t := start(ctx, edgeConfig(dev, commandTag(
				"setpoint", createVirtual(ctx, telem.Float64T), cmdData[0],
			)))
			lower := MustSucceed(framerSvc.OpenWriter(ctx, framer.WriterConfig{
				Keys:        keys,
				Start:       telem.Now(),
				Authorities: []control.Authority{199},
			}))
			DeferCleanup(func() { Expect(lower.Close()).To(Succeed()) })
			host.sendCommand(commandMetric("setpoint", 0, 1.5))
			Eventually(func() int64 { return stored(ctx, cmdData[0]).Len() }).
				Should(BeEquivalentTo(1))
			equal := MustSucceed(framerSvc.OpenWriter(ctx, framer.WriterConfig{
				Keys:        keys,
				Start:       telem.Now(),
				Authorities: []control.Authority{200},
			}))
			DeferCleanup(func() { Expect(equal.Close()).To(Succeed()) })
			host.sendCommand(commandMetric("setpoint", 0, 2.5))
			expectStatus(ctx, t, status.VariantWarning, "unauthorized")
			Expect(stored(ctx, cmdData[0])).
				To(telem.MatchSeriesData(telem.NewSeriesV(1.5)))
		})

		It("Should warn on a command that a higher authority blocks and clear the "+
			"warning on the next command that is written",
			func(ctx SpecContext) {
				cmdIdx, cmdData := createIndexed(ctx, telem.Float64T)
				_, t := start(ctx, edgeConfig(dev, commandTag(
					"setpoint", createVirtual(ctx, telem.Float64T), cmdData[0],
				)))
				holder := MustSucceed(framerSvc.OpenWriter(ctx, framer.WriterConfig{
					Keys:        channel.Keys{cmdData[0].Key(), cmdIdx.Key()},
					Start:       telem.Now(),
					Authorities: []control.Authority{control.AuthorityAbsolute},
				}))
				host.sendCommand(commandMetric("setpoint", 0, 2.5))
				expectStatus(ctx, t, status.VariantWarning, "unauthorized")
				Expect(stored(ctx, cmdData[0]).Len()).To(BeZero())
				Expect(holder.Close()).To(Succeed())
				host.sendCommand(commandMetric("setpoint", 0, 3.5))
				Eventually(func() int64 { return stored(ctx, cmdData[0]).Len() }).
					Should(BeEquivalentTo(1))
				Expect(stored(ctx, cmdData[0])).
					To(telem.MatchSeriesData(telem.NewSeriesV(3.5)))
				expectStatus(ctx, t, status.VariantSuccess, "")
			},
		)
	})

	Describe("Session", func() {
		It("Should publish an NDEATH with the bdSeq of the session on a clean stop",
			func(ctx SpecContext) {
				ch := createVirtual(ctx, telem.Float64T)
				configured, _ := start(
					ctx,
					edgeConfig(dev, edgeTag("temp", ch, "double")),
				)
				Expect(configured.Stop(false)).To(Succeed())
				Eventually(host.deaths).Should(HaveLen(1))
				death := host.deaths()[0]
				Expect(death.topic).To(Equal("spBv1.0/plant/NDEATH/line1"))
				Expect(death.qos).To(Equal(byte(1)))
				Expect(death.retained).To(BeFalse())
				Expect(death.payload.Seq).To(BeNil())
				Expect(death.payload.Metrics).To(HaveLen(1))
				Expect(death.payload.Metrics[0].GetDatatype()).
					To(BeEquivalentTo(sparkplug.Int64))
				Expect(death.bdSeq()).To(Equal(host.births()[0].bdSeq()))
				Eventually(broker.clientCount).Should(BeZero())
				// A clean stop leaves no last will behind.
				Consistently(host.deaths, 200*time.Millisecond).Should(HaveLen(1))
			},
		)

		It("Should leave an NDEATH as the will and be born again after a drop",
			func(ctx SpecContext) {
				idx, data := createIndexed(ctx, telem.Float64T)
				_, t := start(ctx, edgeConfig(dev, edgeTag("temp", data[0], "double")))
				streamValue(ctx, idx, data[0], telem.NewSeriesV(1.5))
				broker.dropClients()
				Eventually(host.deaths).Should(HaveLen(1))
				death := host.deaths()[0]
				Expect(death.qos).To(Equal(byte(1)))
				Expect(death.retained).To(BeFalse())
				Expect(death.bdSeq()).To(Equal(uint64(0)))
				expectStatus(ctx, t, status.VariantWarning, "broker is not connected")
				Eventually(host.births, 10*time.Second).Should(HaveLen(2))
				birth := host.births()[1]
				Expect(birth.bdSeq()).To(Equal(uint64(1)))
				Expect(birth.payload.Seq).To(HaveValue(Equal(uint64(0))))
				// The new session is born with the last values.
				Expect(birth.metric("temp").GetValue()).To(Equal(
					&pb.Payload_Metric_DoubleValue{DoubleValue: 1.5},
				))
				expectStatus(ctx, t, status.VariantSuccess, "")
				stamp := streamValue(ctx, idx, data[0], telem.NewSeriesV(2.5))
				Expect(
					host.dataAt(stamp)[0].payload.Seq,
				).To(HaveValue(Equal(uint64(1))))
				Expect(broker.clientCount()).To(Equal(1))
			},
		)

		It("Should start while the broker is down and be born once it is up",
			func(ctx SpecContext) {
				ch := createVirtual(ctx, telem.Float64T)
				port := broker.port
				broker.stop()
				t := newTask(rackKey, mqtt.EdgeTaskType, edgeConfig(
					dev, edgeTag("temp", ch, "double"),
				))
				configured := configure(ctx, factory, t)
				Expect(configured.Exec(ctx, task.Command{Type: "start", Key: "start"})).
					To(Succeed())
				Expect(taskStatus(ctx, t).Details.Running).To(BeTrue())
				broker = startBroker(port)
				host = startHost(broker, "plant", "line1")
				Eventually(host.births, 10*time.Second).Should(HaveLen(1))
				Expect(host.births()[0].bdSeq()).To(Equal(uint64(0)))
				expectStatus(ctx, t, status.VariantSuccess, "")
			},
		)

		It("Should warn while the broker is down at start", func(ctx SpecContext) {
			ch := createVirtual(ctx, telem.Float64T)
			port := broker.port
			broker.stop()
			t := newTask(rackKey, mqtt.EdgeTaskType, edgeConfig(
				dev, edgeTag("temp", ch, "double"),
			))
			configured := configure(ctx, factory, t)
			Expect(configured.Exec(ctx, task.Command{Type: "start", Key: "start"})).
				To(Succeed())
			expectStatus(ctx, t, status.VariantWarning, "broker is not connected")
			broker = startBroker(port)
			host = startHost(broker, "plant", "line1")
			Eventually(host.births, 10*time.Second).Should(HaveLen(1))
			expectStatus(ctx, t, status.VariantSuccess, "")
		})

		It("Should start and stop many times without a leak", func(ctx SpecContext) {
			ch := createVirtual(ctx, telem.Float64T)
			t := newTask(rackKey, mqtt.EdgeTaskType, edgeConfig(
				dev, edgeTag("temp", ch, "double"),
			))
			configured := configure(ctx, factory, t)
			for i := 1; i <= 3; i++ {
				Expect(configured.Exec(ctx, task.Command{Type: "start", Key: "start"})).
					To(Succeed())
				Eventually(host.births).Should(HaveLen(i))
				Expect(configured.Exec(ctx, task.Command{Type: "stop", Key: "stop"})).
					To(Succeed())
				Eventually(host.deaths).Should(HaveLen(i))
				Eventually(broker.clientCount).Should(BeZero())
			}
			births, deaths := host.births(), host.deaths()
			for i := range 3 {
				Expect(deaths[i].bdSeq()).To(Equal(births[i].bdSeq()))
				Expect(births[i].payload.Seq).To(HaveValue(Equal(uint64(0))))
			}
			Expect(taskStatus(ctx, t).Details.Running).To(BeFalse())
		})
	})

	Describe("Configuration errors", func() {
		// valid returns the config of a task with one double tag on a new channel.
		valid := func(ctx context.Context) msgpack.EncodedJSON {
			GinkgoHelper()
			return edgeConfig(
				dev, edgeTag("temp", createVirtual(ctx, telem.Float64T), "double"),
			)
		}

		expectRejected := func(
			ctx context.Context,
			cfg msgpack.EncodedJSON,
			message string,
		) {
			GinkgoHelper()
			t := newTask(rackKey, mqtt.EdgeTaskType, cfg)
			Expect(factory.ConfigureTask(ctx, t, "configure")).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(message)),
			))
			stat := taskStatus(ctx, t)
			Expect(stat.Variant).To(Equal(status.VariantError))
			Expect(stat.Message).To(ContainSubstring(message))
			Expect(stat.Details.Cmd).To(Equal("configure"))
			Expect(broker.clientCount()).To(BeZero())
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
			Entry("no device", func(ctx context.Context) (msgpack.EncodedJSON, string) {
				cfg := valid(ctx)
				cfg["device"] = ""
				return cfg, "device: a broker is required"
			}),
			Entry("a device that does not exist",
				func(ctx context.Context) (msgpack.EncodedJSON, string) {
					cfg := valid(ctx)
					cfg["device"] = "missing"
					return cfg, "device: broker missing does not exist"
				},
			),
			Entry("a device that is not an MQTT broker",
				func(ctx context.Context) (msgpack.EncodedJSON, string) {
					other := device.Device{
						Key:        uuid.NewString(),
						Rack:       rackKey,
						Location:   "Dev1",
						Name:       "DAQ",
						Make:       "ni",
						Model:      "daq",
						Properties: msgpack.EncodedJSON{},
					}
					Expect(deviceSvc.NewWriter(nil).Create(ctx, &other)).To(Succeed())
					cfg := valid(ctx)
					cfg["device"] = other.Key
					return cfg, "device: DAQ is not an MQTT broker"
				},
			),
			Entry("no group", func(ctx context.Context) (msgpack.EncodedJSON, string) {
				cfg := valid(ctx)
				cfg["group"] = ""
				return cfg, "group: required"
			}),
			Entry(
				"no edge node",
				func(ctx context.Context) (msgpack.EncodedJSON, string) {
					cfg := valid(ctx)
					cfg["edge_node"] = ""
					return cfg, "edge_node: required"
				},
			),
			Entry("a group with a topic separator",
				func(ctx context.Context) (msgpack.EncodedJSON, string) {
					cfg := valid(ctx)
					cfg["group"] = "plant/a"
					return cfg, "group: plant/a must not hold /, +, or #"
				},
			),
			Entry("an edge node with a wildcard",
				func(ctx context.Context) (msgpack.EncodedJSON, string) {
					cfg := valid(ctx)
					cfg["edge_node"] = "line+1"
					return cfg, "edge_node: line+1 must not hold /, +, or #"
				},
			),
			Entry("no tags", func(context.Context) (msgpack.EncodedJSON, string) {
				return edgeConfig(dev), "tags: the task has no enabled tags"
			}),
			Entry("only disabled tags",
				func(ctx context.Context) (msgpack.EncodedJSON, string) {
					tag := edgeTag("temp", createVirtual(ctx, telem.Float64T), "double")
					tag["disabled"] = true
					return edgeConfig(dev, tag), "tags: the task has no enabled tags"
				},
			),
			Entry("a tag with no name",
				func(ctx context.Context) (msgpack.EncodedJSON, string) {
					ch := createVirtual(ctx, telem.Float64T)
					return edgeConfig(dev, edgeTag("", ch, "double")), "tag: required"
				},
			),
			Entry("a channel that does not exist",
				func(context.Context) (msgpack.EncodedJSON, string) {
					tag := edgeTag("temp", channel.Channel{}, "double")
					tag["channel"] = 999999
					return edgeConfig(
						dev,
						tag,
					), "tag temp: channel 999999 does not exist"
				},
			),
			Entry("a Sparkplug B data type it does not know",
				func(ctx context.Context) (msgpack.EncodedJSON, string) {
					ch := createVirtual(ctx, telem.Float64T)
					return edgeConfig(dev, edgeTag("temp", ch, "float128")),
						"tag temp: unknown Sparkplug B data type float128"
				},
			),
			Entry("a string channel as double",
				func(ctx context.Context) (msgpack.EncodedJSON, string) {
					ch := createVirtual(ctx, telem.StringT)
					return edgeConfig(dev, edgeTag("temp", ch, "double")),
						"tag temp: channel " + ch.Name +
							" of type string cannot be sent as Double"
				},
			),
			Entry("a float64 channel as string",
				func(ctx context.Context) (msgpack.EncodedJSON, string) {
					ch := createVirtual(ctx, telem.Float64T)
					return edgeConfig(dev, edgeTag("temp", ch, "string")),
						"tag temp: channel " + ch.Name +
							" of type float64 cannot be sent as String"
				},
			),
			Entry("a timestamp channel as int64",
				func(ctx context.Context) (msgpack.EncodedJSON, string) {
					ch := createVirtual(ctx, telem.TimestampT)
					return edgeConfig(dev, edgeTag("temp", ch, "int64")),
						"tag temp: channel " + ch.Name +
							" of type timestamp cannot be sent as Int64"
				},
			),
			Entry("a tag name that appears two times",
				func(ctx context.Context) (msgpack.EncodedJSON, string) {
					ch := createVirtual(ctx, telem.Float64T)
					again := edgeTag("temp", ch, "double")
					again["key"] = "again"
					return edgeConfig(dev, edgeTag("temp", ch, "double"), again),
						"tags: tag temp appears more than once"
				},
			),
			Entry("a command channel that does not exist",
				func(ctx context.Context) (msgpack.EncodedJSON, string) {
					tag := edgeTag("temp", createVirtual(ctx, telem.Float64T), "double")
					tag["command_channel"] = 999999
					return edgeConfig(dev, tag),
						"command channel: tag plant/line1/temp: " +
							"channel 999999 does not exist"
				},
			),
			Entry("a command channel that cannot take the values",
				func(ctx context.Context) (msgpack.EncodedJSON, string) {
					idx, _ := createIndexed(ctx)
					tag := edgeTag("temp", createVirtual(ctx, telem.Float64T), "double")
					tag["command_channel"] = idx.Key()
					return edgeConfig(dev, tag),
						"command channel: tag plant/line1/temp: channel " + idx.Name +
							" cannot take the values of a tag"
				},
			),
		)
	})
})
