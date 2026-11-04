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
	"encoding/json/jsontext"
	"encoding/json/v2"
	"math"
	"net"
	"slices"
	"strconv"
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
	"github.com/twmb/franz-go/pkg/kfake"
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

	// read returns every sample persisted in key, asserting through g.
	read := func(g Gomega, ctx context.Context, key channel.Key) telem.Series {
		GinkgoHelper()
		it, err := framerSvc.OpenIterator(ctx, framer.IteratorConfig{
			Keys:   channel.Keys{key},
			Bounds: telem.TimeRangeMax,
		})
		g.Expect(err).To(Succeed())
		defer func() { g.Expect(it.Close()).To(Succeed()) }()
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
		g.Expect(it.Error()).To(Succeed())
		return s
	}

	stored := func(ctx context.Context, key channel.Key) telem.Series {
		GinkgoHelper()
		return read(Default, ctx, key)
	}

	// expectStored waits until key holds n samples. A read can fail while a commit is
	// in flight (SY-5459), so a failed read retries.
	expectStored := func(ctx context.Context, key channel.Key, n int64) telem.Series {
		GinkgoHelper()
		Eventually(func(g Gomega) int64 { return read(g, ctx, key).Len() }).
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

	// unstampedFields reads /v into data and no timestamp field.
	unstampedFields := func(data channel.Channel) []kafka.ReadField {
		return []kafka.ReadField{{
			Key:      "v",
			Channel:  data.Key(),
			Pointer:  "/v",
			DataType: telem.Float64T,
		}}
	}

	// produceAt produces a JSON record whose Kafka timestamp is ms milliseconds.
	produceAt := func(ctx context.Context, topic string, ms int64, doc any) {
		GinkgoHelper()
		rec := &kgo.Record{
			Topic:     topic,
			Value:     MustSucceed(json.Marshal(doc)),
			Timestamp: time.UnixMilli(ms),
		}
		Expect(producer.ProduceSync(ctx, rec).FirstErr()).To(Succeed())
	}

	It("Should stamp a group without a timestamp field with the Kafka timestamp",
		func(ctx SpecContext) {
			index, data := createIndexed(ctx, telem.Float64T)
			dev := createDevice(ctx, clusterProperties())
			topic := newTopic()
			dt, t := configure(ctx, kafka.ReadConfig{
				Device:  dev.Key,
				Topic:   topic,
				Records: single(unstampedFields(data)),
			})
			start(ctx, dt, t)
			produceAt(ctx, topic, 10_000, map[string]any{"v": 1.0})
			produceAt(ctx, topic, 11_000, map[string]any{"v": 2.0})
			Expect(floats(expectStored(ctx, data.Key(), 2))).To(Equal([]float64{1, 2}))
			Expect(timestamps(stored(ctx, index.Key()))).
				To(Equal([]telem.TimeStamp{10 * telem.SecondTS, 11 * telem.SecondTS}))
		},
	)

	It("Should move a record that repeats a Kafka timestamp past the last stamp",
		func(ctx SpecContext) {
			index, data := createIndexed(ctx, telem.Float64T)
			dev := createDevice(ctx, clusterProperties())
			topic := newTopic()
			dt, t := configure(ctx, kafka.ReadConfig{
				Device:  dev.Key,
				Topic:   topic,
				Records: single(unstampedFields(data)),
			})
			start(ctx, dt, t)
			produceAt(ctx, topic, 10_000, map[string]any{"v": 1.0})
			produceAt(ctx, topic, 10_000, map[string]any{"v": 2.0})
			Expect(floats(expectStored(ctx, data.Key(), 2))).To(Equal([]float64{1, 2}))
			Expect(timestamps(stored(ctx, index.Key()))).
				To(Equal([]telem.TimeStamp{10 * telem.SecondTS, 10*telem.SecondTS + 1}))
			Expect(taskStatus(ctx, t).Variant).To(Equal(status.VariantSuccess))
		},
	)

	It("Should silently skip the replay of records stamped by Kafka",
		func(ctx SpecContext) {
			index, data := createIndexed(ctx, telem.Float64T)
			dev := createDevice(ctx, clusterProperties())
			topic := newTopic()
			cfg := kafka.ReadConfig{
				Device:  dev.Key,
				Topic:   topic,
				Records: single(unstampedFields(data)),
			}
			dt, t := configure(ctx, cfg)
			start(ctx, dt, t)
			produceAt(ctx, topic, 10_000, map[string]any{"v": 1.0})
			produceAt(ctx, topic, 11_000, map[string]any{"v": 2.0})
			expectStored(ctx, data.Key(), 2)
			Expect(dt.Stop(true)).To(Succeed())

			cfg.Group = "fresh-" + t.Key.String()
			dt2, t2 := configure(ctx, cfg)
			start(ctx, dt2, t2)
			produceAt(ctx, topic, 12_000, map[string]any{"v": 3.0})
			Expect(floats(expectStored(ctx, data.Key(), 3))).
				To(Equal([]float64{1, 2, 3}))
			Expect(timestamps(stored(ctx, index.Key()))).To(Equal([]telem.TimeStamp{
				10 * telem.SecondTS, 11 * telem.SecondTS, 12 * telem.SecondTS,
			}))
			Expect(taskStatus(ctx, t2).Variant).To(Equal(status.VariantSuccess))
		},
	)

	It("Should write records after the start command's context ends",
		func(ctx SpecContext) {
			index, data := createIndexed(ctx, telem.Float64T)
			dev := createDevice(ctx, clusterProperties())
			topic := newTopic()
			dt, t := configure(ctx, kafka.ReadConfig{
				Device:  dev.Key,
				Topic:   topic,
				Records: single(stampedFields(index, data)),
			})
			cmdCtx, cancel := context.WithCancel(ctx)
			start(cmdCtx, dt, t)
			cancel()
			produceJSON(ctx, producer, topic, "", map[string]any{"ts": 10e9, "v": 1.0})
			Expect(floats(expectStored(ctx, data.Key(), 1))).To(Equal([]float64{1}))
		},
	)

	It("Should skip a record lacking a field with a warning", func(ctx SpecContext) {
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
		Expect(stat.Variant).To(Equal(status.VariantWarning))
		// The two records can arrive in one poll or in two.
		Expect(stat.Message).To(HavePrefix("Skipped 1 of "))
		Expect(stat.Description).
			To(Equal("Record has no value at /v (partition 0, offset 0)."))
		Expect(stat.Details.Running).To(BeTrue())
	})

	It("Should clear the warning 5 seconds after the last skipped record",
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
			Eventually(func() status.Variant { return taskStatus(ctx, t).Variant }).
				WithTimeout(15 * time.Second).Should(Equal(status.VariantWarning))
			skipped := time.Now()
			Eventually(func() status.Variant { return taskStatus(ctx, t).Variant }).
				WithTimeout(15 * time.Second).Should(Equal(status.VariantSuccess))
			Expect(time.Since(skipped)).To(BeNumerically(">=", 4*time.Second))
			stat := taskStatus(ctx, t)
			Expect(stat.Message).To(Equal("Started task"))
			Expect(stat.Description).To(BeEmpty())
			Expect(stat.Details.Data).To(BeNil())
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
			// A control handles one request, and a consumer another spec left
			// behind can still fetch, so only a fetch of this topic fails.
			id := topicID(ctx, producer, topic)
			cluster.ControlKey(
				int16(kmsg.Fetch),
				func(req kmsg.Request) (kmsg.Response, error, bool) {
					fetch := req.(*kmsg.FetchRequest)
					ours := slices.ContainsFunc(
						fetch.Topics,
						func(rt kmsg.FetchRequestTopic) bool {
							return rt.TopicID == id || rt.Topic == topic
						},
					)
					if !ours {
						return nil, nil, false
					}
					res := req.ResponseKind().(*kmsg.FetchResponse)
					for _, rt := range fetch.Topics {
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
				WithTimeout(15 * time.Second).Should(Equal("Failed to fetch records"))
			start(ctx, dt, t)
			produceJSON(ctx, producer, topic, "", map[string]any{"ts": 10e9, "v": 1.0})
			Expect(floats(expectStored(ctx, data.Key(), 1))).To(Equal([]float64{1}))
		},
	)

	It("Should resume consuming without duplicates when the cluster comes back",
		func(ctx SpecContext) {
			// The broker persists to dir, so its restart keeps the topic and its ID.
			dir := GinkgoT().TempDir()
			first := MustSucceed(kfake.NewCluster(
				kfake.NumBrokers(1),
				kfake.DataDir(dir),
			))
			DeferCleanup(first.Close)
			addr := first.ListenAddrs()[0]
			index, data := createIndexed(ctx, telem.Float64T)
			dev := createDevice(ctx, kafka.Properties{Brokers: []string{addr}})
			topic := uniqueTopic()
			Expect(first.CreateTopic(topic, 1, nil)).To(Succeed())
			dt, t := configure(ctx, kafka.ReadConfig{
				Device:  dev.Key,
				Topic:   topic,
				Records: single(unstampedFields(data)),
			})
			start(ctx, dt, t)
			produce := func(ms int64, v float64) {
				GinkgoHelper()
				pr := MustSucceed(kgo.NewClient(kgo.SeedBrokers(addr)))
				defer pr.Close()
				Expect(pr.ProduceSync(ctx, &kgo.Record{
					Topic:     topic,
					Value:     MustSucceed(json.Marshal(map[string]any{"v": v})),
					Timestamp: time.UnixMilli(ms),
				}).FirstErr()).To(Succeed())
			}
			// The cluster closes before the offset commits, so the rejoined group
			// replays this record.
			produce(10_000, 1)
			expectStored(ctx, data.Key(), 1)
			first.Close()
			Eventually(func() string { return taskStatus(ctx, t).Message }).
				WithTimeout(15 * time.Second).
				Should(Equal("Lost connection to cluster"))
			stat := taskStatus(ctx, t)
			Expect(stat.Variant).To(Equal(status.VariantWarning))
			Expect(stat.Details.Running).To(BeTrue())
			_, port := MustSucceed2(net.SplitHostPort(addr))
			second := MustSucceed(kfake.NewCluster(
				kfake.Ports(MustSucceed(strconv.Atoi(port))),
				kfake.DataDir(dir),
			))
			DeferCleanup(second.Close)
			produce(11_000, 2)
			Expect(floats(expectStored(ctx, data.Key(), 2))).To(Equal([]float64{1, 2}))
			Expect(timestamps(stored(ctx, index.Key()))).To(Equal(
				[]telem.TimeStamp{10 * telem.SecondTS, 11 * telem.SecondTS},
			))
			Eventually(func() string { return taskStatus(ctx, t).Message }).
				Should(Equal("Reconnected to cluster"))
			stat = taskStatus(ctx, t)
			Expect(stat.Variant).To(Equal(status.VariantSuccess))
			Expect(stat.Details.Running).To(BeTrue())
			// The offsets commit on stop, which needs the cluster cleanup closes.
			Expect(dt.Stop(false)).To(Succeed())
		},
	)

	It("Should report a record skipped in the first poll after reconnecting",
		func(ctx SpecContext) {
			dir := GinkgoT().TempDir()
			first := MustSucceed(kfake.NewCluster(
				kfake.NumBrokers(1),
				kfake.DataDir(dir),
			))
			DeferCleanup(first.Close)
			addr := first.ListenAddrs()[0]
			index, data := createIndexed(ctx, telem.Float64T)
			dev := createDevice(ctx, kafka.Properties{Brokers: []string{addr}})
			topic := uniqueTopic()
			Expect(first.CreateTopic(topic, 1, nil)).To(Succeed())
			dt, t := configure(ctx, kafka.ReadConfig{
				Device:  dev.Key,
				Topic:   topic,
				Records: single(stampedFields(index, data)),
			})
			start(ctx, dt, t)
			produce := func(doc map[string]any) {
				GinkgoHelper()
				pr := MustSucceed(kgo.NewClient(kgo.SeedBrokers(addr)))
				defer pr.Close()
				produceJSON(ctx, pr, topic, "", doc)
			}
			produce(map[string]any{"ts": 5e9})
			Eventually(func() string { return taskStatus(ctx, t).Message }).
				WithTimeout(15 * time.Second).Should(Equal("Skipped 1 of 1 record"))
			first.Close()
			Eventually(func() string { return taskStatus(ctx, t).Message }).
				WithTimeout(15 * time.Second).
				Should(Equal("Lost connection to cluster"))
			_, port := MustSucceed2(net.SplitHostPort(addr))
			second := MustSucceed(kfake.NewCluster(
				kfake.Ports(MustSucceed(strconv.Atoi(port))),
				kfake.DataDir(dir),
			))
			DeferCleanup(second.Close)
			produce(map[string]any{"ts": 11e9})
			produce(map[string]any{"ts": 12e9, "v": 1.0})
			Expect(floats(expectStored(ctx, data.Key(), 1))).To(Equal([]float64{1}))
			// The rejoined group may replay the first skipped record too.
			stat := taskStatus(ctx, t)
			Expect(stat.Message).To(HavePrefix("Skipped"))
			Expect(stat.Description).
				To(HavePrefix("Record has no value at /v (partition 0, offset "))
			Expect(stat.Variant).To(Equal(status.VariantWarning))
			Expect(stat.Details.Running).To(BeTrue())
			// The offsets commit on stop, which needs the cluster cleanup closes.
			Expect(dt.Stop(false)).To(Succeed())
		},
	)

	It("Should count the records it reads and skips", func(ctx SpecContext) {
		index, data := createIndexed(ctx, telem.Float64T)
		dev := createDevice(ctx, clusterProperties())
		topic := newTopic()
		var recs []*kgo.Record
		for range 3 {
			recs = append(recs, &kgo.Record{Topic: topic, Value: []byte(`{"ts": 1}`)})
		}
		Expect(producer.ProduceSync(ctx, recs...).FirstErr()).To(Succeed())
		dt, t := configure(ctx, kafka.ReadConfig{
			Device:  dev.Key,
			Topic:   topic,
			Records: single(stampedFields(index, data)),
		})
		// The first poll can report the skips before the start status is read.
		Expect(dt.Exec(ctx, task.Command{Task: t.Key, Type: "start", Key: "cmd-2"})).
			To(Succeed())
		Eventually(func() string { return taskStatus(ctx, t).Message }).
			WithTimeout(15 * time.Second).Should(Equal("Skipped 3 of 3 records"))
		stat := taskStatus(ctx, t)
		Expect(stat.Description).
			To(Equal("Record has no value at /v (partition 0, offset 2)."))
		Expect(stat.Details.Data).To(HaveKeyWithValue("stats", SatisfyAll(
			HaveKeyWithValue("read", BeNumerically("==", 3)),
			HaveKeyWithValue("skipped", BeNumerically("==", 3)),
		)))
	})

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

	It("Should skip a record that repeats a field's value", func(ctx SpecContext) {
		index, data := createIndexed(ctx, telem.Float64T)
		dev := createDevice(ctx, clusterProperties())
		topic := newTopic()
		dt, t := configure(ctx, kafka.ReadConfig{
			Device:  dev.Key,
			Topic:   topic,
			Records: single(stampedFields(index, data)),
		})
		start(ctx, dt, t)
		rec := &kgo.Record{Topic: topic, Value: []byte(`{"ts":10e9,"v":1,"v":2}`)}
		Expect(producer.ProduceSync(ctx, rec).FirstErr()).To(Succeed())
		Eventually(func() string { return taskStatus(ctx, t).Description }).
			WithTimeout(15 * time.Second).
			Should(ContainSubstring("duplicate value at /v"))
		produceJSON(ctx, producer, topic, "", map[string]any{"ts": 11e9, "v": 3.0})
		Expect(floats(expectStored(ctx, data.Key(), 1))).To(Equal([]float64{3}))
	})

	It("Should read a record that repeats a name no field addresses",
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
			rec := &kgo.Record{
				Topic: topic,
				Value: []byte(`{"ts":10e9,"x":1,"x":2,"v":1.5}`),
			}
			Expect(producer.ProduceSync(ctx, rec).FirstErr()).To(Succeed())
			Expect(floats(expectStored(ctx, data.Key(), 1))).To(Equal([]float64{1.5}))
		},
	)

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

	It("Should not read an array index written with a leading zero",
		func(ctx SpecContext) {
			index, data := createIndexed(ctx, telem.Float64T)
			dev := createDevice(ctx, clusterProperties())
			topic := newTopic()
			fields := stampedFields(index, data)
			fields[1].Pointer = "/vals/01"
			dt, t := configure(ctx, kafka.ReadConfig{
				Device:  dev.Key,
				Topic:   topic,
				Records: single(fields),
			})
			start(ctx, dt, t)
			produceJSON(ctx, producer, topic, "", map[string]any{
				"ts": 10e9, "vals": []any{1.0, 2.5},
			})
			Eventually(func() string { return taskStatus(ctx, t).Description }).
				WithTimeout(15 * time.Second).
				Should(Equal(
					"Record has no value at /vals/01 (partition 0, offset 0).",
				))
		},
	)

	It("Should store a nested JSON value in a JSON channel", func(ctx SpecContext) {
		index, data := createIndexed(ctx, telem.JSONT)
		dev := createDevice(ctx, clusterProperties())
		topic := newTopic()
		fields := stampedFields(index, data)
		fields[1].DataType = telem.JSONT
		dt, t := configure(ctx, kafka.ReadConfig{
			Device:  dev.Key,
			Topic:   topic,
			Records: single(fields),
		})
		start(ctx, dt, t)
		doc := map[string]any{"mode": "auto", "limits": []any{1, 2.5}}
		produceJSON(ctx, producer, topic, "", map[string]any{"ts": 10e9, "v": doc})
		// Members of a marshaled map come in no fixed order.
		got := jsontext.Value(expectStored(ctx, data.Key(), 1).At(0))
		Expect(got.Canonicalize()).To(Succeed())
		Expect(string(got)).To(Equal(`{"limits":[1,2.5],"mode":"auto"}`))
	})

	It("Should skip a record with data after its JSON value", func(ctx SpecContext) {
		index, data := createIndexed(ctx, telem.Float64T)
		dev := createDevice(ctx, clusterProperties())
		topic := newTopic()
		dt, t := configure(ctx, kafka.ReadConfig{
			Device:  dev.Key,
			Topic:   topic,
			Records: single(stampedFields(index, data)),
		})
		start(ctx, dt, t)
		rec := &kgo.Record{Topic: topic, Value: []byte(`{"ts":10e9,"v":1} {}`)}
		Expect(producer.ProduceSync(ctx, rec).FirstErr()).To(Succeed())
		Eventually(func() string { return taskStatus(ctx, t).Description }).
			WithTimeout(15 * time.Second).
			Should(ContainSubstring("Record is not JSON"))
		produceJSON(ctx, producer, topic, "", map[string]any{"ts": 11e9, "v": 2.0})
		Expect(floats(expectStored(ctx, data.Key(), 1))).To(Equal([]float64{2}))
	})

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
		Entry("the smallest integer of the channel's range",
			telem.Int8T, http.TimeFormatUnixNs, nil,
			map[string]any{"ts": 10e9, "v": -128}, telem.NewSeriesV[int8](-128)),
		Entry("an integral float into an integer channel",
			telem.Uint16T, http.TimeFormatUnixNs, nil,
			map[string]any{"ts": 10e9, "v": jsontext.Value("6.5e4")},
			telem.NewSeriesV[uint16](65000)),
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
			Expect(stat.Variant).To(Equal(status.VariantWarning))
			Expect(stat.Details.Running).To(BeTrue())
		},
		Entry("a number where an ISO 8601 timestamp is expected",
			telem.Float64T, http.TimeFormatISO8601, nil,
			map[string]any{"ts": 10, "v": 1.5},
			"Value at /ts: cannot convert 10 to timestamp: numeric values cannot be "+
				"converted with ISO 8601 format"),
		Entry("a malformed ISO 8601 timestamp",
			telem.Float64T, http.TimeFormatISO8601, nil,
			map[string]any{"ts": "yesterday", "v": 1.5},
			`Value at /ts: cannot convert "yesterday" to timestamp: not a valid ISO `+
				"8601 timestamp"),
		Entry("an unknown enum label",
			telem.Uint8T, http.TimeFormatUnixNs, onOff,
			map[string]any{"ts": 10e9, "v": "MAYBE"},
			`Value at /v: cannot convert "MAYBE" to uint8: not a valid number`),
		Entry("an object where a number is expected",
			telem.Float64T, http.TimeFormatUnixNs, nil,
			map[string]any{"ts": 10e9, "v": map[string]any{"x": 1}},
			`Value at /v: cannot convert {"x":1} to float64`),
		Entry("a null where a number is expected",
			telem.Float64T, http.TimeFormatUnixNs, nil,
			map[string]any{"ts": 10e9, "v": nil},
			"Value at /v: cannot convert null to float64"),
		Entry(
			"a number where a UUID is expected",
			telem.UUIDT,
			http.TimeFormatUnixNs,
			nil,
			map[string]any{
				"ts": 10e9,
				"v":  1,
			},
			"Value at /v: cannot convert 1 to uuid",
		),
		Entry("a malformed UUID",
			telem.UUIDT, http.TimeFormatUnixNs, nil,
			map[string]any{"ts": 10e9, "v": "not-a-uuid"},
			`Value at /v: cannot convert "not-a-uuid" to uuid: not a valid UUID`),
		Entry("an integer beyond the channel's range",
			telem.Uint8T, http.TimeFormatUnixNs, nil,
			map[string]any{"ts": 10e9, "v": 256},
			"Value at /v: cannot convert 256 to uint8: value is out of bounds"),
		Entry("a negative integer into an unsigned channel",
			telem.Uint32T, http.TimeFormatUnixNs, nil,
			map[string]any{"ts": 10e9, "v": -1},
			"Value at /v: cannot convert -1 to uint32: value is out of bounds"),
		Entry(
			"a fraction into an integer channel",
			telem.Int32T,
			http.TimeFormatUnixNs,
			nil,
			map[string]any{"ts": 10e9, "v": 1.5},
			"Value at /v: cannot convert 1.5 to int32: value has a fractional "+
				"component",
		),
		Entry("a float beyond the integer channel's range",
			telem.Int64T, http.TimeFormatUnixNs, nil,
			map[string]any{"ts": 10e9, "v": jsontext.Value("1e19")},
			"Value at /v: cannot convert 1e19 to int64: value is out of bounds"),
		Entry("a number beyond float32",
			telem.Float32T, http.TimeFormatUnixNs, nil,
			map[string]any{"ts": 10e9, "v": 1e39},
			"Value at /v: cannot convert 1e+39 to float32: value is out of bounds"),
		Entry("an enum value beyond the channel's range",
			telem.Int8T, http.TimeFormatUnixNs,
			[]http.EnumEntry{{Label: "BIG", Value: 200}},
			map[string]any{"ts": 10e9, "v": "BIG"},
			`Value at /v: cannot convert "BIG" to int8: value is out of bounds`),
		Entry("an integer timestamp beyond the nanosecond range",
			telem.Float64T, http.TimeFormatUnixMs, nil,
			map[string]any{"ts": int64(1_000_000_000_000_000_000), "v": 1.5},
			"Value at /ts: cannot convert 1000000000000000000 to timestamp: value is "+
				"out of bounds"),
		Entry("a floating-point timestamp beyond the nanosecond range",
			telem.Float64T, http.TimeFormatUnixSec, nil,
			map[string]any{"ts": jsontext.Value("1e13"), "v": 1.5},
			"Value at /ts: cannot convert 1e13 to timestamp: value is out of bounds"),
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
			}, `field "v": pointer must not be empty`),
		Entry("a pointer without a leading slash",
			func(_ context.Context, _, data channel.Channel) []kafka.ReadField {
				return []kafka.ReadField{{
					Key:      "v",
					Channel:  data.Key(),
					Pointer:  "v",
					DataType: telem.Float64T,
				}}
			}, `field "v": json pointer "v" must be empty or start with "/"`),
		Entry("a bytes channel",
			func(ctx context.Context, _, _ channel.Channel) []kafka.ReadField {
				_, data := createIndexed(ctx, telem.BytesT)
				return []kafka.ReadField{{
					Key:      "v",
					Channel:  data.Key(),
					Pointer:  "/v",
					DataType: telem.BytesT,
				}}
			}, `field "v": cannot convert a JSON value to bytes`),
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

	It("Should reject two records whose channels share one index",
		func(ctx SpecContext) {
			index, data := createIndexed(ctx, telem.Float64T)
			other := channel.Channel{
				Name:       "other_" + data.Name,
				DataType:   telem.Float64T,
				LocalIndex: index.LocalKey,
			}
			Expect(channelSvc.NewWriter(nil).Create(ctx, &other)).To(Succeed())
			dev := createDevice(ctx, clusterProperties())
			field := func(key string, ch channel.Channel) []kafka.ReadField {
				return []kafka.ReadField{{
					Key: key, Channel: ch.Key(), Pointer: "/" + key,
					DataType: telem.Float64T,
				}}
			}
			cfg := kafka.ReadConfig{
				Device: dev.Key,
				Topic:  "t",
				Records: []kafka.ReadRecord{
					{Key: "r1", RecordKey: "a", Fields: field("v", data)},
					{Key: "r2", RecordKey: "b", Fields: field("w", other)},
				},
			}
			Expect(f.ConfigureTask(ctx, newTask(kafka.ReadTaskType, cfg), "cmd-1")).
				Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(
					ContainSubstring("channels on one index belong in one record"),
				),
			))
		},
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
