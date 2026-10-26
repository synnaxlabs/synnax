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
	"encoding/json/v2"
	"time"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/http"
	"github.com/synnaxlabs/synnax/pkg/service/kafka"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
	"github.com/twmb/franz-go/pkg/kgo"
)

var _ = Describe("Write task", func() {
	var (
		f         driver.Factory
		unixMs    = http.TimeFormatUnixMs
		isoFormat = http.TimeFormatISO8601
		onOff     = []http.EnumEntry{{Label: "OFF", Value: 0}, {Label: "ON", Value: 1}}
		writeUUID = uuid.MustParse("6f1c2a4e-8d3b-4f5a-9c7e-2b1d0e3f4a5b")
	)
	BeforeEach(func() { f = newFactory() })

	// startTask configures and starts a write task, stopping it when the spec ends.
	startTask := func(
		ctx context.Context,
		cfg kafka.WriteConfig,
	) (driver.Task, task.Task) {
		GinkgoHelper()
		t := newTask(kafka.WriteTaskType, cfg)
		dt := MustSucceed(f.ConfigureTask(ctx, t, "cmd-1"))
		DeferCleanup(func() { Expect(dt.Stop(false)).To(Succeed()) })
		Expect(dt.Exec(ctx, task.Command{Task: t.Key, Type: "start", Key: "cmd-2"})).
			To(Succeed())
		stat := taskStatus(ctx, t)
		Expect(stat.Variant).To(Equal(status.VariantSuccess))
		Expect(stat.Details.Running).To(BeTrue())
		Expect(stat.Details.Cmd).To(Equal("cmd-2"))
		// The relay applies the task's streamer demands asynchronously.
		time.Sleep(50 * time.Millisecond)
		return dt, t
	}

	// consume reads at least n records from topic.
	consume := func(ctx context.Context, topic string, n int) []*kgo.Record {
		GinkgoHelper()
		cl := newKafkaClient(
			kgo.ConsumeTopics(topic),
			kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()),
		)
		var recs []*kgo.Record
		Eventually(func() int {
			pollCtx, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
			defer cancel()
			recs = append(recs, cl.PollFetches(pollCtx).Records()...)
			return len(recs)
		}).WithTimeout(15 * time.Second).Should(BeNumerically(">=", n))
		return recs
	}

	decode := func(rec *kgo.Record) map[string]any {
		GinkgoHelper()
		var doc map[string]any
		Expect(json.Unmarshal(rec.Value, &doc)).To(Succeed())
		return doc
	}

	writeFrame := func(
		ctx context.Context,
		index, data channel.Channel,
		stamps []telem.TimeStamp,
		values telem.Series,
	) {
		GinkgoHelper()
		w := MustSucceed(framerSvc.OpenWriter(ctx, framer.WriterConfig{
			Keys:  channel.Keys{index.Key(), data.Key()},
			Start: stamps[0],
		}))
		Expect(MustSucceed(w.Write(frame.NewMulti(
			[]channel.Key{index.Key(), data.Key()},
			[]telem.Series{telem.NewSeries(stamps), values},
		)))).To(BeTrue())
		Expect(w.Close()).To(Succeed())
	}

	It("Should produce one record per sample in the narrow layout",
		func(ctx SpecContext) {
			index, data := createIndexed(ctx, telem.Float64T)
			dev := createDevice(ctx, clusterProperties())
			topic := uniqueTopic()
			startTask(ctx, kafka.WriteConfig{
				Device: dev.Key,
				Topic:  topic,
				Record: kafka.Record{
					ChannelPointer:   new("/channel"),
					TimestampPointer: new("/timestamp"),
				},
				Channels: []kafka.WriteChannel{{Key: "c1", Channel: data.Key()}},
			})
			start := telem.TimeStamp(10 * telem.Second)
			writeFrame(ctx, index, data,
				[]telem.TimeStamp{start, start + telem.SecondTS},
				telem.NewSeriesV[float64](1.5, 2.5),
			)
			recs := consume(ctx, topic, 2)
			Expect(recs).To(HaveLen(2))
			Expect(decode(recs[0])).To(Equal(map[string]any{
				"value":     1.5,
				"channel":   data.Name,
				"timestamp": float64(start),
			}))
			Expect(string(recs[0].Key)).To(Equal(data.Name))
			// Kafka record timestamps carry millisecond precision.
			Expect(recs[0].Timestamp.UnixMilli()).To(Equal(int64(10_000)))
			Expect(decode(recs[1])).To(HaveKeyWithValue("value", 2.5))
			Expect(recs[1].Timestamp.UnixMilli()).To(Equal(int64(11_000)))
		},
	)

	It("Should apply the record shape, enum labels, and extra fields",
		func(ctx SpecContext) {
			index, data := createIndexed(ctx, telem.Uint8T)
			dev := createDevice(ctx, clusterProperties())
			topic := uniqueTopic()
			iso := http.TimeFormatISO8601
			startTask(ctx, kafka.WriteConfig{
				Device:    dev.Key,
				Topic:     topic,
				RecordKey: kafka.RecordKeyNone,
				Record: kafka.Record{
					ValuePointer: "/payload/state",
					Fields: []http.WriteField{
						{Variant: http.StaticWriteField{
							BaseWriteField: http.BaseWriteField{
								Key: "source", Pointer: "/meta/source",
							},
							JSONType: http.JSONTypeString,
							Value:    "synnax",
						}},
						{Variant: http.GeneratedWriteField{
							BaseWriteField: http.BaseWriteField{
								Key: "id", Pointer: "/id",
							},
							Generator: http.GeneratorTypeUUID,
						}},
						{Variant: http.GeneratedWriteField{
							BaseWriteField: http.BaseWriteField{
								Key: "sent", Pointer: "/meta/sent",
							},
							Generator:  http.GeneratorTypeTimestamp,
							TimeFormat: &iso,
						}},
					},
				},
				Channels: []kafka.WriteChannel{{
					Key:      "c1",
					Channel:  data.Key(),
					JSONType: http.JSONTypeString,
					EnumValues: []http.EnumEntry{
						{Label: "OFF", Value: 0},
						{Label: "ON", Value: 1},
					},
				}},
			})
			start := telem.TimeStamp(20 * telem.Second)
			writeFrame(ctx, index, data,
				[]telem.TimeStamp{start},
				telem.NewSeriesV[uint8](1),
			)
			recs := consume(ctx, topic, 1)
			doc := decode(recs[0])
			Expect(doc).NotTo(HaveKey("channel"))
			Expect(doc).NotTo(HaveKey("timestamp"))
			Expect(doc["payload"]).To(HaveKeyWithValue("state", "ON"))
			Expect(doc["meta"]).To(HaveKeyWithValue("source", "synnax"))
			Expect(doc["meta"]).To(HaveKeyWithValue("sent", "1970-01-01T00:00:20Z"))
			Expect(doc["id"]).To(HaveLen(36))
			Expect(recs[0].Key).To(BeEmpty())
		},
	)

	It("Should answer start with an error status when the cluster is unreachable",
		func(ctx SpecContext) {
			_, data := createIndexed(ctx, telem.Float64T)
			dev := createDevice(ctx, kafka.Properties{Brokers: []string{"127.0.0.1:1"}})
			t := newTask(kafka.WriteTaskType, kafka.WriteConfig{
				Device:   dev.Key,
				Topic:    uniqueTopic(),
				Channels: []kafka.WriteChannel{{Key: "c1", Channel: data.Key()}},
			})
			dt := MustSucceed(f.ConfigureTask(ctx, t, "cmd-1"))
			DeferCleanup(func() { Expect(dt.Stop(false)).To(Succeed()) })
			Expect(dt.Exec(ctx, task.Command{
				Task: t.Key, Type: "start", Key: "cmd-2",
			})).To(MatchError(ContainSubstring("connecting to cluster")))
			stat := taskStatus(ctx, t)
			Expect(stat.Variant).To(Equal(status.VariantError))
			Expect(stat.Details.Running).To(BeFalse())
			Expect(stat.Details.Cmd).To(Equal("cmd-2"))
		},
	)

	It("Should reject an unknown channel", func(ctx SpecContext) {
		dev := createDevice(ctx, clusterProperties())
		cfg := kafka.WriteConfig{
			Device:   dev.Key,
			Topic:    "t",
			Channels: []kafka.WriteChannel{{Key: "c1", Channel: 999999}},
		}
		Expect(f.ConfigureTask(ctx, newTask(kafka.WriteTaskType, cfg), "cmd-1")).
			Error().To(MatchError(query.ErrNotFound))
	})

	// startOne configures and starts a write task producing one channel of dt.
	startOne := func(
		ctx context.Context,
		dt telem.DataType,
		cfg kafka.WriteChannel,
	) (index, data channel.Channel, topic string, t task.Task) {
		GinkgoHelper()
		index, data = createIndexed(ctx, dt)
		dev := createDevice(ctx, clusterProperties())
		topic = uniqueTopic()
		cfg.Key, cfg.Channel = "c1", data.Key()
		_, t = startTask(ctx, kafka.WriteConfig{
			Device:   dev.Key,
			Topic:    topic,
			Channels: []kafka.WriteChannel{cfg},
		})
		return index, data, topic, t
	}

	It("Should report data that arrives without its index", func(ctx SpecContext) {
		index, data, topic, t := startOne(ctx, telem.Float64T, kafka.WriteChannel{})
		start := telem.TimeStamp(20 * telem.Second)
		write := func(key channel.Key, s telem.Series) {
			GinkgoHelper()
			w := MustSucceed(framerSvc.OpenWriter(ctx, framer.WriterConfig{
				Keys:  channel.Keys{key},
				Start: start,
			}))
			Expect(MustSucceed(w.Write(frame.NewUnary(key, s)))).To(BeTrue())
			Expect(w.Close()).To(Succeed())
		}
		write(index.Key(), telem.NewSeries([]telem.TimeStamp{start}))
		write(data.Key(), telem.NewSeriesV[float64](1.5))
		Eventually(func() string { return taskStatus(ctx, t).Description }).
			Should(Equal("Channel " + data.Name + " arrived without its index"))
		Expect(taskStatus(ctx, t).Variant).To(Equal(status.VariantError))
		Consistently(func() []*kgo.Record {
			cl := newKafkaClient(
				kgo.ConsumeTopics(topic),
				kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()),
			)
			defer cl.Close()
			pollCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
			defer cancel()
			return cl.PollFetches(pollCtx).Records()
		}).WithTimeout(500 * time.Millisecond).Should(BeEmpty())
	})

	DescribeTable("Should serialize the sample as the configured JSON type",
		func(
			ctx SpecContext,
			dt telem.DataType,
			values telem.Series,
			cfg kafka.WriteChannel,
			expected any,
		) {
			index, data, topic, _ := startOne(ctx, dt, cfg)
			writeFrame(ctx, index, data,
				[]telem.TimeStamp{telem.TimeStamp(10 * telem.Second)}, values)
			recs := consume(ctx, topic, 1)
			Expect(decode(recs[0])).To(HaveKeyWithValue("value", expected))
		},
		Entry("float32 as a number",
			telem.Float32T, telem.NewSeriesV[float32](1.5), kafka.WriteChannel{}, 1.5),
		Entry("int64 as a number",
			telem.Int64T, telem.NewSeriesV[int64](-7), kafka.WriteChannel{}, -7.0),
		Entry("int32 as a number",
			telem.Int32T, telem.NewSeriesV[int32](-7), kafka.WriteChannel{}, -7.0),
		Entry("int16 as a number",
			telem.Int16T, telem.NewSeriesV[int16](-7), kafka.WriteChannel{}, -7.0),
		Entry("int8 as a number",
			telem.Int8T, telem.NewSeriesV[int8](-7), kafka.WriteChannel{}, -7.0),
		Entry("uint64 as a number",
			telem.Uint64T, telem.NewSeriesV[uint64](7), kafka.WriteChannel{}, 7.0),
		Entry("uint32 as a number",
			telem.Uint32T, telem.NewSeriesV[uint32](7), kafka.WriteChannel{}, 7.0),
		Entry("uint16 as a number",
			telem.Uint16T, telem.NewSeriesV[uint16](7), kafka.WriteChannel{}, 7.0),
		Entry("uint8 as a boolean",
			telem.Uint8T, telem.NewSeriesV[uint8](1),
			kafka.WriteChannel{JSONType: http.JSONTypeBoolean}, true),
		Entry("float64 as a string",
			telem.Float64T, telem.NewSeriesV[float64](2.5),
			kafka.WriteChannel{JSONType: http.JSONTypeString}, "2.5"),
		Entry("int64 as a string",
			telem.Int64T, telem.NewSeriesV[int64](9007199254740993),
			kafka.WriteChannel{JSONType: http.JSONTypeString}, "9007199254740993"),
		Entry("a timestamp in unix milliseconds",
			telem.TimestampT, telem.NewSeriesV(10*telem.SecondTS),
			kafka.WriteChannel{TimeFormat: &unixMs}, 10_000.0),
		Entry("a timestamp in ISO 8601",
			telem.TimestampT, telem.NewSeriesV(10*telem.SecondTS),
			kafka.WriteChannel{TimeFormat: &isoFormat}, "1970-01-01T00:00:10Z"),
		Entry("a string",
			telem.StringT, telem.NewSeriesV("hi"),
			kafka.WriteChannel{JSONType: http.JSONTypeString}, "hi"),
		Entry("a UUID",
			telem.UUIDT, telem.NewSeriesV(writeUUID),
			kafka.WriteChannel{JSONType: http.JSONTypeString}, writeUUID.String()),
	)

	It("Should keep an int64 sample beyond float64 precision", func(ctx SpecContext) {
		index, data, topic, _ := startOne(ctx, telem.Int64T, kafka.WriteChannel{})
		writeFrame(ctx, index, data,
			[]telem.TimeStamp{telem.TimeStamp(10 * telem.Second)},
			telem.NewSeriesV[int64](9007199254740993))
		recs := consume(ctx, topic, 1)
		Expect(string(recs[0].Value)).To(ContainSubstring(`"value":9007199254740993`))
	})

	DescribeTable("Should report a sample that does not serialize",
		func(
			ctx SpecContext,
			dt telem.DataType,
			values telem.Series,
			cfg kafka.WriteChannel,
			msg string,
		) {
			index, data, _, t := startOne(ctx, dt, cfg)
			writeFrame(ctx, index, data,
				[]telem.TimeStamp{telem.TimeStamp(10 * telem.Second)}, values)
			Eventually(func() string { return taskStatus(ctx, t).Description }).
				WithTimeout(15 * time.Second).Should(ContainSubstring(msg))
			stat := taskStatus(ctx, t)
			Expect(stat.Variant).To(Equal(status.VariantError))
			Expect(stat.Details.Running).To(BeTrue())
		},
		Entry("a value outside the enum",
			telem.Uint8T, telem.NewSeriesV[uint8](5),
			kafka.WriteChannel{JSONType: http.JSONTypeString, EnumValues: onOff},
			"no enum label for value 5"),
		Entry("a string as a number",
			telem.StringT, telem.NewSeriesV("hi"), kafka.WriteChannel{},
			"only serializes as a string"),
		Entry("a UUID as a number",
			telem.UUIDT, telem.NewSeriesV(writeUUID), kafka.WriteChannel{},
			"only serializes as a string"),
	)

	DescribeTable("Should reject an invalid config",
		func(
			ctx SpecContext,
			build func(data channel.Channel) kafka.WriteConfig,
			msg string,
		) {
			_, data := createIndexed(ctx, telem.TimestampT)
			dev := createDevice(ctx, clusterProperties())
			cfg := build(data)
			cfg.Device = dev.Key
			Expect(f.ConfigureTask(ctx, newTask(kafka.WriteTaskType, cfg), "cmd-1")).
				Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(msg)),
			))
		},
		Entry("empty topic", func(data channel.Channel) kafka.WriteConfig {
			return kafka.WriteConfig{
				Channels: []kafka.WriteChannel{{Key: "c1", Channel: data.Key()}},
			}
		}, "topic"),
		Entry("no enabled channel", func(data channel.Channel) kafka.WriteConfig {
			return kafka.WriteConfig{
				Topic: "t",
				Channels: []kafka.WriteChannel{
					{Key: "c1", Channel: data.Key(), Disabled: true},
				},
			}
		}, "at least one"),
		Entry("timestamp channel without a time format",
			func(data channel.Channel) kafka.WriteConfig {
				return kafka.WriteConfig{
					Topic:    "t",
					Channels: []kafka.WriteChannel{{Key: "c1", Channel: data.Key()}},
				}
			}, "time_format"),
	)
})
