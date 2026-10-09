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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/framer/iterator"
	"github.com/synnaxlabs/synnax/pkg/service/http"
	"github.com/synnaxlabs/synnax/pkg/service/kafka"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
	"github.com/twmb/franz-go/pkg/kgo"
)

var _ = Describe("Read task", func() {
	var (
		f        driver.Factory
		producer *kgo.Client
		unixNs   = http.TimeFormatUnixNs
	)
	BeforeEach(func() {
		f = newFactory()
		producer = newKafkaClient(kgo.AllowAutoTopicCreation())
	})

	// newTopic creates a fresh single-partition topic.
	newTopic := func() string {
		GinkgoHelper()
		topic := uniqueTopic()
		Expect(cluster.CreateTopic(topic, 1, nil)).To(Succeed())
		return topic
	}

	// configure configures a read task, stopping it when the spec ends.
	configure := func(
		ctx context.Context,
		cfg kafka.ReadConfig,
	) (driver.Task, task.Task) {
		GinkgoHelper()
		if cfg.StartOffset == "" {
			cfg.StartOffset = kafka.StartOffsetEarliest
		}
		t := newTask(kafka.ReadTaskType, cfg)
		dt := MustSucceed(f.ConfigureTask(ctx, t, "cmd-1"))
		DeferCleanup(func() { Expect(dt.Stop(false)).To(Succeed()) })
		return dt, t
	}

	start := func(ctx context.Context, dt driver.Task, t task.Task) {
		GinkgoHelper()
		Expect(dt.Exec(ctx, task.Command{Task: t.Key, Type: "start", Key: "cmd-2"})).
			To(Succeed())
		stat := taskStatus(ctx, t)
		Expect(stat.Variant).To(Equal(status.VariantSuccess))
		Expect(stat.Details.Running).To(BeTrue())
	}

	// stored returns every sample persisted in key.
	stored := func(ctx context.Context, key channel.Key) telem.Series {
		GinkgoHelper()
		it := MustSucceed(framerSvc.OpenIterator(ctx, framer.IteratorConfig{
			Keys:   channel.Keys{key},
			Bounds: telem.TimeRangeMax,
		}))
		defer func() { Expect(it.Close()).To(Succeed()) }()
		var s telem.Series
		if !it.SeekFirst() {
			return s
		}
		for it.Next(iterator.AutoSpan) {
			for _, ser := range it.Value().Get(key).Series {
				s.DataType = ser.DataType
				s.Data = append(s.Data, ser.Data...)
			}
		}
		Expect(it.Error()).To(Succeed())
		return s
	}

	expectStored := func(ctx context.Context, key channel.Key, n int64) telem.Series {
		GinkgoHelper()
		Eventually(func() int64 { return stored(ctx, key).Len() }).
			WithTimeout(15 * time.Second).Should(Equal(n))
		return stored(ctx, key)
	}

	timestamps := func(s telem.Series) []telem.TimeStamp {
		out := make([]telem.TimeStamp, s.Len())
		for i := range out {
			out[i] = s.ValueAt[telem.TimeStamp](i)
		}
		return out
	}

	floats := func(s telem.Series) []float64 {
		out := make([]float64, s.Len())
		for i := range out {
			out[i] = s.ValueAt[float64](i)
		}
		return out
	}

	stampedFields := func(index, data channel.Channel) []kafka.ReadField {
		return []kafka.ReadField{
			{
				Key:        "ts",
				Channel:    index.Key(),
				Pointer:    "/ts",
				DataType:   telem.TimestampT,
				TimeFormat: &unixNs,
			},
			{Key: "v", Channel: data.Key(), Pointer: "/v", DataType: telem.Float64T},
		}
	}

	It("Should write record fields stamped by the record", func(ctx SpecContext) {
		index, data := createIndexed(ctx, telem.Float64T)
		dev := createDevice(ctx, clusterProperties())
		topic := newTopic()
		dt, t := configure(ctx, kafka.ReadConfig{
			Device: dev.Key,
			Topic:  topic,
			Fields: stampedFields(index, data),
		})
		start(ctx, dt, t)
		produceJSON(ctx, producer, topic, "", map[string]any{"ts": 10e9, "v": 1.5})
		produceJSON(ctx, producer, topic, "", map[string]any{"ts": 11e9, "v": 2.5})
		Expect(floats(expectStored(ctx, data.Key(), 2))).To(Equal([]float64{1.5, 2.5}))
		Expect(timestamps(stored(ctx, index.Key()))).
			To(Equal([]telem.TimeStamp{10 * telem.SecondTS, 11 * telem.SecondTS}))
		Expect(dt.Exec(ctx, task.Command{Task: t.Key, Type: "stop", Key: "cmd-3"})).
			To(Succeed())
		stat := taskStatus(ctx, t)
		Expect(stat.Variant).To(Equal(status.VariantSuccess))
		Expect(stat.Details.Running).To(BeFalse())
		Expect(stat.Details.Cmd).To(Equal("cmd-3"))
	})

	It("Should stamp a group without a timestamp field with the receive time",
		func(ctx SpecContext) {
			index, data := createIndexed(ctx, telem.Float64T)
			dev := createDevice(ctx, clusterProperties())
			topic := newTopic()
			dt, t := configure(ctx, kafka.ReadConfig{
				Device: dev.Key,
				Topic:  topic,
				Fields: []kafka.ReadField{{
					Key:      "v",
					Channel:  data.Key(),
					Pointer:  "/v",
					DataType: telem.Float64T,
				}},
			})
			before := telem.Now()
			start(ctx, dt, t)
			produceJSON(ctx, producer, topic, "", map[string]any{"v": 1.0})
			produceJSON(ctx, producer, topic, "", map[string]any{"v": 2.0})
			Expect(floats(expectStored(ctx, data.Key(), 2))).To(Equal([]float64{1, 2}))
			stamps := timestamps(stored(ctx, index.Key()))
			Expect(stamps[0]).To(BeNumerically(">", before))
			Expect(stamps[1]).To(BeNumerically(">", stamps[0]))
		},
	)

	It("Should skip a record lacking a field with an error status",
		func(ctx SpecContext) {
			index, data := createIndexed(ctx, telem.Float64T)
			dev := createDevice(ctx, clusterProperties())
			topic := newTopic()
			dt, t := configure(ctx, kafka.ReadConfig{
				Device: dev.Key,
				Topic:  topic,
				Fields: stampedFields(index, data),
			})
			start(ctx, dt, t)
			produceJSON(ctx, producer, topic, "", map[string]any{"ts": 10e9})
			produceJSON(ctx, producer, topic, "", map[string]any{"ts": 11e9, "v": 2.5})
			Expect(floats(expectStored(ctx, data.Key(), 1))).To(Equal([]float64{2.5}))
			stat := taskStatus(ctx, t)
			Expect(stat.Variant).To(Equal(status.VariantError))
			Expect(stat.Message).To(Equal("Record has no value at /v"))
			Expect(stat.Details.Running).To(BeTrue())
		},
	)

	It("Should skip a record stamped at or before the last written timestamp",
		func(ctx SpecContext) {
			index, data := createIndexed(ctx, telem.Float64T)
			dev := createDevice(ctx, clusterProperties())
			topic := newTopic()
			dt, t := configure(ctx, kafka.ReadConfig{
				Device: dev.Key,
				Topic:  topic,
				Fields: stampedFields(index, data),
			})
			start(ctx, dt, t)
			produceJSON(ctx, producer, topic, "", map[string]any{"ts": 20e9, "v": 1.0})
			expectStored(ctx, data.Key(), 1)
			produceJSON(ctx, producer, topic, "", map[string]any{"ts": 15e9, "v": 2.0})
			Eventually(func() string { return taskStatus(ctx, t).Message }).
				WithTimeout(15 * time.Second).
				Should(ContainSubstring("is not after the last written"))
			produceJSON(ctx, producer, topic, "", map[string]any{"ts": 25e9, "v": 3.0})
			Expect(floats(expectStored(ctx, data.Key(), 2))).To(Equal([]float64{1, 3}))
		},
	)

	It("Should silently skip the replay of stored records on a fresh group",
		func(ctx SpecContext) {
			index, data := createIndexed(ctx, telem.Float64T)
			dev := createDevice(ctx, clusterProperties())
			topic := newTopic()
			cfg := kafka.ReadConfig{
				Device: dev.Key,
				Topic:  topic,
				Fields: stampedFields(index, data),
			}
			dt, t := configure(ctx, cfg)
			start(ctx, dt, t)
			produceJSON(ctx, producer, topic, "", map[string]any{"ts": 10e9, "v": 1.0})
			produceJSON(ctx, producer, topic, "", map[string]any{"ts": 11e9, "v": 2.0})
			expectStored(ctx, data.Key(), 2)
			Expect(dt.Stop(true)).To(Succeed())

			cfg.Group = "fresh-" + t.Key.String()
			dt2, t2 := configure(ctx, cfg)
			start(ctx, dt2, t2)
			produceJSON(ctx, producer, topic, "", map[string]any{"ts": 12e9, "v": 3.0})
			Expect(floats(expectStored(ctx, data.Key(), 3))).
				To(Equal([]float64{1, 2, 3}))
			Expect(taskStatus(ctx, t2).Variant).To(Equal(status.VariantSuccess))
		},
	)

	It("Should resume from the committed offset after a restart",
		func(ctx SpecContext) {
			index, data := createIndexed(ctx, telem.Float64T)
			dev := createDevice(ctx, clusterProperties())
			topic := newTopic()
			dt, t := configure(ctx, kafka.ReadConfig{
				Device: dev.Key,
				Topic:  topic,
				Fields: stampedFields(index, data),
			})
			start(ctx, dt, t)
			produceJSON(ctx, producer, topic, "", map[string]any{"ts": 10e9, "v": 1.0})
			expectStored(ctx, data.Key(), 1)
			Expect(dt.Stop(true)).To(Succeed())
			produceJSON(ctx, producer, topic, "", map[string]any{"ts": 11e9, "v": 2.0})
			start(ctx, dt, t)
			Expect(floats(expectStored(ctx, data.Key(), 2))).To(Equal([]float64{1, 2}))
		},
	)

	It("Should route keyed records to their groups", func(ctx SpecContext) {
		indexA, dataA := createIndexed(ctx, telem.Float64T)
		indexB, dataB := createIndexed(ctx, telem.Float64T)
		dev := createDevice(ctx, clusterProperties())
		topic := newTopic()
		fields := stampedFields(indexA, dataA)
		for i := range fields {
			fields[i].RecordKey = "a"
		}
		for _, fld := range stampedFields(indexB, dataB) {
			fld.RecordKey = "b"
			fields = append(fields, fld)
		}
		dt, t := configure(ctx, kafka.ReadConfig{
			Device: dev.Key,
			Topic:  topic,
			Fields: fields,
		})
		start(ctx, dt, t)
		produceJSON(ctx, producer, topic, "a", map[string]any{"ts": 10e9, "v": 1.0})
		produceJSON(ctx, producer, topic, "b", map[string]any{"ts": 10e9, "v": 2.0})
		produceJSON(ctx, producer, topic, "c", map[string]any{"ts": 10e9, "v": 3.0})
		Expect(floats(expectStored(ctx, dataA.Key(), 1))).To(Equal([]float64{1}))
		Expect(floats(expectStored(ctx, dataB.Key(), 1))).To(Equal([]float64{2}))
		Expect(taskStatus(ctx, t).Variant).To(Equal(status.VariantSuccess))
	})

	It("Should answer start with an error when a none start offset has no commit",
		func(ctx SpecContext) {
			index, data := createIndexed(ctx, telem.Float64T)
			dev := createDevice(ctx, clusterProperties())
			t := newTask(kafka.ReadTaskType, kafka.ReadConfig{
				Device:      dev.Key,
				Topic:       newTopic(),
				StartOffset: kafka.StartOffsetNone,
				Fields:      stampedFields(index, data),
			})
			dt := MustSucceed(f.ConfigureTask(ctx, t, "cmd-1"))
			DeferCleanup(func() { Expect(dt.Stop(false)).To(Succeed()) })
			Expect(dt.Exec(ctx, task.Command{
				Task: t.Key, Type: "start", Key: "cmd-2",
			})).To(MatchError(ContainSubstring("no committed offset")))
			stat := taskStatus(ctx, t)
			Expect(stat.Variant).To(Equal(status.VariantError))
			Expect(stat.Details.Running).To(BeFalse())
		},
	)

	DescribeTable("Should reject an invalid config",
		func(
			ctx SpecContext,
			build func(
				ctx context.Context,
				index, data channel.Channel,
			) []kafka.ReadField,
			msg string,
		) {
			index, data := createIndexed(ctx, telem.Float64T)
			dev := createDevice(ctx, clusterProperties())
			cfg := kafka.ReadConfig{
				Device: dev.Key,
				Topic:  "t",
				Fields: build(ctx, index, data),
			}
			Expect(f.ConfigureTask(ctx, newTask(kafka.ReadTaskType, cfg), "cmd-1")).
				Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(msg)),
			))
		},
		Entry("no enabled field",
			func(_ context.Context, _, data channel.Channel) []kafka.ReadField {
				return []kafka.ReadField{
					{Key: "v", Channel: data.Key(), Disabled: true},
				}
			}, "at least one"),
		Entry("empty pointer",
			func(_ context.Context, _, data channel.Channel) []kafka.ReadField {
				return []kafka.ReadField{{Key: "v", Channel: data.Key()}}
			}, "pointer"),
		Entry("data type mismatch",
			func(_ context.Context, _, data channel.Channel) []kafka.ReadField {
				return []kafka.ReadField{{
					Key:      "v",
					Channel:  data.Key(),
					Pointer:  "/v",
					DataType: telem.Int32T,
				}}
			}, "does not match"),
		Entry("timestamp without time format",
			func(_ context.Context, index, _ channel.Channel) []kafka.ReadField {
				return []kafka.ReadField{{
					Key: "ts", Channel: index.Key(), Pointer: "/ts",
					DataType: telem.TimestampT,
				}}
			}, "time_format"),
		Entry("channels on different indexes",
			func(ctx context.Context, _, data channel.Channel) []kafka.ReadField {
				_, other := createIndexed(ctx, telem.Float64T)
				return []kafka.ReadField{
					{
						Key:      "v",
						Channel:  data.Key(),
						Pointer:  "/v",
						DataType: telem.Float64T,
					},
					{
						Key:      "w",
						Channel:  other.Key(),
						Pointer:  "/w",
						DataType: telem.Float64T,
					},
				}
			}, "share one index"),
	)
})
