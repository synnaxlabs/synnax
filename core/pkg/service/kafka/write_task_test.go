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
	"fmt"
	"math"
	"sync"
	"time"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/http"
	"github.com/synnaxlabs/synnax/pkg/service/kafka"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/payload"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/hookkv"
	"github.com/synnaxlabs/x/validate"
	"github.com/twmb/franz-go/pkg/kfake"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/kmsg"
)

var _ = Describe("Write task", func() {
	var (
		f         driver.Factory
		unixMs    = payload.TimeFormatUnixMs
		isoFormat = payload.TimeFormatISO8601
		onOff     = []payload.EnumEntry{
			{Label: "OFF", Value: 0},
			{Label: "ON", Value: 1},
		}
		writeUUID = uuid.MustParse("6f1c2a4e-8d3b-4f5a-9c7e-2b1d0e3f4a5b")
	)
	BeforeEach(func() { f = newFactory() })

	// startTask configures and starts a write task, stopping it when the spec ends.
	startTask := func(
		ctx context.Context,
		cfg kafka.WriteConfig,
	) (driver.Task, task.Task) {
		GinkgoHelper()
		t := newTask(ctx, kafka.WriteTaskType, cfg)
		dt := MustSucceed(f.ConfigureTask(ctx, t, "cmd-1"))
		DeferCleanup(func() { Expect(dt.Stop(false)).To(Succeed()) })
		Expect(dt.Exec(ctx, task.Command{Task: t.Key, Type: "start", Key: "cmd-2"})).
			To(Succeed())
		stat := taskStatus(ctx, t)
		Expect(stat.Variant).To(Equal(status.VariantSuccess))
		Expect(stat.Details.Running).To(BeTrue())
		Expect(stat.Details.Cmd).To(Equal("cmd-2"))
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
			iso := payload.TimeFormatISO8601
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
							JSONType: payload.JSONTypeString,
							Value:    "synnax",
						}},
						{Variant: http.GeneratedWriteField{
							BaseWriteField: http.BaseWriteField{
								Key: "id", Pointer: "/id",
							},
							Generator: payload.GeneratorTypeUUID,
						}},
						{Variant: http.GeneratedWriteField{
							BaseWriteField: http.BaseWriteField{
								Key: "sent", Pointer: "/meta/sent",
							},
							Generator:  payload.GeneratorTypeTimestamp,
							TimeFormat: &iso,
						}},
					},
				},
				Channels: []kafka.WriteChannel{{
					Key:      "c1",
					Channel:  data.Key(),
					JSONType: payload.JSONTypeString,
					EnumValues: []payload.EnumEntry{
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

	It("Should encode a generated timestamp field with no time format as ISO 8601",
		func(ctx SpecContext) {
			index, data := createIndexed(ctx, telem.Float64T)
			dev := createDevice(ctx, clusterProperties())
			topic := uniqueTopic()
			startTask(ctx, kafka.WriteConfig{
				Device: dev.Key,
				Topic:  topic,
				Record: kafka.Record{
					ValuePointer: "/value",
					Fields: []http.WriteField{{Variant: http.GeneratedWriteField{
						BaseWriteField: http.BaseWriteField{
							Key:     "sent",
							Pointer: "/sent",
						},
						Generator: payload.GeneratorTypeTimestamp,
					}}},
				},
				Channels: []kafka.WriteChannel{{Key: "c1", Channel: data.Key()}},
			})
			writeFrame(ctx, index, data,
				[]telem.TimeStamp{telem.TimeStamp(20 * telem.Second)},
				telem.NewSeriesV[float64](1.5),
			)
			recs := consume(ctx, topic, 1)
			Expect(decode(recs[0])).To(HaveKeyWithValue("sent", "1970-01-01T00:00:20Z"))
		},
	)

	It("Should produce every series a free channel carries in one frame",
		func(ctx SpecContext) {
			ch := channel.Channel{
				Name:     "free_" + uuid.New().String()[:8],
				DataType: telem.Float64T,
				Virtual:  true,
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return channelSvc.NewWriter(tx).Create(ctx, &ch)
			})).To(Succeed())
			dev := createDevice(ctx, clusterProperties())
			topic := uniqueTopic()
			startTask(ctx, kafka.WriteConfig{
				Device:   dev.Key,
				Topic:    topic,
				Channels: []kafka.WriteChannel{{Key: "c1", Channel: ch.Key()}},
			})
			w := MustSucceed(framerSvc.OpenWriter(ctx, framer.WriterConfig{
				Keys:  channel.Keys{ch.Key()},
				Start: telem.Now(),
			}))
			Expect(MustSucceed(w.Write(frame.NewMulti(
				[]channel.Key{ch.Key(), ch.Key()},
				[]telem.Series{
					telem.NewSeriesV[float64](1.5),
					telem.NewSeriesV[float64](2.5),
				},
			)))).To(BeTrue())
			Expect(w.Close()).To(Succeed())
			recs := consume(ctx, topic, 2)
			Expect(recs).To(HaveLen(2))
			Expect(decode(recs[0])).To(HaveKeyWithValue("value", 1.5))
			Expect(decode(recs[1])).To(HaveKeyWithValue("value", 2.5))
		},
	)

	It("Should answer start with an error status when the cluster is unreachable",
		func(ctx SpecContext) {
			_, data := createIndexed(ctx, telem.Float64T)
			dev := createDevice(ctx, kafka.Properties{Brokers: []string{"127.0.0.1:1"}})
			t := newTask(ctx, kafka.WriteTaskType, kafka.WriteConfig{
				Device:   dev.Key,
				Topic:    uniqueTopic(),
				Channels: []kafka.WriteChannel{{Key: "c1", Channel: data.Key()}},
			})
			dt := MustSucceed(f.ConfigureTask(ctx, t, "cmd-1"))
			DeferCleanup(func() { Expect(dt.Stop(false)).To(Succeed()) })
			Expect(dt.Exec(ctx, task.Command{
				Task: t.Key, Type: "start", Key: "cmd-2",
			})).To(Succeed())
			stat := taskStatus(ctx, t)
			Expect(stat.Variant).To(Equal(status.VariantError))
			Expect(stat.Description).To(ContainSubstring("connecting to cluster"))
			Expect(stat.Details.Running).To(BeFalse())
			Expect(stat.Details.Cmd).To(Equal("cmd-2"))
		},
	)

	It("Should return the error from Exec for a start it could not answer",
		func(ctx SpecContext) {
			_, data := createIndexed(ctx, telem.Float64T)
			dev := createDevice(ctx, clusterProperties())
			t := newTask(ctx, kafka.WriteTaskType, kafka.WriteConfig{
				Device:   dev.Key,
				Topic:    uniqueTopic(),
				Channels: []kafka.WriteChannel{{Key: "c1", Channel: data.Key()}},
			})
			dt := MustSucceed(f.ConfigureTask(ctx, t, "cmd-1"))
			DeferCleanup(func() { Expect(dt.Stop(false)).To(Succeed()) })
			Expect(dt.Exec(hookkv.WithHook(ctx, refuse), task.Command{
				Task: t.Key, Type: "start", Key: "cmd-2",
			})).To(MatchError(errCommit))
		},
	)

	It("Should return no error from Exec for a start answered before ctx ended",
		func(ctx SpecContext) {
			_, data := createIndexed(ctx, telem.Float64T)
			dev := createDevice(ctx, kafka.Properties{Brokers: []string{"127.0.0.1:1"}})
			t := newTask(ctx, kafka.WriteTaskType, kafka.WriteConfig{
				Device:   dev.Key,
				Topic:    uniqueTopic(),
				Channels: []kafka.WriteChannel{{Key: "c1", Channel: data.Key()}},
			})
			cmdCtx, cancel := context.WithCancel(ctx)
			cmdCtx = hookkv.WithHook(cmdCtx, func(commit func() error) error {
				err := commit()
				cancel()
				return err
			})
			f = newFactory(kafka.FactoryConfig{PingTimeout: 100 * telem.Millisecond})
			dt := MustSucceed(f.ConfigureTask(ctx, t, "cmd-1"))
			DeferCleanup(func() { Expect(dt.Stop(false)).To(Succeed()) })
			Expect(dt.Exec(cmdCtx, task.Command{
				Task: t.Key, Type: "start", Key: "cmd-2",
			})).To(Succeed())
			Expect(cmdCtx.Err()).To(MatchError(context.Canceled))
			stat := taskStatus(ctx, t)
			Expect(stat.Message).To(Equal("Failed to start task"))
			Expect(stat.Details.Cmd).To(Equal("cmd-2"))
		},
	)

	Describe("Lost cluster", func() {
		// startBlocked starts a task on a cluster that drops every produce request, and
		// returns once the producer holds more records than its buffer.
		startBlocked := func(ctx SpecContext) (driver.Task, task.Task) {
			GinkgoHelper()
			lost := MustSucceed(kfake.NewCluster(
				kfake.NumBrokers(1),
				kfake.AllowAutoTopicCreation(),
			))
			DeferCleanup(lost.Close)
			producing := make(chan struct{})
			var once sync.Once
			lost.ControlKey(
				kmsg.Produce.Int16(),
				func(kmsg.Request) (kmsg.Response, error, bool) {
					lost.KeepControl()
					once.Do(func() { close(producing) })
					return nil, errors.New("cluster lost"), true
				},
			)
			index, data := createIndexed(ctx, telem.Float64T)
			dev := createDevice(ctx, kafka.Properties{Brokers: lost.ListenAddrs()})
			f = newFactory(kafka.FactoryConfig{PingTimeout: 200 * telem.Millisecond})
			t := newTask(ctx, kafka.WriteTaskType, kafka.WriteConfig{
				Device:   dev.Key,
				Topic:    uniqueTopic(),
				Channels: []kafka.WriteChannel{{Key: "c1", Channel: data.Key()}},
			})
			dt := MustSucceed(f.ConfigureTask(ctx, t, "cmd-1"))
			Expect(dt.Exec(ctx, task.Command{
				Task: t.Key, Type: "start", Key: "cmd-2",
			})).To(Succeed())
			// More samples than the producer buffers by default.
			const n = 60_000
			stamps := make([]telem.TimeStamp, n)
			values := make([]float64, n)
			for i := range n {
				stamps[i] = telem.TimeStamp(i+1) * telem.MillisecondTS
				values[i] = float64(i)
			}
			writeFrame(ctx, index, data, stamps, telem.NewSeries(values))
			Eventually(producing).WithTimeout(10 * time.Second).Should(BeClosed())
			return dt, t
		}

		It("Should answer a stop command with a timeout", func(ctx SpecContext) {
			dt, t := startBlocked(ctx)
			Expect(dt.Exec(ctx, task.Command{
				Task: t.Key, Type: "stop", Key: "cmd-3",
			})).To(Succeed())
			stat := taskStatus(ctx, t)
			Expect(stat.Variant).To(Equal(status.VariantError))
			Expect(stat.Message).To(Equal("Failed to stop task"))
			Expect(stat.Description).To(ContainSubstring("timed out producing records"))
			Expect(stat.Details.Running).To(BeFalse())
			Expect(stat.Details.Cmd).To(Equal("cmd-3"))
			Consistently(func() string { return taskStatus(ctx, t).Message }).
				WithTimeout(250 * time.Millisecond).
				Should(Equal("Failed to stop task"))
		})

		It("Should time out a stop without a status", func(ctx SpecContext) {
			dt, _ := startBlocked(ctx)
			stopped := make(chan error, 1)
			go func() { stopped <- dt.Stop(false) }()
			Eventually(stopped).WithTimeout(10 * time.Second).
				Should(Receive(MatchError(ContainSubstring("timed out producing"))))
		})
	})

	It("Should reject an unknown channel", func(ctx SpecContext) {
		dev := createDevice(ctx, clusterProperties())
		cfg := kafka.WriteConfig{
			Device:   dev.Key,
			Topic:    "t",
			Channels: []kafka.WriteChannel{{Key: "c1", Channel: 999999}},
		}
		Expect(f.ConfigureTask(ctx, newTask(ctx, kafka.WriteTaskType, cfg), "cmd-1")).
			Error().To(MatchError(query.ErrNotFound))
	})

	DescribeTable("Should reject output pointers that collide",
		func(ctx SpecContext, rec kafka.Record, msg string) {
			_, data := createIndexed(ctx, telem.Float64T)
			dev := createDevice(ctx, clusterProperties())
			cfg := kafka.WriteConfig{
				Device:   dev.Key,
				Topic:    "t",
				Record:   rec,
				Channels: []kafka.WriteChannel{{Key: "c1", Channel: data.Key()}},
			}
			Expect(
				f.ConfigureTask(ctx, newTask(ctx, kafka.WriteTaskType, cfg), "cmd-1"),
			).
				Error().
				To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring(msg)),
				))
		},
		Entry(
			"a channel pointer equal to the value pointer",
			kafka.Record{ValuePointer: "/value", ChannelPointer: new("/value")},
			`record.value_pointer "/value" and record.channel_pointer "/value" `+
				`overlap`,
		),
		Entry("a channel pointer nested inside the value pointer",
			kafka.Record{ValuePointer: "/value", ChannelPointer: new("/value/name")},
			`record.value_pointer "/value" and `+
				`record.channel_pointer "/value/name" overlap`),
		Entry("a timestamp pointer lacking its leading slash",
			kafka.Record{TimestampPointer: new("timestamp")},
			`record.timestamp_pointer: json pointer "timestamp" must be empty or `+
				`start with "/"`),
		Entry("a field pointer that contains the value pointer",
			kafka.Record{
				ValuePointer: "/meta/value",
				Fields: []http.WriteField{{Variant: http.StaticWriteField{
					BaseWriteField: http.BaseWriteField{
						Key:     "source",
						Pointer: "/meta",
					},
					JSONType: payload.JSONTypeString,
					Value:    "synnax",
				}}},
			},
			`record.value_pointer "/meta/value" and field "source" "/meta" `+
				`overlap`),
	)

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
			kafka.WriteChannel{JSONType: payload.JSONTypeBoolean}, true),
		Entry("a boolean as a number",
			telem.BooleanT, telem.NewSeriesV(true), kafka.WriteChannel{}, 1.0),
		Entry("a boolean as a string",
			telem.BooleanT, telem.NewSeriesV(false),
			kafka.WriteChannel{JSONType: payload.JSONTypeString}, "0"),
		Entry("a float as a boolean",
			telem.Float64T, telem.NewSeriesV[float64](0.5),
			kafka.WriteChannel{JSONType: payload.JSONTypeBoolean}, true),
		Entry(
			"a uint64 as a string",
			telem.Uint64T,
			telem.NewSeriesV[uint64](18446744073709551615),
			kafka.WriteChannel{
				JSONType: payload.JSONTypeString,
			},
			"18446744073709551615",
		),
		Entry("float64 as a string",
			telem.Float64T, telem.NewSeriesV[float64](2.5),
			kafka.WriteChannel{JSONType: payload.JSONTypeString}, "2.5"),
		Entry(
			"a value as its enum label",
			telem.Uint8T,
			telem.NewSeriesV[uint8](1),
			kafka.WriteChannel{
				JSONType:   payload.JSONTypeString,
				EnumValues: onOff,
			},
			"ON",
		),
		Entry(
			"a value outside the enum as text",
			telem.Uint8T,
			telem.NewSeriesV[uint8](5),
			kafka.WriteChannel{
				JSONType:   payload.JSONTypeString,
				EnumValues: onOff,
			},
			"5",
		),
		Entry("infinity as text",
			telem.Float64T, telem.NewSeriesV(math.Inf(-1)),
			kafka.WriteChannel{JSONType: payload.JSONTypeString}, "-Infinity"),
		Entry("int64 as a string",
			telem.Int64T, telem.NewSeriesV[int64](9007199254740993),
			kafka.WriteChannel{JSONType: payload.JSONTypeString}, "9007199254740993"),
		Entry("a timestamp in unix milliseconds",
			telem.TimestampT, telem.NewSeriesV(10*telem.SecondTS),
			kafka.WriteChannel{TimeFormat: &unixMs}, 10_000.0),
		Entry("a timestamp in ISO 8601",
			telem.TimestampT, telem.NewSeriesV(10*telem.SecondTS),
			kafka.WriteChannel{TimeFormat: &isoFormat}, "1970-01-01T00:00:10Z"),
		Entry("a string",
			telem.StringT, telem.NewSeriesV("hi"),
			kafka.WriteChannel{JSONType: payload.JSONTypeString}, "hi"),
		Entry("a UUID",
			telem.UUIDT, telem.NewSeriesV(writeUUID),
			kafka.WriteChannel{JSONType: payload.JSONTypeString}, writeUUID.String()),
		Entry("bytes as base64",
			telem.BytesT, telem.NewSeriesV([]byte{0x00, 0xff, 0x3f}),
			kafka.WriteChannel{JSONType: payload.JSONTypeString}, "AP8/"),
	)

	It("Should keep an int64 sample beyond float64 precision", func(ctx SpecContext) {
		index, data, topic, _ := startOne(ctx, telem.Int64T, kafka.WriteChannel{})
		writeFrame(ctx, index, data,
			[]telem.TimeStamp{telem.TimeStamp(10 * telem.Second)},
			telem.NewSeriesV[int64](9007199254740993))
		recs := consume(ctx, topic, 1)
		Expect(string(recs[0].Value)).To(ContainSubstring(`"value":9007199254740993`))
	})

	It("Should write record members in config order", func(ctx SpecContext) {
		index, data := createIndexed(ctx, telem.Float64T)
		dev := createDevice(ctx, clusterProperties())
		topic := uniqueTopic()
		startTask(ctx, kafka.WriteConfig{
			Device: dev.Key,
			Topic:  topic,
			Record: kafka.Record{
				ValuePointer:     "/payload/value",
				ChannelPointer:   new("/channel"),
				TimestampPointer: new("/meta/sent"),
				TimeFormat:       payload.TimeFormatUnixNs,
				Fields: []http.WriteField{
					{Variant: http.StaticWriteField{
						BaseWriteField: http.BaseWriteField{
							Key: "unit", Pointer: "/payload/unit",
						},
						JSONType: payload.JSONTypeString,
						Value:    "psi",
					}},
					{Variant: http.StaticWriteField{
						BaseWriteField: http.BaseWriteField{
							Key: "source", Pointer: "/meta/source",
						},
						JSONType: payload.JSONTypeString,
						Value:    "synnax",
					}},
				},
			},
			Channels: []kafka.WriteChannel{{Key: "c1", Channel: data.Key()}},
		})
		start := 10 * telem.SecondTS
		writeFrame(ctx, index, data, []telem.TimeStamp{start},
			telem.NewSeriesV[float64](1.5))
		recs := consume(ctx, topic, 1)
		Expect(string(recs[0].Value)).To(Equal(
			`{"payload":{"value":1.5,"unit":"psi"},"channel":"` + data.Name +
				`","meta":{"sent":10000000000,"source":"synnax"}}`,
		))
	})

	DescribeTable("Should write a number in its shortest JSON form",
		func(ctx SpecContext, value float64, expected string) {
			index, data, topic, _ := startOne(ctx, telem.Float64T, kafka.WriteChannel{})
			writeFrame(ctx, index, data,
				[]telem.TimeStamp{telem.TimeStamp(10 * telem.Second)},
				telem.NewSeriesV(value))
			recs := consume(ctx, topic, 1)
			Expect(string(recs[0].Value)).To(Equal(`{"value":` + expected + `}`))
		},
		Entry("a whole number", 100000.0, "100000"),
		Entry("a fraction", 0.1, "0.1"),
		Entry("negative zero", math.Copysign(0, -1), "-0"),
		Entry("the largest plain number", 1e20, "100000000000000000000"),
		Entry("the first exponent past it", 1e21, "1e+21"),
		Entry("the smallest plain number", 1e-6, "0.000001"),
		Entry("a one-digit negative exponent", 1e-7, "1e-7"),
		Entry("a two-digit negative exponent", 1.5e-10, "1.5e-10"),
	)

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
		Entry("NaN as a number",
			telem.Float64T, telem.NewSeriesV(math.NaN()), kafka.WriteChannel{},
			"NaN is not a JSON number"),
		Entry("infinity as a number",
			telem.Float64T, telem.NewSeriesV(math.Inf(1)), kafka.WriteChannel{},
			"Infinity is not a JSON number"),
	)

	DescribeTable("Should reject a JSON type that a channel cannot serialize as",
		func(ctx SpecContext, dt telem.DataType, jsonType payload.JSONType) {
			_, data := createIndexed(ctx, dt)
			dev := createDevice(ctx, clusterProperties())
			cfg := kafka.WriteConfig{
				Device: dev.Key,
				Topic:  "t",
				Channels: []kafka.WriteChannel{
					{Key: "c1", Channel: data.Key(), JSONType: jsonType},
				},
			}
			Expect(
				f.ConfigureTask(ctx, newTask(ctx, kafka.WriteTaskType, cfg), "cmd-1"),
			).
				Error().
				To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring(fmt.Sprintf(
						`channel "c1": data type %s only serializes as a string`, dt,
					))),
				))
		},
		Entry("a string as a number", telem.StringT, payload.JSONTypeNumber),
		Entry("a UUID as a boolean", telem.UUIDT, payload.JSONTypeBoolean),
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
			Expect(
				f.ConfigureTask(ctx, newTask(ctx, kafka.WriteTaskType, cfg), "cmd-1"),
			).
				Error().
				To(SatisfyAll(
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
		Entry("timestamp channel with a JSON type",
			func(data channel.Channel) kafka.WriteConfig {
				return kafka.WriteConfig{
					Topic: "t",
					Channels: []kafka.WriteChannel{{
						Key:        "c1",
						Channel:    data.Key(),
						TimeFormat: &unixMs,
						JSONType:   payload.JSONTypeString,
					}},
				}
			}, "time_format alone encodes"),
		Entry("timestamp channel with enum values",
			func(data channel.Channel) kafka.WriteConfig {
				return kafka.WriteConfig{
					Topic: "t",
					Channels: []kafka.WriteChannel{{
						Key:        "c1",
						Channel:    data.Key(),
						TimeFormat: &unixMs,
						EnumValues: onOff,
					}},
				}
			}, "time_format alone encodes"),
	)

	It("Should reject enum values on a number JSON type", func(ctx SpecContext) {
		_, data := createIndexed(ctx, telem.Uint8T)
		dev := createDevice(ctx, clusterProperties())
		cfg := kafka.WriteConfig{
			Device: dev.Key,
			Topic:  "t",
			Channels: []kafka.WriteChannel{{
				Key:        "c1",
				Channel:    data.Key(),
				JSONType:   payload.JSONTypeNumber,
				EnumValues: onOff,
			}},
		}
		Expect(
			f.ConfigureTask(ctx, newTask(ctx, kafka.WriteTaskType, cfg), "cmd-1"),
		).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring(
				`channel "c1": enum values require the string JSON type`,
			)),
		))
	})
})
