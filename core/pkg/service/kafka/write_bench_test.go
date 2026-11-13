// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//go:build !windows

package kafka_test

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"math"
	"os"
	"strconv"
	"time"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/kafka"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/payload"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/twmb/franz-go/pkg/kgo"
)

// writeBenchRecordPeriod spaces the frames per record they hold. A streamer that falls
// behind drops frames instead of slowing the writer, so the benchmark holds a rate the
// task can keep up with under exact allocation profiling.
const writeBenchRecordPeriod = 250 * time.Microsecond

type writeBenchScenario struct {
	channels, samples, frames int
	// fields adds a static and a generated field to every record.
	fields bool
}

func (s writeBenchScenario) name() string {
	name := fmt.Sprintf(
		"write-%dch-%dsamples-%dframes",
		s.channels,
		s.samples,
		s.frames,
	)
	if s.fields {
		name += "-fields"
	}
	return name
}

var _ = Describe("Write task benchmark", Serial, func() {
	BeforeEach(func() {
		if os.Getenv("KAFKA_BENCH") == "" {
			Skip("set KAFKA_BENCH to run the write task benchmark")
		}
	})

	DescribeTable("Should produce every sample with its exact value and stamp",
		func(ctx SpecContext, sc writeBenchScenario) {
			runWriteBench(ctx, sc)
		},
		func(sc writeBenchScenario) string { return sc.name() },
		Entry(nil, writeBenchScenario{channels: 1, samples: 100, frames: 200}),
		Entry(nil, writeBenchScenario{channels: 100, samples: 10, frames: 40}),
		Entry(nil, writeBenchScenario{
			channels: 100,
			samples:  10,
			frames:   40,
			fields:   true,
		}),
	)
})

func runWriteBench(ctx SpecContext, sc writeBenchScenario) {
	GinkgoHelper()
	f := newFactory()
	groups := createBenchGroups(
		ctx, benchScenario{shape: wide, channels: sc.channels},
	)
	index, data := groups[0].index, groups[0].data
	topic := uniqueTopic()
	channels := make([]kafka.WriteChannel, len(data))
	byName := make(map[string]int, len(data))
	for i, ch := range data {
		base := kafka.BaseWriteChannel{Key: strconv.Itoa(i), Channel: ch.Key()}
		channels[i] = kafka.WriteChannel{
			Variant: kafka.FloatWriteChannel{BaseWriteChannel: base},
		}
		byName[ch.Name] = i
	}
	cfg := kafka.WriteConfig{
		Device: createDevice(ctx, clusterProperties()).Key,
		Topic:  topic,
		Record: kafka.Record{
			ChannelPointer:   new("/channel"),
			TimestampPointer: new("/timestamp"),
		},
		Channels: channels,
	}
	if sc.fields {
		cfg.Record.Fields = []payload.WriteField{
			{Variant: payload.StringWriteField{
				BaseWriteField: payload.BaseWriteField{
					Key: "source", Pointer: "/meta/source",
				},
				Value: "synnax",
			}},
			{Variant: payload.UuidWriteField{
				BaseWriteField: payload.BaseWriteField{Key: "id", Pointer: "/id"},
			}},
		}
	}
	t := newTask(ctx, kafka.WriteTaskType, cfg)
	dt := MustSucceed(f.ConfigureTask(ctx, t, "cmd-1"))
	DeferCleanup(func() { Expect(dt.Stop(false)).To(Succeed()) })
	Expect(dt.Exec(ctx, task.Command{Task: t.Key, Type: "start", Key: "cmd-2"})).
		To(Succeed())

	keys := make(channel.Keys, 0, len(data)+1)
	keys = append(keys, index.Key())
	for _, ch := range data {
		keys = append(keys, ch.Key())
	}
	base := telem.Now() - telem.TimeStamp(telem.Hour)
	stamp := func(frameI, sampleI int) telem.TimeStamp {
		return base + telem.TimeStamp(frameI*sc.samples+sampleI)*telem.MillisecondTS
	}
	w := MustOpen(framerSvc.OpenWriter(ctx, framer.WriterConfig{
		Keys:  keys,
		Start: base,
	}))
	stamps := make([]telem.TimeStamp, sc.samples)
	values := make([]float64, sc.samples)

	period := time.Duration(sc.channels*sc.samples) * writeBenchRecordPeriod
	profile := startBenchProfile(sc.name())
	start := time.Now()
	for k := range sc.frames {
		if wait := time.Until(start.Add(time.Duration(k) * period)); wait > 0 {
			time.Sleep(wait)
		}
		for i := range sc.samples {
			stamps[i] = stamp(k, i)
		}
		// The relay reads a written frame after Write returns, so each one is new.
		series := make([]telem.Series, len(keys))
		series[0] = telem.NewSeries(stamps)
		for c := range data {
			for i := range sc.samples {
				values[i] = benchValue(c, k*sc.samples+i)
			}
			series[c+1] = telem.NewSeries(values)
		}
		Expect(MustSucceed(w.Write(frame.NewMulti(keys, series)))).To(BeTrue())
	}
	total := sc.channels * sc.samples * sc.frames
	records := consumeAll(ctx, topic, total)
	elapsed := time.Since(start)
	profile.stop()

	seen := make([]int, len(data))
	ids := make(set.Set[uuid.UUID])
	for _, rec := range records {
		var doc struct {
			Value     float64         `json:"value"`
			Channel   string          `json:"channel"`
			Timestamp telem.TimeStamp `json:"timestamp"`
			ID        string          `json:"id"`
			Meta      struct {
				Source string `json:"source"`
			} `json:"meta"`
		}
		Expect(json.Unmarshal(rec.Value, &doc)).To(Succeed())
		Expect(byName).To(HaveKey(doc.Channel))
		c := byName[doc.Channel]
		Expect(string(rec.Key)).To(Equal(doc.Channel))
		i := seen[c]
		seen[c]++
		Expect(math.Float64bits(doc.Value)).
			To(Equal(math.Float64bits(benchValue(c, i))))
		Expect(doc.Timestamp).To(Equal(stamp(i/sc.samples, i%sc.samples)))
		if sc.fields {
			Expect(doc.Meta.Source).To(Equal("synnax"))
			id := MustSucceed(uuid.Parse(doc.ID))
			Expect(ids.Contains(id)).To(BeFalse())
			ids.Add(id)
		}
	}
	GinkgoWriter.Printf("BENCH %s %d records in %s\n", sc.name(), len(records), elapsed)
}

// consumeAll reads exactly n records from topic.
func consumeAll(ctx context.Context, topic string, n int) []*kgo.Record {
	GinkgoHelper()
	cl := newKafkaClient(
		kgo.ConsumeTopics(topic),
		kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()),
		kgo.FetchMaxBytes(16<<20),
	)
	recs := make([]*kgo.Record, 0, n)
	Eventually(func() int {
		pollCtx, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
		defer cancel()
		recs = append(recs, cl.PollFetches(pollCtx).Records()...)
		return len(recs)
	}).WithTimeout(2 * time.Minute).WithPolling(time.Millisecond).
		Should(BeNumerically(">=", n))
	Expect(recs).To(HaveLen(n))
	return recs
}
