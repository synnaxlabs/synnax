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
	"slices"
	"sync"
	"time"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/framer/iterator"
	"github.com/synnaxlabs/synnax/pkg/service/http"
	"github.com/synnaxlabs/synnax/pkg/service/kafka"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	. "github.com/synnaxlabs/x/io/fs/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/hookkv"
	"github.com/synnaxlabs/x/validate"
	"github.com/twmb/franz-go/pkg/kadm"
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
		t := newTask(ctx, kafka.ReadTaskType, cfg)
		dt := MustSucceed(f.ConfigureTask(ctx, t, "cmd-1"))
		DeferCleanup(func() { Expect(dt.Stop(false)).To(Succeed()) })
		return dt, t
	}

	// start starts the task. The first poll can already have replaced the start status
	// with a skip, which keeps the task running.
	start := func(ctx context.Context, dt driver.Task, t task.Task) {
		GinkgoHelper()
		Expect(dt.Exec(ctx, task.Command{Task: t.Key, Type: "start", Key: "cmd-2"})).
			To(Succeed())
		Expect(taskStatus(ctx, t).Details.Running).To(BeTrue())
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

	// expectStored waits until key holds n samples.
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

	It("Should skip a later poll's record stamped by Kafka before the last stamp",
		func(ctx SpecContext) {
			index, data := createIndexed(ctx, telem.Float64T)
			dev := createDevice(ctx, clusterProperties())
			topic := uniqueTopic()
			Expect(cluster.CreateTopic(topic, 2, nil)).To(Succeed())
			partitioned := newKafkaClient(
				kgo.RecordPartitioner(kgo.ManualPartitioner()),
			)
			produce := func(partition int32, ms int64, value float64) {
				GinkgoHelper()
				Expect(partitioned.ProduceSync(ctx, &kgo.Record{
					Topic:     topic,
					Partition: partition,
					Value:     MustSucceed(json.Marshal(map[string]any{"v": value})),
					Timestamp: time.UnixMilli(ms),
				}).FirstErr()).To(Succeed())
			}
			dt, t := configure(ctx, kafka.ReadConfig{
				Device:  dev.Key,
				Topic:   topic,
				Records: single(unstampedFields(data)),
			})
			start(ctx, dt, t)
			produce(0, 10_000, 1)
			expectStored(ctx, data.Key(), 1)
			produce(1, 9_000, 2)
			Eventually(func() string { return taskStatus(ctx, t).Description }).
				WithTimeout(15 * time.Second).
				Should(ContainSubstring("is not after the last written"))
			produce(1, 10_000, 3)
			Expect(floats(expectStored(ctx, data.Key(), 2))).To(Equal([]float64{1, 3}))
			Expect(timestamps(stored(ctx, index.Key()))).
				To(Equal([]telem.TimeStamp{10 * telem.SecondTS, 10*telem.SecondTS + 1}))
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

	// holdWrite holds the first write to fs until release is called or the spec ends.
	// The returned channel closes once that write is held.
	holdWrite := func(fs *FaultyFS) (<-chan struct{}, func()) {
		GinkgoHelper()
		var (
			writing = make(chan struct{})
			resume  = make(chan struct{})
			hold    = sync.OnceFunc(func() {
				GinkgoHelper()
				defer GinkgoRecover()
				close(writing)
				Eventually(resume).WithTimeout(40 * time.Second).Should(BeClosed())
			})
			release = sync.OnceFunc(func() { close(resume) })
		)
		fs.SetOptions(WithBeforeWrite(hold))
		DeferCleanup(func() { fs.SetOptions() })
		DeferCleanup(release)
		return writing, release
	}

	It(
		"Should finish the write in progress when the task stops",
		func(ctx SpecContext) {
			index, data := createIndexed(ctx, telem.Float64T)
			dev := createDevice(ctx, clusterProperties())
			topic := newTopic()
			group := "group-" + topic
			dt, t := configure(ctx, kafka.ReadConfig{
				Device:  dev.Key,
				Topic:   topic,
				Group:   group,
				Records: single(stampedFields(index, data)),
			})
			start(ctx, dt, t)
			writing, release := holdWrite(faultyFS)
			produceJSON(ctx, producer, topic, "", map[string]any{"ts": 10e9, "v": 2.5})
			Eventually(writing).WithTimeout(15 * time.Second).Should(BeClosed())
			stopped := make(chan error, 1)
			go func() { stopped <- dt.Stop(false) }()
			Consistently(
				stopped,
			).WithTimeout(100 * time.Millisecond).
				ShouldNot(Receive())
			release()
			Eventually(stopped).WithTimeout(15 * time.Second).Should(Receive(Succeed()))
			Expect(floats(stored(ctx, data.Key()))).To(Equal([]float64{2.5}))
			res := MustSucceed(kadm.NewClient(producer).
				FetchOffsetsForTopics(ctx, group, topic))
			Expect(MustBeOk(res.Lookup(topic, 0)).At).To(Equal(int64(1)))
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

	// failFetch makes the next fetch of topic fail. A control handles one request, and
	// a consumer another spec left behind can still fetch, so only a fetch of this
	// topic
	// fails. A fetch in a session names no topics, so the control resets the session to
	// make the consumer name them.
	failFetch := func(ctx context.Context, topic string) {
		GinkgoHelper()
		id := topicID(ctx, producer, topic)
		cluster.ControlKey(
			int16(kmsg.Fetch),
			func(req kmsg.Request) (kmsg.Response, error, bool) {
				fetch := req.(*kmsg.FetchRequest)
				if fetch.SessionEpoch > 0 && len(fetch.Topics) == 0 {
					cluster.KeepControl()
					res := req.ResponseKind().(*kmsg.FetchResponse)
					res.ErrorCode = kerr.FetchSessionIDNotFound.Code
					return res, nil, true
				}
				ours := slices.ContainsFunc(
					fetch.Topics,
					func(rt kmsg.FetchRequestTopic) bool {
						return rt.TopicID == id || rt.Topic == topic
					},
				)
				if !ours {
					return nil, nil, false
				}
				cluster.DropControl()
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
	}

	// failWrite makes the task's next write fail with a record stamped at the last
	// representable time, which leaves no room for a domain end.
	failWrite := func(ctx context.Context, topic string) {
		GinkgoHelper()
		doc := map[string]any{"ts": int64(math.MaxInt64), "v": 9.0}
		produceJSON(ctx, producer, topic, "", doc)
	}

	// dropRecords removes the records of topic, so a consumer that starts next does not
	// replay one that failed its write.
	dropRecords := func(ctx context.Context, topic string) {
		GinkgoHelper()
		var offsets kadm.Offsets
		offsets.AddOffset(topic, 0, -1, -1)
		res := MustSucceed(kadm.NewClient(producer).DeleteRecords(ctx, offsets))
		Expect(res.Error()).To(Succeed())
	}

	DescribeTable("Should restart a consumer that fails while start is called",
		func(ctx SpecContext, fail func(context.Context, string)) {
			index, data := createIndexed(ctx, telem.Float64T)
			dev := createDevice(ctx, clusterProperties())
			topic := newTopic()
			dt, t := configure(ctx, kafka.ReadConfig{
				Device:  dev.Key,
				Topic:   topic,
				Records: single(stampedFields(index, data)),
			})
			// Cleanup runs in reverse, so a failed spec opens the gate before the stop
			// waits on the consumer that holds it.
			held, gate := make(chan struct{}), make(chan struct{})
			release := sync.OnceFunc(func() { close(gate) })
			DeferCleanup(release)
			start(ctx, dt, t)
			DeferCleanup(hookedKV.HookKey(
				[]byte(taskStatus(ctx, t).Key),
				hookkv.Hold(held, gate),
			))
			fail(ctx, topic)
			Eventually(held).WithTimeout(15 * time.Second).Should(BeClosed())
			Expect(taskStatus(ctx, t).Message).To(Equal("Started task"))
			dropRecords(ctx, topic)
			started := make(chan error, 1)
			go func() {
				started <- dt.Exec(
					ctx,
					task.Command{Task: t.Key, Type: "start", Key: "cmd-3"},
				)
			}()
			Consistently(started).ShouldNot(Receive())
			release()
			Eventually(started).WithTimeout(15 * time.Second).Should(Receive(BeNil()))
			stat := taskStatus(ctx, t)
			Expect(stat.Variant).To(Equal(status.VariantSuccess))
			Expect(stat.Message).To(Equal("Started task"))
			Expect(stat.Details.Cmd).To(Equal("cmd-3"))
			Expect(stat.Details.Running).To(BeTrue())
			produceJSON(ctx, producer, topic, "", map[string]any{"ts": 30e9, "v": 3.0})
			Expect(floats(expectStored(ctx, data.Key(), 1))).To(Equal([]float64{3}))
		},
		Entry("failed fetch", failFetch),
		Entry("failed write", failWrite),
	)

	DescribeTable("Should release its channels when the consumer fails",
		func(ctx SpecContext, fail func(context.Context, string)) {
			index, data := createIndexed(ctx, telem.Float64T)
			dev := createDevice(ctx, clusterProperties())
			topic := newTopic()
			dt, t := configure(ctx, kafka.ReadConfig{
				Device:  dev.Key,
				Topic:   topic,
				Records: single(stampedFields(index, data)),
			})
			start(ctx, dt, t)
			fail(ctx, topic)
			Eventually(func() error {
				w, err := framerSvc.OpenWriter(ctx, framer.WriterConfig{
					Keys:              channel.Keys{index.Key(), data.Key()},
					Start:             telem.Now(),
					ErrOnUnauthorized: new(true),
				})
				if err != nil {
					return err
				}
				return w.Close()
			}).WithTimeout(15 * time.Second).Should(Succeed())
			dropRecords(ctx, topic)
			start(ctx, dt, t)
		},
		Entry("failed fetch", failFetch),
		Entry("failed write", failWrite),
	)

	It("Should consume again when started after a failed write",
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
			failWrite(ctx, topic)
			Eventually(func(g Gomega) {
				stat := taskStatus(ctx, t)
				g.Expect(stat.Variant).To(Equal(status.VariantError))
				g.Expect(stat.Message).To(Equal("Failed to process records"))
				g.Expect(stat.Details.Running).To(BeFalse())
			}).WithTimeout(15 * time.Second).Should(Succeed())
			dropRecords(ctx, topic)
			start(ctx, dt, t)
			produceJSON(ctx, producer, topic, "", map[string]any{"ts": 10e9, "v": 1.0})
			Expect(floats(expectStored(ctx, data.Key(), 1))).To(Equal([]float64{1}))
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
			failFetch(ctx, topic)
			Eventually(func() string { return taskStatus(ctx, t).Message }).
				WithTimeout(15 * time.Second).Should(Equal("Failed to fetch records"))
			start(ctx, dt, t)
			produceJSON(ctx, producer, topic, "", map[string]any{"ts": 10e9, "v": 1.0})
			Expect(floats(expectStored(ctx, data.Key(), 1))).To(Equal([]float64{1}))
		},
	)

	It("Should resume consuming without duplicates when the cluster comes back",
		func(ctx SpecContext) {
			broker := MustSucceed(kfake.NewCluster(kfake.NumBrokers(1)))
			DeferCleanup(broker.Close)
			addr := broker.ListenAddrs()[0]
			index, data := createIndexed(ctx, telem.Float64T)
			dev := createDevice(ctx, kafka.Properties{Brokers: []string{addr}})
			topic := uniqueTopic()
			Expect(broker.CreateTopic(topic, 1, nil)).To(Succeed())
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
			// The outage starts before the offset commits, so the rejoined group
			// replays this record.
			produce(10_000, 1)
			expectStored(ctx, data.Key(), 1)
			rejoined := broker.Fault(kfake.Fault{
				Keys:    []kmsg.Key{kmsg.SyncGroup},
				Observe: true,
				Count:   -1,
			})
			restore := cutConnections(broker)
			Eventually(func() string { return taskStatus(ctx, t).Message }).
				WithTimeout(15 * time.Second).
				Should(Equal("Lost connection to cluster"))
			stat := taskStatus(ctx, t)
			Expect(stat.Variant).To(Equal(status.VariantWarning))
			Expect(stat.Details.Running).To(BeTrue())
			restore()
			Eventually(rejoined.Hits).
				WithTimeout(15 * time.Second).Should(BeNumerically(">", 0))
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
			broker := MustSucceed(kfake.NewCluster(kfake.NumBrokers(1)))
			DeferCleanup(broker.Close)
			addr := broker.ListenAddrs()[0]
			index, data := createIndexed(ctx, telem.Float64T)
			dev := createDevice(ctx, kafka.Properties{Brokers: []string{addr}})
			topic := uniqueTopic()
			Expect(broker.CreateTopic(topic, 1, nil)).To(Succeed())
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
			restore := cutConnections(broker)
			Eventually(func() string { return taskStatus(ctx, t).Message }).
				WithTimeout(15 * time.Second).
				Should(Equal("Lost connection to cluster"))
			restore()
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
		start(ctx, dt, t)
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
			t := newTask(ctx, kafka.ReadTaskType, kafka.ReadConfig{
				Device:      dev.Key,
				Topic:       newTopic(),
				StartOffset: kafka.StartOffsetNone,
				Records:     single(stampedFields(index, data)),
			})
			dt := MustSucceed(f.ConfigureTask(ctx, t, "cmd-1"))
			DeferCleanup(func() { Expect(dt.Stop(false)).To(Succeed()) })
			Expect(dt.Exec(ctx, task.Command{
				Task: t.Key, Type: "start", Key: "cmd-2",
			})).To(Succeed())
			stat := taskStatus(ctx, t)
			Expect(stat.Variant).To(Equal(status.VariantError))
			Expect(stat.Description).To(ContainSubstring("no committed offset"))
			Expect(stat.Details.Running).To(BeFalse())
		},
	)

	It("Should return the error from Exec for a start it could not answer",
		func(ctx SpecContext) {
			index, data := createIndexed(ctx, telem.Float64T)
			dev := createDevice(ctx, clusterProperties())
			t := newTask(ctx, kafka.ReadTaskType, kafka.ReadConfig{
				Device:      dev.Key,
				Topic:       newTopic(),
				StartOffset: kafka.StartOffsetEarliest,
				Records:     single(stampedFields(index, data)),
			})
			dt := MustSucceed(f.ConfigureTask(ctx, t, "cmd-1"))
			DeferCleanup(func() { Expect(dt.Stop(false)).To(Succeed()) })
			Expect(dt.Exec(hookkv.WithHook(ctx, refuse), task.Command{
				Task: t.Key, Type: "start", Key: "cmd-2",
			})).To(MatchError(errCommit))
			w := MustSucceed(framerSvc.OpenWriter(ctx, framer.WriterConfig{
				Keys:              channel.Keys{index.Key(), data.Key()},
				Start:             telem.Now(),
				ErrOnUnauthorized: new(true),
			}))
			Expect(w.Close()).To(Succeed())
		},
	)

	It("Should return no error from Exec for a start answered before ctx ended",
		func(ctx SpecContext) {
			index, data := createIndexed(ctx, telem.Float64T)
			dev := createDevice(ctx, clusterProperties())
			t := newTask(ctx, kafka.ReadTaskType, kafka.ReadConfig{
				Device:      dev.Key,
				Topic:       newTopic(),
				StartOffset: kafka.StartOffsetNone,
				Records:     single(stampedFields(index, data)),
			})
			cmdCtx, cancel := context.WithCancel(ctx)
			cmdCtx = hookkv.WithHook(cmdCtx, func(commit func() error) error {
				err := commit()
				cancel()
				return err
			})
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

	It("Should write one poll's partitions in timestamp order", func(ctx SpecContext) {
		index, data := createIndexed(ctx, telem.Float64T)
		dev := createDevice(ctx, clusterProperties())
		topic := uniqueTopic()
		Expect(cluster.CreateTopic(topic, 2, nil)).To(Succeed())
		partitioned := newKafkaClient(kgo.RecordPartitioner(kgo.ManualPartitioner()))
		record := func(partition int32, stamp, value float64) *kgo.Record {
			GinkgoHelper()
			doc := map[string]any{"ts": stamp, "v": value}
			return &kgo.Record{
				Topic:     topic,
				Partition: partition,
				Value:     MustSucceed(json.Marshal(doc)),
			}
		}
		// Producing before the start puts both partitions in the task's first poll.
		Expect(partitioned.ProduceSync(ctx,
			record(0, 10e9, 1),
			record(0, 12e9, 3),
			record(1, 11e9, 2),
			record(1, 13e9, 4),
		).FirstErr()).To(Succeed())
		// A consumer fetches as soon as one partition's offset loads, so only a fetch
		// of both is served. A response with no session makes the next fetch name them
		// all.
		id := topicID(ctx, producer, topic)
		cluster.ControlKey(
			int16(kmsg.Fetch),
			func(req kmsg.Request) (kmsg.Response, error, bool) {
				fetch := req.(*kmsg.FetchRequest)
				i := slices.IndexFunc(
					fetch.Topics,
					func(rt kmsg.FetchRequestTopic) bool {
						return rt.TopicID == id || rt.Topic == topic
					},
				)
				if i == -1 {
					return nil, nil, false
				}
				if len(fetch.Topics[i].Partitions) == 2 {
					cluster.DropControl()
					return nil, nil, false
				}
				cluster.KeepControl()
				return req.ResponseKind(), nil, true
			},
		)
		dt, t := configure(ctx, kafka.ReadConfig{
			Device:  dev.Key,
			Topic:   topic,
			Records: single(stampedFields(index, data)),
		})
		start(ctx, dt, t)
		Expect(floats(expectStored(ctx, data.Key(), 4))).
			To(Equal([]float64{1, 2, 3, 4}))
		Expect(timestamps(stored(ctx, index.Key()))).To(Equal([]telem.TimeStamp{
			10 * telem.SecondTS,
			11 * telem.SecondTS,
			12 * telem.SecondTS,
			13 * telem.SecondTS,
		}))
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

	Context("Records that convert differently within one poll", func() {
		// startText starts a task that reads /s into a string channel and /v into a
		// float channel on one index, after records are produced, so they share a poll.
		startText := func(
			ctx context.Context,
			records ...*kgo.Record,
		) (data, text channel.Channel) {
			GinkgoHelper()
			var index channel.Channel
			index, data = createIndexed(ctx, telem.Float64T)
			text = channel.Channel{
				Name:       data.Name + "_text",
				DataType:   telem.StringT,
				LocalIndex: index.LocalKey,
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return channelSvc.NewWriter(tx).Create(ctx, &text)
			})).To(Succeed())
			fields := stampedFields(index, data)
			fields = slices.Insert(fields, 1, kafka.ReadField{
				Key:      "s",
				Channel:  text.Key(),
				Pointer:  "/s",
				DataType: telem.StringT,
			})
			dev := createDevice(ctx, clusterProperties())
			topic := newTopic()
			for _, rec := range records {
				rec.Topic = topic
			}
			Expect(producer.ProduceSync(ctx, records...).FirstErr()).To(Succeed())
			dt, t := configure(ctx, kafka.ReadConfig{
				Device:  dev.Key,
				Topic:   topic,
				Records: single(fields),
			})
			start(ctx, dt, t)
			return data, text
		}

		texts := func(s telem.Series) []string {
			var out []string
			for sample := range s.Samples() {
				out = append(out, string(sample))
			}
			return out
		}

		It("Should drop every sample of a record that fails on a later field",
			func(ctx SpecContext) {
				data, text := startText(ctx,
					&kgo.Record{Value: []byte(`{"ts":10e9,"s":"first","v":1}`)},
					&kgo.Record{Value: []byte(`{"ts":11e9,"s":"bad","v":"none"}`)},
					&kgo.Record{Value: []byte(`{"ts":12e9,"s":"last","v":3}`)},
				)
				Expect(
					floats(expectStored(ctx, data.Key(), 2)),
				).To(Equal([]float64{1, 3}))
				Expect(texts(expectStored(ctx, text.Key(), 2))).
					To(Equal([]string{"first", "last"}))
			},
		)

		It("Should reorder samples of varying size by timestamp",
			func(ctx SpecContext) {
				data, text := startText(ctx,
					&kgo.Record{Value: []byte(`{"ts":12e9,"s":"c","v":3}`)},
					&kgo.Record{Value: []byte(`{"ts":10e9,"s":"longest","v":1}`)},
					&kgo.Record{Value: []byte(`{"ts":11e9,"s":"\u00e9\n","v":2}`)},
				)
				Expect(floats(expectStored(ctx, data.Key(), 3))).
					To(Equal([]float64{1, 2, 3}))
				Expect(texts(expectStored(ctx, text.Key(), 3))).
					To(Equal([]string{"longest", "é\n", "c"}))
			},
		)
	})

	It("Should match an object member whose name holds an escape",
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
			rec := &kgo.Record{Topic: topic, Value: []byte(`{"ts":10e9,"\u0076":2.5}`)}
			Expect(producer.ProduceSync(ctx, rec).FirstErr()).To(Succeed())
			Expect(floats(expectStored(ctx, data.Key(), 1))).To(Equal([]float64{2.5}))
		},
	)

	It("Should read a value inside another field's value", func(ctx SpecContext) {
		index, data := createIndexed(ctx, telem.JSONT)
		inner := channel.Channel{
			Name:       data.Name + "_inner",
			DataType:   telem.Float64T,
			LocalIndex: index.LocalKey,
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channelSvc.NewWriter(tx).Create(ctx, &inner)
		})).To(Succeed())
		fields := stampedFields(index, data)
		fields[1].DataType = telem.JSONT
		fields = append(fields, kafka.ReadField{
			Key:      "inner",
			Channel:  inner.Key(),
			Pointer:  "/v/a/b",
			DataType: telem.Float64T,
		})
		dev := createDevice(ctx, clusterProperties())
		topic := newTopic()
		dt, t := configure(ctx, kafka.ReadConfig{
			Device:  dev.Key,
			Topic:   topic,
			Records: single(fields),
		})
		start(ctx, dt, t)
		rec := &kgo.Record{
			Topic: topic,
			Value: []byte(`{"ts":10e9,"v": { "a": { "b": 7.5 } } }`),
		}
		Expect(producer.ProduceSync(ctx, rec).FirstErr()).To(Succeed())
		Expect(floats(expectStored(ctx, inner.Key(), 1))).To(Equal([]float64{7.5}))
		Expect(string(expectStored(ctx, data.Key(), 1).At(0))).
			To(Equal(`{"a":{"b":7.5}}`))
	})

	// startSingle starts a task reading /v of each record into a channel of dt, and
	// produces body.
	startSingle := func(
		ctx context.Context,
		dt telem.DataType,
		body string,
	) (data channel.Channel, t task.Task) {
		GinkgoHelper()
		var index channel.Channel
		index, data = createIndexed(ctx, dt)
		dev := createDevice(ctx, clusterProperties())
		topic := newTopic()
		fields := stampedFields(index, data)
		fields[1].DataType = dt
		var dtask driver.Task
		dtask, t = configure(ctx, kafka.ReadConfig{
			Device:  dev.Key,
			Topic:   topic,
			Records: single(fields),
		})
		start(ctx, dtask, t)
		rec := &kgo.Record{Topic: topic, Value: []byte(body)}
		Expect(producer.ProduceSync(ctx, rec).FirstErr()).To(Succeed())
		return data, t
	}

	DescribeTable("Should store a JSON value as the record spells it",
		func(ctx SpecContext, value, expected string) {
			data, _ := startSingle(ctx, telem.JSONT, `{"ts":10e9,"v":`+value+`}`)
			Expect(string(expectStored(ctx, data.Key(), 1).At(0))).To(Equal(expected))
		},
		Entry("a fraction with a trailing zero", `1.0`, `1.0`),
		Entry("a number past the range of a float", `1e999`, `1e999`),
		Entry("members out of order", `{"z":1,"a":2}`, `{"z":1,"a":2}`),
		Entry("spaces inside a value", `{ "a" : [ 1 , 2 ] }`, `{"a":[1,2]}`),
	)

	It("Should skip a number past the range of a numeric channel",
		func(ctx SpecContext) {
			_, t := startSingle(ctx, telem.Float64T, `{"ts":10e9,"v":1e999}`)
			Eventually(func() string { return taskStatus(ctx, t).Description }).
				WithTimeout(15 * time.Second).
				Should(ContainSubstring("Value at /v: "))
			Expect(
				taskStatus(ctx, t).Description,
			).To(ContainSubstring("invalid number"))
		},
	)

	DescribeTable(
		"Should encode a number in an integer channel",
		func(
			ctx SpecContext,
			dt telem.DataType,
			value string,
			expected telem.Series,
		) {
			body := `{"ts":10e9,"v":` + value + `}`
			data, _ := startSingle(ctx, dt, body)
			Expect(expectStored(ctx, data.Key(), 1).Data).To(Equal(expected.Data))
		},
		Entry("a negative int8", telem.Int8T, `-2`, telem.NewSeriesV[int8](-2)),
		Entry("a negative int16", telem.Int16T, `-300`, telem.NewSeriesV[int16](-300)),
		Entry(
			"a negative int32",
			telem.Int32T,
			`-70000`,
			telem.NewSeriesV[int32](-70000),
		),
		Entry("a negative int64", telem.Int64T, `-5000000000`,
			telem.NewSeriesV[int64](-5000000000)),
		Entry("the largest uint8", telem.Uint8T, `255`, telem.NewSeriesV[uint8](255)),
		Entry("the largest uint16", telem.Uint16T, `65535`,
			telem.NewSeriesV[uint16](65535)),
		Entry("the largest uint32", telem.Uint32T, `4294967295`,
			telem.NewSeriesV[uint32](4294967295)),
		Entry("the largest uint64", telem.Uint64T, `18446744073709551615`,
			telem.NewSeriesV[uint64](18446744073709551615)),
		Entry("a negative whole float", telem.Int16T, `-3.0`,
			telem.NewSeriesV[int16](-3)),
		Entry("a float past the range of an int64", telem.Uint64T, `1e19`,
			telem.NewSeriesV[uint64](10000000000000000000)),
	)

	It("Should skip a JSON value that repeats a member", func(ctx SpecContext) {
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
		rec := &kgo.Record{Topic: topic, Value: []byte(`{"ts":10e9,"v":{"a":1,"a":2}}`)}
		Expect(producer.ProduceSync(ctx, rec).FirstErr()).To(Succeed())
		Eventually(func() string { return taskStatus(ctx, t).Description }).
			WithTimeout(15 * time.Second).
			Should(ContainSubstring("duplicate object member name"))
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
		Entry("bytes from base64",
			telem.BytesT, http.TimeFormatUnixNs, nil,
			map[string]any{"ts": 10e9, "v": "AP8/"},
			telem.NewSeriesV([]byte{0x00, 0xff, 0x3f})),
		Entry("empty bytes from empty base64",
			telem.BytesT, http.TimeFormatUnixNs, nil,
			map[string]any{"ts": 10e9, "v": ""}, telem.NewSeriesV([]byte{})),
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
			map[string]any{"ts": 10, "v": 1.5}, "expected an ISO 8601 string"),
		Entry("a malformed ISO 8601 timestamp",
			telem.Float64T, http.TimeFormatISO8601, nil,
			map[string]any{"ts": "yesterday", "v": 1.5},
			"invalid ISO 8601 timestamp"),
		Entry("a boolean where a numeric timestamp is expected",
			telem.Float64T, http.TimeFormatUnixNs, nil,
			map[string]any{"ts": true, "v": 1.5}, "expected a number, got a boolean"),
		Entry("a string that holds no number where a numeric timestamp is expected",
			telem.Float64T, http.TimeFormatUnixNs, nil,
			map[string]any{"ts": "soon", "v": 1.5}, `invalid number "soon"`),
		Entry("an unknown enum label",
			telem.Uint8T, http.TimeFormatUnixNs, onOff,
			map[string]any{"ts": 10e9, "v": "MAYBE"}, "unknown enum label"),
		Entry("an object where a number is expected",
			telem.Float64T, http.TimeFormatUnixNs, nil,
			map[string]any{"ts": 10e9, "v": map[string]any{"x": 1}},
			"expected a number or boolean"),
		Entry("a null where a number is expected",
			telem.Float64T, http.TimeFormatUnixNs, nil,
			map[string]any{"ts": 10e9, "v": nil}, "expected a number or boolean"),
		Entry("a number where a string is expected",
			telem.StringT, http.TimeFormatUnixNs, nil,
			map[string]any{"ts": 10e9, "v": 1}, "expected a string"),
		Entry("a number where a UUID is expected",
			telem.UUIDT, http.TimeFormatUnixNs, nil,
			map[string]any{"ts": 10e9, "v": 1}, "expected a UUID string"),
		Entry("a malformed UUID",
			telem.UUIDT, http.TimeFormatUnixNs, nil,
			map[string]any{"ts": 10e9, "v": "not-a-uuid"}, "invalid UUID"),
		Entry("text that is not base64",
			telem.BytesT, http.TimeFormatUnixNs, nil,
			map[string]any{"ts": 10e9, "v": "a*b="}, "illegal base64 data"),
		Entry("an integer beyond the channel's range",
			telem.Uint8T, http.TimeFormatUnixNs, nil,
			map[string]any{"ts": 10e9, "v": 256}, "256 is out of range for uint8"),
		Entry("a negative integer into an unsigned channel",
			telem.Uint32T, http.TimeFormatUnixNs, nil,
			map[string]any{"ts": 10e9, "v": -1}, "-1 is out of range for uint32"),
		Entry("a fraction into an integer channel",
			telem.Int32T, http.TimeFormatUnixNs, nil,
			map[string]any{"ts": 10e9, "v": 1.5}, "1.5 is not an integer"),
		Entry("an exponent beyond the integer channel's range",
			telem.Int64T, http.TimeFormatUnixNs, nil,
			map[string]any{"ts": 10e9, "v": jsontext.Value("1e19")},
			"10000000000000000000 is out of range for int64"),
		Entry("a number beyond float32",
			telem.Float32T, http.TimeFormatUnixNs, nil,
			map[string]any{"ts": 10e9, "v": 1e39}, "1e+39 overflows float32"),
		Entry("an integer timestamp beyond the nanosecond range",
			telem.Float64T, http.TimeFormatUnixMs, nil,
			map[string]any{"ts": int64(1_000_000_000_000_000_000), "v": 1.5},
			"1000000000000000000 is out of range for a unix_ms timestamp"),
		Entry("a floating-point timestamp beyond the nanosecond range",
			telem.Float64T, http.TimeFormatUnixSec, nil,
			map[string]any{"ts": jsontext.Value("1e13"), "v": 1.5},
			"1e13 is out of range for a unix_sec timestamp"),
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
			Expect(
				f.ConfigureTask(ctx, newTask(ctx, kafka.ReadTaskType, cfg), "cmd-1"),
			).
				Error().
				To(SatisfyAll(
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
		Entry("an enum value beyond the channel's range",
			func(ctx context.Context, _, _ channel.Channel) []kafka.ReadField {
				_, data := createIndexed(ctx, telem.Int8T)
				return []kafka.ReadField{{
					Key:        "v",
					Channel:    data.Key(),
					Pointer:    "/v",
					DataType:   telem.Int8T,
					EnumValues: []http.EnumEntry{{Label: "BIG", Value: 200}},
				}}
			}, `enum label "BIG": 200 is out of range for int8`),
		Entry("an enum value on a string channel",
			func(ctx context.Context, _, _ channel.Channel) []kafka.ReadField {
				_, data := createIndexed(ctx, telem.StringT)
				return []kafka.ReadField{{
					Key:        "v",
					Channel:    data.Key(),
					Pointer:    "/v",
					DataType:   telem.StringT,
					EnumValues: []http.EnumEntry{{Label: "ON", Value: 1}},
				}}
			}, `enum label "ON": a float64 value does not convert to string`),
		Entry("a pointer without a leading slash",
			func(_ context.Context, _, data channel.Channel) []kafka.ReadField {
				return []kafka.ReadField{{
					Key:      "v",
					Channel:  data.Key(),
					Pointer:  "v",
					DataType: telem.Float64T,
				}}
			}, `field "v": json pointer "v" must be empty or start with "/"`),
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
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return channelSvc.NewWriter(tx).Create(ctx, &other)
			})).To(Succeed())
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
			Expect(
				f.ConfigureTask(ctx, newTask(ctx, kafka.ReadTaskType, cfg), "cmd-1"),
			).
				Error().
				To(SatisfyAll(
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
		Expect(f.ConfigureTask(ctx, newTask(ctx, kafka.ReadTaskType, cfg), "cmd-1")).
			Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring(`record key "a": claimed by more than one`)),
		))
	})
})
