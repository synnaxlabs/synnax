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
	"math"
	"time"
	"uuid"

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
	"github.com/twmb/franz-go/pkg/kerr"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/kmsg"
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

	// single wraps fields into one record that reads every key, indexed by the field
	// keyed "ts" when there is one.
	single := func(fields []kafka.ReadField) []kafka.ReadRecord {
		rec := kafka.ReadRecord{Key: "r", Fields: fields}
		for _, fld := range fields {
			if fld.Key == "ts" {
				rec.Index = fld.Key
			}
		}
		return []kafka.ReadRecord{rec}
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
			Device:  dev.Key,
			Topic:   topic,
			Records: single(stampedFields(index, data)),
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
				Records: single([]kafka.ReadField{{
					Key:      "v",
					Channel:  data.Key(),
					Pointer:  "/v",
					DataType: telem.Float64T,
				}}),
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
				Device:  dev.Key,
				Topic:   topic,
				Records: single(stampedFields(index, data)),
			})
			start(ctx, dt, t)
			produceJSON(ctx, producer, topic, "", map[string]any{"ts": 10e9})
			produceJSON(ctx, producer, topic, "", map[string]any{"ts": 11e9, "v": 2.5})
			Expect(floats(expectStored(ctx, data.Key(), 1))).To(Equal([]float64{2.5}))
			stat := taskStatus(ctx, t)
			Expect(stat.Variant).To(Equal(status.VariantError))
			Expect(stat.Message).To(Equal("Skipped record"))
			Expect(stat.Description).To(Equal("Record has no value at /v"))
			Expect(stat.Details.Running).To(BeTrue())
		},
	)

	It("Should skip a record stamped at or before the last written timestamp",
		func(ctx SpecContext) {
			index, data := createIndexed(ctx, telem.Float64T)
			dev := createDevice(ctx, clusterProperties())
			topic := newTopic()
			dt, t := configure(ctx, kafka.ReadConfig{
				Device:  dev.Key,
				Topic:   topic,
				Records: single(stampedFields(index, data)),
			})
			start(ctx, dt, t)
			produceJSON(ctx, producer, topic, "", map[string]any{"ts": 20e9, "v": 1.0})
			expectStored(ctx, data.Key(), 1)
			produceJSON(ctx, producer, topic, "", map[string]any{"ts": 15e9, "v": 2.0})
			Eventually(func() string { return taskStatus(ctx, t).Description }).
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
				Device:  dev.Key,
				Topic:   topic,
				Records: single(stampedFields(index, data)),
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
				Device:  dev.Key,
				Topic:   topic,
				Records: single(stampedFields(index, data)),
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

	It("Should consume again when started after a failed fetch",
		func(ctx SpecContext) {
			index, data := createIndexed(ctx, telem.Float64T)
			dev := createDevice(ctx, clusterProperties())
			topic := newTopic()
			dt, t := configure(ctx, kafka.ReadConfig{
				Device:  dev.Key,
				Topic:   topic,
				Records: single(stampedFields(index, data)),
			})
			start(ctx, dt, t)
			cluster.ControlKey(
				int16(kmsg.Fetch),
				func(req kmsg.Request) (kmsg.Response, error, bool) {
					res := req.ResponseKind().(*kmsg.FetchResponse)
					for _, rt := range req.(*kmsg.FetchRequest).Topics {
						st := kmsg.NewFetchResponseTopic()
						st.Topic, st.TopicID = rt.Topic, rt.TopicID
						for _, rp := range rt.Partitions {
							sp := kmsg.NewFetchResponseTopicPartition()
							sp.Partition = rp.Partition
							sp.ErrorCode = kerr.TopicAuthorizationFailed.Code
							st.Partitions = append(st.Partitions, sp)
						}
						res.Topics = append(res.Topics, st)
					}
					return res, nil, true
				},
			)
			Eventually(func() string { return taskStatus(ctx, t).Message }).
				Should(Equal("Failed to fetch records"))
			start(ctx, dt, t)
			produceJSON(ctx, producer, topic, "", map[string]any{"ts": 10e9, "v": 1.0})
			Expect(floats(expectStored(ctx, data.Key(), 1))).To(Equal([]float64{1}))
		},
	)

	It("Should route keyed records to their groups", func(ctx SpecContext) {
		indexA, dataA := createIndexed(ctx, telem.Float64T)
		indexB, dataB := createIndexed(ctx, telem.Float64T)
		dev := createDevice(ctx, clusterProperties())
		topic := newTopic()
		dt, t := configure(ctx, kafka.ReadConfig{
			Device: dev.Key,
			Topic:  topic,
			Records: []kafka.ReadRecord{
				{
					Key:       "a",
					RecordKey: "a",
					Index:     "ts",
					Fields:    stampedFields(indexA, dataA),
				},
				{
					Key:       "b",
					RecordKey: "b",
					Index:     "ts",
					Fields:    stampedFields(indexB, dataB),
				},
			},
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
				Records:     single(stampedFields(index, data)),
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

	It("Should start with a none start offset once the group has a commit",
		func(ctx SpecContext) {
			index, data := createIndexed(ctx, telem.Float64T)
			dev := createDevice(ctx, clusterProperties())
			topic := newTopic()
			cfg := kafka.ReadConfig{
				Device:  dev.Key,
				Topic:   topic,
				Group:   "group-" + topic,
				Records: single(stampedFields(index, data)),
			}
			dt, t := configure(ctx, cfg)
			start(ctx, dt, t)
			produceJSON(ctx, producer, topic, "", map[string]any{"ts": 10e9, "v": 1.0})
			expectStored(ctx, data.Key(), 1)
			Expect(dt.Stop(true)).To(Succeed())

			cfg.StartOffset = kafka.StartOffsetNone
			dt2, t2 := configure(ctx, cfg)
			start(ctx, dt2, t2)
			produceJSON(ctx, producer, topic, "", map[string]any{"ts": 11e9, "v": 2.0})
			Expect(floats(expectStored(ctx, data.Key(), 2))).To(Equal([]float64{1, 2}))
		},
	)

	It("Should skip a record that is not JSON", func(ctx SpecContext) {
		index, data := createIndexed(ctx, telem.Float64T)
		dev := createDevice(ctx, clusterProperties())
		topic := newTopic()
		dt, t := configure(ctx, kafka.ReadConfig{
			Device:  dev.Key,
			Topic:   topic,
			Records: single(stampedFields(index, data)),
		})
		start(ctx, dt, t)
		rec := &kgo.Record{Topic: topic, Value: []byte("{")}
		Expect(producer.ProduceSync(ctx, rec).FirstErr()).To(Succeed())
		Eventually(func() string { return taskStatus(ctx, t).Description }).
			WithTimeout(15 * time.Second).
			Should(ContainSubstring("Record is not JSON"))
		produceJSON(ctx, producer, topic, "", map[string]any{"ts": 10e9, "v": 1.0})
		Expect(floats(expectStored(ctx, data.Key(), 1))).To(Equal([]float64{1}))
	})

	It("Should resolve array indexes and escaped tokens in pointers",
		func(ctx SpecContext) {
			index, data := createIndexed(ctx, telem.Float64T)
			dev := createDevice(ctx, clusterProperties())
			topic := newTopic()
			fields := stampedFields(index, data)
			fields[0].Pointer = "/t~1s"
			fields[1].Pointer = "/vals/1"
			dt, t := configure(ctx, kafka.ReadConfig{
				Device:  dev.Key,
				Topic:   topic,
				Records: single(fields),
			})
			start(ctx, dt, t)
			produceJSON(ctx, producer, topic, "", map[string]any{
				"t/s": 10e9, "vals": []any{1.0, 2.5},
			})
			Expect(floats(expectStored(ctx, data.Key(), 1))).To(Equal([]float64{2.5}))
			Expect(timestamps(stored(ctx, index.Key()))).
				To(Equal([]telem.TimeStamp{10 * telem.SecondTS}))
		},
	)

	It("Should skip a record when a pointer lacks its leading slash",
		func(ctx SpecContext) {
			index, data := createIndexed(ctx, telem.Float64T)
			dev := createDevice(ctx, clusterProperties())
			topic := newTopic()
			fields := stampedFields(index, data)
			fields[1].Pointer = "v"
			dt, t := configure(ctx, kafka.ReadConfig{
				Device:  dev.Key,
				Topic:   topic,
				Records: single(fields),
			})
			start(ctx, dt, t)
			produceJSON(ctx, producer, topic, "", map[string]any{"ts": 10e9, "v": 1.0})
			Eventually(func() string { return taskStatus(ctx, t).Description }).
				WithTimeout(15 * time.Second).
				Should(ContainSubstring(`pointer "v" must start with '/'`))
		},
	)

	onOff := []http.EnumEntry{{Label: "OFF", Value: 0}, {Label: "ON", Value: 1}}
	readUUID := uuid.MustParse("6f1c2a4e-8d3b-4f5a-9c7e-2b1d0e3f4a5b")

	// valueFields reads the timestamp at /ts in format and the value at /v.
	valueFields := func(
		index, data channel.Channel,
		dt telem.DataType,
		format *http.TimeFormat,
		enums []http.EnumEntry,
	) []kafka.ReadField {
		return []kafka.ReadField{
			{
				Key:        "ts",
				Channel:    index.Key(),
				Pointer:    "/ts",
				DataType:   telem.TimestampT,
				TimeFormat: format,
			},
			{
				Key:        "v",
				Channel:    data.Key(),
				Pointer:    "/v",
				DataType:   dt,
				EnumValues: enums,
			},
		}
	}

	DescribeTable("Should convert the record value into the channel's data type",
		func(
			ctx SpecContext,
			dataType telem.DataType,
			format http.TimeFormat,
			enums []http.EnumEntry,
			doc map[string]any,
			expected telem.Series,
		) {
			index, data := createIndexed(ctx, dataType)
			dev := createDevice(ctx, clusterProperties())
			topic := newTopic()
			dt, t := configure(ctx, kafka.ReadConfig{
				Device:  dev.Key,
				Topic:   topic,
				Records: single(valueFields(index, data, dataType, &format, enums)),
			})
			start(ctx, dt, t)
			produceJSON(ctx, producer, topic, "", doc)
			Expect(expectStored(ctx, data.Key(), 1).Data).To(Equal(expected.Data))
			Expect(timestamps(stored(ctx, index.Key()))).
				To(Equal([]telem.TimeStamp{10 * telem.SecondTS}))
		},
		Entry("a unix second timestamp",
			telem.Float64T, http.TimeFormatUnixSec, nil,
			map[string]any{"ts": 10, "v": 1.5}, telem.NewSeriesV[float64](1.5)),
		Entry("a unix millisecond timestamp",
			telem.Float64T, http.TimeFormatUnixMs, nil,
			map[string]any{"ts": 10_000, "v": 1.5}, telem.NewSeriesV[float64](1.5)),
		Entry("a unix microsecond timestamp",
			telem.Float64T, http.TimeFormatUnixUs, nil,
			map[string]any{"ts": 10_000_000, "v": 1.5},
			telem.NewSeriesV[float64](1.5)),
		Entry("an ISO 8601 timestamp",
			telem.Float64T, http.TimeFormatISO8601, nil,
			map[string]any{"ts": "1970-01-01T00:00:10Z", "v": 1.5},
			telem.NewSeriesV[float64](1.5)),
		Entry("an integer into a float channel",
			telem.Float64T, http.TimeFormatUnixNs, nil,
			map[string]any{"ts": 10e9, "v": 2}, telem.NewSeriesV[float64](2)),
		Entry("an integer beyond float64 precision",
			telem.Int64T, http.TimeFormatUnixNs, nil,
			map[string]any{"ts": 10e9, "v": int64(9007199254740993)},
			telem.NewSeriesV[int64](9007199254740993)),
		Entry("the largest unsigned integer",
			telem.Uint64T, http.TimeFormatUnixNs, nil,
			map[string]any{"ts": 10e9, "v": uint64(math.MaxUint64)},
			telem.NewSeriesV[uint64](math.MaxUint64)),
		Entry("an enum label",
			telem.Uint8T, http.TimeFormatUnixNs, onOff,
			map[string]any{"ts": 10e9, "v": "ON"}, telem.NewSeriesV[uint8](1)),
		Entry("a string",
			telem.StringT, http.TimeFormatUnixNs, nil,
			map[string]any{"ts": 10e9, "v": "hello"}, telem.NewSeriesV("hello")),
		Entry("a UUID",
			telem.UUIDT, http.TimeFormatUnixNs, nil,
			map[string]any{"ts": 10e9, "v": readUUID.String()},
			telem.NewSeriesV(readUUID)),
	)

	It("Should keep every nanosecond of a unix_ns timestamp", func(ctx SpecContext) {
		index, data := createIndexed(ctx, telem.Float64T)
		dev := createDevice(ctx, clusterProperties())
		topic := newTopic()
		dt, t := configure(ctx, kafka.ReadConfig{
			Device:  dev.Key,
			Topic:   topic,
			Records: single(valueFields(index, data, telem.Float64T, &unixNs, nil)),
		})
		start(ctx, dt, t)
		stamp := int64(1_700_000_000_000_000_001)
		produceJSON(ctx, producer, topic, "", map[string]any{"ts": stamp, "v": 1.5})
		expectStored(ctx, data.Key(), 1)
		Expect(timestamps(stored(ctx, index.Key()))).
			To(Equal([]telem.TimeStamp{telem.TimeStamp(stamp)}))
	})

	DescribeTable("Should skip a record whose value does not convert",
		func(
			ctx SpecContext,
			dataType telem.DataType,
			format http.TimeFormat,
			enums []http.EnumEntry,
			doc map[string]any,
			msg string,
		) {
			index, data := createIndexed(ctx, dataType)
			dev := createDevice(ctx, clusterProperties())
			topic := newTopic()
			dt, t := configure(ctx, kafka.ReadConfig{
				Device:  dev.Key,
				Topic:   topic,
				Records: single(valueFields(index, data, dataType, &format, enums)),
			})
			start(ctx, dt, t)
			produceJSON(ctx, producer, topic, "", doc)
			Eventually(func() string { return taskStatus(ctx, t).Description }).
				WithTimeout(15 * time.Second).Should(ContainSubstring(msg))
			stat := taskStatus(ctx, t)
			Expect(stat.Variant).To(Equal(status.VariantError))
			Expect(stat.Details.Running).To(BeTrue())
		},
		Entry("a number where an ISO 8601 timestamp is expected",
			telem.Float64T, http.TimeFormatISO8601, nil,
			map[string]any{"ts": 10, "v": 1.5}, "expected an ISO 8601 string"),
		Entry("a malformed ISO 8601 timestamp",
			telem.Float64T, http.TimeFormatISO8601, nil,
			map[string]any{"ts": "yesterday", "v": 1.5},
			"invalid ISO 8601 timestamp"),
		Entry("a string where a numeric timestamp is expected",
			telem.Float64T, http.TimeFormatUnixNs, nil,
			map[string]any{"ts": "10", "v": 1.5}, "expected a number"),
		Entry("an unknown enum label",
			telem.Uint8T, http.TimeFormatUnixNs, onOff,
			map[string]any{"ts": 10e9, "v": "MAYBE"}, "unknown enum label"),
		Entry("an object where a number is expected",
			telem.Float64T, http.TimeFormatUnixNs, nil,
			map[string]any{"ts": 10e9, "v": map[string]any{"x": 1}},
			"expected a number or boolean"),
		Entry("a number where a string is expected",
			telem.StringT, http.TimeFormatUnixNs, nil,
			map[string]any{"ts": 10e9, "v": 1}, "expected a string"),
		Entry("a number where a UUID is expected",
			telem.UUIDT, http.TimeFormatUnixNs, nil,
			map[string]any{"ts": 10e9, "v": 1}, "expected a UUID string"),
		Entry("a malformed UUID",
			telem.UUIDT, http.TimeFormatUnixNs, nil,
			map[string]any{"ts": 10e9, "v": "not-a-uuid"}, "invalid UUID"),
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
				Device:  dev.Key,
				Topic:   "t",
				Records: single(build(ctx, index, data)),
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
		Entry("an index field that is disabled",
			func(_ context.Context, index, data channel.Channel) []kafka.ReadField {
				return []kafka.ReadField{
					{
						Key:        "ts",
						Channel:    index.Key(),
						Pointer:    "/ts",
						DataType:   telem.TimestampT,
						TimeFormat: &unixNs,
						Disabled:   true,
					},
					{
						Key:      "v",
						Channel:  data.Key(),
						Pointer:  "/v",
						DataType: telem.Float64T,
					},
				}
			}, "not an enabled field"),
		Entry("an index field on a data channel",
			func(_ context.Context, _, data channel.Channel) []kafka.ReadField {
				return []kafka.ReadField{{
					Key: "ts", Channel: data.Key(), Pointer: "/v",
					DataType: telem.Float64T,
				}}
			}, "only the record's index field"),
		Entry("a data field on an index channel",
			func(_ context.Context, index, _ channel.Channel) []kafka.ReadField {
				return []kafka.ReadField{{
					Key: "t", Channel: index.Key(), Pointer: "/t",
					DataType: telem.TimestampT, TimeFormat: &unixNs,
				}}
			}, "only the record's index field"),
	)

	It("Should reject two records claiming one record key", func(ctx SpecContext) {
		index, data := createIndexed(ctx, telem.Float64T)
		dev := createDevice(ctx, clusterProperties())
		rec := kafka.ReadRecord{
			RecordKey: "a",
			Index:     "ts",
			Fields:    stampedFields(index, data),
		}
		first, second := rec, rec
		first.Key, second.Key = "r1", "r2"
		cfg := kafka.ReadConfig{
			Device:  dev.Key,
			Topic:   "t",
			Records: []kafka.ReadRecord{first, second},
		}
		Expect(f.ConfigureTask(ctx, newTask(kafka.ReadTaskType, cfg), "cmd-1")).
			Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring(`record key "a": claimed by more than one`)),
		))
	})
})
