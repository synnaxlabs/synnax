// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package kafka_test

import (
	"context"
	"time"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/synnax/pkg/service/kafka"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
	"github.com/twmb/franz-go/pkg/kfake"
	"github.com/twmb/franz-go/pkg/kmsg"
)

var _ = Describe("Write task with targets", func() {
	var (
		f        driver.Factory
		setpoint icd.Message
		ic       icd.ICD
	)
	BeforeEach(func(ctx SpecContext) {
		f = newFactory()
		setpoint = jsonMessage(
			"Setpoint", nil, jsonField("value", "/value", icd.FloatJSONField{}),
		)
		ic = createICD(ctx, setpoint)
	})

	// start configures and starts a task of cfg, and stops it when the spec ends.
	start := func(ctx context.Context, cfg kafka.WriteConfig) task.Task {
		GinkgoHelper()
		t := newTask(ctx, kafka.WriteTaskType, cfg)
		dt := MustSucceed(f.ConfigureTask(ctx, t, "cmd-1"))
		DeferCleanup(func() { Expect(dt.Stop(false)).To(Succeed()) })
		Expect(dt.Exec(ctx, task.Command{Task: t.Key, Type: "start", Key: "cmd-2"})).
			To(Succeed())
		stat := taskStatus(ctx, t)
		Expect(stat.Variant).To(Equal(status.VariantSuccess))
		Expect(stat.Details.Running).To(BeTrue())
		return t
	}

	// createVirtual creates a virtual float64 channel.
	createVirtual := func(ctx context.Context) channel.Channel {
		GinkgoHelper()
		ch := channel.Channel{
			Name:     "cmd_" + uuid.New().String()[:8],
			DataType: telem.Float64T,
			Virtual:  true,
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channelSvc.NewWriter(tx).Create(ctx, &ch)
		})).To(Succeed())
		return ch
	}

	// write writes s to the channel with key alone, in a writer that starts at start.
	write := func(
		ctx context.Context,
		key channel.Key,
		start telem.TimeStamp,
		s telem.Series,
	) {
		GinkgoHelper()
		w := MustSucceed(framerSvc.OpenWriter(ctx, framer.WriterConfig{
			Keys:  channel.Keys{key},
			Start: start,
		}))
		Expect(MustSucceed(w.Write(frame.NewUnary(key, s)))).To(BeTrue())
		Expect(w.Close()).To(Succeed())
	}

	It("Should produce the document of each command with the time of its index",
		func(ctx SpecContext) {
			index, data := createIndexed(ctx, telem.Float64T)
			dev := createDevice(ctx, clusterProperties())
			topic := uniqueTopic()
			wt := writeTarget(topic, ic, setpoint, command(setpoint, 0, data))
			wt.RecordKey = "pump"
			start(ctx, kafka.WriteConfig{
				Device:  dev.Key,
				Targets: []kafka.WriteTarget{wt},
			})
			stamp := telem.TimeStamp(10 * telem.Second)
			writeFrame(ctx, index, data,
				[]telem.TimeStamp{stamp, stamp + telem.SecondTS},
				telem.NewSeriesV(1.5, 2.5),
			)
			recs := consume(ctx, topic, 2)
			Expect(recs).To(HaveLen(2))
			Expect(decode(recs[0])).To(Equal(map[string]any{"value": 1.5}))
			Expect(decode(recs[1])).To(Equal(map[string]any{"value": 2.5}))
			Expect(string(recs[0].Key)).To(Equal("pump"))
			// Kafka record timestamps carry millisecond precision.
			Expect(recs[0].Timestamp.UnixMilli()).To(Equal(int64(10_000)))
			Expect(recs[1].Timestamp.UnixMilli()).To(Equal(int64(11_000)))
		},
	)

	It(
		"Should give the record of a virtual channel the time it was produced and no key",
		func(ctx SpecContext) {
			ch := createVirtual(ctx)
			dev := createDevice(ctx, clusterProperties())
			topic := uniqueTopic()
			start(ctx, kafka.WriteConfig{
				Device: dev.Key,
				Targets: []kafka.WriteTarget{
					writeTarget(topic, ic, setpoint, command(setpoint, 0, ch)),
				},
			})
			before := time.Now().Truncate(time.Millisecond)
			write(ctx, ch.Key(), telem.Now(), telem.NewSeriesV(3.0))
			recs := consume(ctx, topic, 1)
			Expect(decode(recs[0])).To(Equal(map[string]any{"value": 3.0}))
			Expect(recs[0].Key).To(BeNil())
			Expect(recs[0].Timestamp).To(BeTemporally(">=", before))
			Expect(recs[0].Timestamp).To(BeTemporally("<=", time.Now()))
		},
	)

	It("Should skip a disabled target", func(ctx SpecContext) {
		ch := createVirtual(ctx)
		dev := createDevice(ctx, clusterProperties())
		topic := uniqueTopic()
		disabled := writeTarget(uniqueTopic(), icd.ICD{Key: uuid.New()}, setpoint)
		disabled.Disabled = true
		start(ctx, kafka.WriteConfig{
			Device: dev.Key,
			Targets: []kafka.WriteTarget{
				disabled,
				writeTarget(topic, ic, setpoint, command(setpoint, 0, ch)),
			},
		})
		write(ctx, ch.Key(), telem.Now(), telem.NewSeriesV(4.0))
		Expect(decode(consume(ctx, topic, 1)[0])).
			To(Equal(map[string]any{"value": 4.0}))
	})

	It("Should refuse a task with no enabled target", func(ctx SpecContext) {
		ch := createVirtual(ctx)
		dev := createDevice(ctx, clusterProperties())
		wt := writeTarget(uniqueTopic(), ic, setpoint, command(setpoint, 0, ch))
		wt.Disabled = true
		t := newTask(ctx, kafka.WriteTaskType, kafka.WriteConfig{
			Device:  dev.Key,
			Targets: []kafka.WriteTarget{wt},
		})
		Expect(f.ConfigureTask(ctx, t, "cmd-1")).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("targets: the task has no enabled targets")),
		))
	})

	It("Should refuse an enabled target with no topic", func(ctx SpecContext) {
		ch := createVirtual(ctx)
		dev := createDevice(ctx, clusterProperties())
		t := newTask(ctx, kafka.WriteTaskType, kafka.WriteConfig{
			Device: dev.Key,
			Targets: []kafka.WriteTarget{
				writeTarget(uniqueTopic(), ic, setpoint, command(setpoint, 0, ch)),
				writeTarget("", ic, setpoint, command(setpoint, 0, ch)),
			},
		})
		Expect(f.ConfigureTask(ctx, t, "cmd-1")).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("targets.1.topic: required")),
		))
	})

	It("Should drop a command that arrives without its index", func(ctx SpecContext) {
		index, data := createIndexed(ctx, telem.Float64T)
		dev := createDevice(ctx, clusterProperties())
		topic := uniqueTopic()
		t := start(ctx, kafka.WriteConfig{
			Device: dev.Key,
			Targets: []kafka.WriteTarget{
				writeTarget(topic, ic, setpoint, command(setpoint, 0, data)),
			},
		})
		stamp := telem.TimeStamp(20 * telem.Second)
		write(ctx, index.Key(), stamp, telem.NewSeries([]telem.TimeStamp{stamp}))
		write(ctx, data.Key(), stamp, telem.NewSeriesV(1.5))
		Eventually(func(g Gomega) {
			stat := taskStatus(ctx, t)
			g.Expect(stat.Variant).To(Equal(status.VariantWarning))
			g.Expect(stat.Message).To(Equal("Skipped 1 of 1 command"))
			g.Expect(stat.Description).To(Equal(
				"Command for target " + topic + ": the command arrived without its index",
			))
		}).Should(Succeed())
	})

	It("Should warn and keep running when the cluster does not take a record",
		func(ctx SpecContext) {
			lost := MustSucceed(kfake.NewCluster(
				kfake.NumBrokers(1),
				kfake.AllowAutoTopicCreation(),
			))
			DeferCleanup(lost.Close)
			lost.ControlKey(
				kmsg.Produce.Int16(),
				func(kmsg.Request) (kmsg.Response, error, bool) {
					lost.KeepControl()
					return nil, errors.New("cluster lost"), true
				},
			)
			f = newFactory(kafka.FactoryConfig{PingTimeout: 200 * telem.Millisecond})
			ch := createVirtual(ctx)
			dev := createDevice(ctx, kafka.Properties{Brokers: lost.ListenAddrs()})
			topic := uniqueTopic()
			t := start(ctx, kafka.WriteConfig{
				Device: dev.Key,
				Targets: []kafka.WriteTarget{
					writeTarget(topic, ic, setpoint, command(setpoint, 0, ch)),
				},
			})
			write(ctx, ch.Key(), telem.Now(), telem.NewSeriesV(5.0))
			Eventually(func(g Gomega) {
				stat := taskStatus(ctx, t)
				g.Expect(stat.Variant).To(Equal(status.VariantWarning))
				g.Expect(stat.Message).To(Equal("Failed to write data"))
				g.Expect(stat.Description).
					To(HavePrefix("producing to topic " + topic + ": "))
				g.Expect(stat.Details.Running).To(BeTrue())
			}).WithTimeout(15 * time.Second).Should(Succeed())
		},
	)
})
