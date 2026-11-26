// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//go:build !windows

package mqtt_test

import (
	"context"
	"fmt"
	"math"
	"os"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	mochi "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/packets"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/mqtt"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/payload"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// benchScenario is a run of a benchmark: messages of fields float fields at rate
// messages per second for duration. A write run sends samples commands to each channel
// in each frame.
type benchScenario struct {
	fields, rate, samples int
	duration              time.Duration
}

func (s benchScenario) messages() int {
	return int(s.duration.Seconds() * float64(s.rate))
}

// benchValue is the sample of field i in message k. It uses every mantissa bit, so any
// rounding shows.
func benchValue(i, k int) float64 {
	return math.Sin(float64(k)*0.001+float64(i)) * math.Pow(10, float64(i%12-6))
}

// benchCost is the cost of a run: its time, its CPU time, and the bytes and objects
// that it allocated.
type benchCost struct {
	elapsed, cpu  time.Duration
	bytes, allocs uint64
}

// measure starts measuring the process and returns the function that ends it.
func measure() func() benchCost {
	GinkgoHelper()
	var before runtime.MemStats
	runtime.ReadMemStats(&before)
	cpuBefore, start := cpuTime(), time.Now()
	return func() benchCost {
		GinkgoHelper()
		var after runtime.MemStats
		runtime.ReadMemStats(&after)
		return benchCost{
			elapsed: time.Since(start),
			cpu:     cpuTime() - cpuBefore,
			bytes:   after.TotalAlloc - before.TotalAlloc,
			allocs:  after.Mallocs - before.Mallocs,
		}
	}
}

func cpuTime() time.Duration {
	GinkgoHelper()
	var ru syscall.Rusage
	Expect(syscall.Getrusage(syscall.RUSAGE_SELF, &ru)).To(Succeed())
	return time.Duration(ru.Utime.Nano() + ru.Stime.Nano())
}

// report prints the cost of each of n messages as a line that benchstat reads. Run go
// test with -v and -ginkgo.v to see it. The cost is of the whole process: the broker,
// the client, and the Core too.
func report(name string, n int, c benchCost, extra ...string) {
	line := fmt.Sprintf(
		"Benchmark%s %d %.0f cpu-ns/msg %.0f B/msg %.1f allocs/msg",
		name,
		n,
		float64(c.cpu.Nanoseconds())/float64(n),
		float64(c.bytes)/float64(n),
		float64(c.allocs)/float64(n),
	)
	GinkgoWriter.Println(strings.Join(append([]string{line}, extra...), " "))
}

func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return -1
	}
	return sorted[max(int(math.Ceil(p*float64(len(sorted))))-1, 0)]
}

// benchDuration returns the duration that MQTT_BENCH_DURATION sets, or else d.
func benchDuration(d time.Duration) time.Duration {
	if env, err := time.ParseDuration(os.Getenv("MQTT_BENCH_DURATION")); err == nil {
		return env
	}
	return d
}

var _ = Describe("Task benchmark", Serial, func() {
	var (
		broker  *testBroker
		rackKey rack.Key
		dev     device.Device
	)

	BeforeEach(func(ctx SpecContext) {
		if os.Getenv("MQTT_BENCH") == "" {
			Skip("set MQTT_BENCH to run the task benchmarks")
		}
		broker = startBroker(0)
		rackKey = createRack(ctx)
		dev = createBrokerDevice(ctx, rackKey, broker.port, nil)
	})

	start := func(
		ctx context.Context,
		taskType string,
		cfg msgpack.EncodedJSON,
		f driver.Factory,
	) {
		GinkgoHelper()
		t := newTask(ctx, rackKey, taskType, cfg)
		configured := configure(ctx, f, t)
		Expect(configured.Exec(ctx, task.Command{Type: "start", Key: "start"})).
			To(Succeed())
	}

	DescribeTable(
		"Should store every field of every message",
		func(
			ctx SpecContext,
			sc benchScenario,
		) {
			sc.duration = benchDuration(sc.duration)
			fields := append(make([]icd.JSONField, 0, sc.fields+1), jsonField(
				"ts",
				"/ts",
				icd.TimestampJSONField{TimeFormat: payload.TimeFormatUnixNs},
			))
			for i := range sc.fields {
				name := "f" + strconv.Itoa(i)
				fields = append(fields, jsonField(name, "/"+name, icd.FloatJSONField{}))
			}
			m := jsonMessage("Reading", nil, fields...)
			ic := createICD(ctx, m)
			dataTypes := slices.Repeat([]telem.DataType{telem.Float64T}, sc.fields)
			idx, data := createIndexed(ctx, dataTypes...)
			bindings := append(
				make([]map[string]any, 0, len(data)+1),
				binding(m, 0, idx),
			)
			for i, ch := range data {
				bindings = append(bindings, binding(m, i+1, ch))
			}
			read := readMessage(m, bindings...)
			read["index"] = fieldKey(m, 0)
			const topic = "bench/reading"
			total := sc.messages()
			start(
				ctx,
				mqtt.ReadTaskType,
				readConfig(ctx, dev, icdEntry(topic, ic, read)),
				newFactory(total),
			)
			Eventually(broker.subscriptions).Should(BeNumerically(">", 0))

			s := stream(ctx, channel.Keys{idx.Key()})
			var (
				mu        sync.Mutex
				latencies []float64
				received  atomic.Int64
			)
			collected := make(chan struct{})
			go func() {
				defer GinkgoRecover()
				defer close(collected)
				for res := range s.responses {
					now := telem.Now()
					ms := res.Frame.Get(idx.Key())
					mu.Lock()
					for _, series := range ms.Series {
						for i := range series.Len() {
							stamp := series.ValueAt[telem.TimeStamp](int(i))
							latencies = append(latencies, float64(now-stamp)/1e6)
						}
					}
					mu.Unlock()
					if received.Add(ms.Len()) >= int64(total) {
						return
					}
				}
			}()

			period := time.Second / time.Duration(sc.rate)
			end := measure()
			begin := time.Now()
			var last telem.TimeStamp
			// The measured loops check errors without Expect, which allocates.
			for k := range total {
				if wait := time.Until(begin.Add(time.Duration(k) * period)); wait > 0 {
					time.Sleep(wait)
				}
				last = max(telem.Now(), last+1)
				doc := strconv.AppendInt([]byte(`{"ts":`), int64(last), 10)
				for i := range sc.fields {
					doc = append(doc, `,"f`...)
					doc = strconv.AppendInt(doc, int64(i), 10)
					doc = append(doc, `":`...)
					doc = strconv.AppendFloat(doc, benchValue(i, k), 'g', -1, 64)
				}
				doc = append(doc, '}')
				if err := broker.server.Publish(topic, doc, false, 0); err != nil {
					Fail(err.Error())
				}
			}
			Eventually(collected).WithTimeout(time.Minute).Should(BeClosed())
			cost := end()

			mu.Lock()
			sorted := slices.Sorted(slices.Values(latencies))
			mu.Unlock()
			report(
				fmt.Sprintf("MQTTRead/fields=%d/rate=%d", sc.fields, sc.rate),
				total,
				cost,
				fmt.Sprintf("%.3f p50-ms", percentile(sorted, 0.5)),
				fmt.Sprintf("%.3f p99-ms", percentile(sorted, 0.99)),
			)
			Expect(received.Load()).To(BeEquivalentTo(total))
			for i, ch := range data {
				values := stored(ctx, ch)
				Expect(values.Len()).To(BeEquivalentTo(total))
				for k := range total {
					Expect(values.ValueAt[float64](k)).To(Equal(benchValue(i, k)))
				}
			}
		},
		Entry(
			"one field",
			benchScenario{fields: 1, rate: 2000, duration: 5 * time.Second},
		),
		Entry(
			"ten fields",
			benchScenario{fields: 10, rate: 2000, duration: 5 * time.Second},
		),
		Entry(
			"a hundred fields",
			benchScenario{fields: 100, rate: 1000, duration: 5 * time.Second},
		),
	)

	DescribeTable("Should publish every command of every frame", func(
		ctx SpecContext,
		sc benchScenario,
	) {
		sc.duration = benchDuration(sc.duration)
		const topic = "bench/command"
		var published atomic.Int64
		Expect(broker.server.Subscribe(topic, 1, func(
			*mochi.Client, packets.Subscription, packets.Packet,
		) {
			published.Add(1)
		})).To(Succeed())
		fields := make([]icd.JSONField, 0, sc.fields)
		for i := range sc.fields {
			name := "f" + strconv.Itoa(i)
			fields = append(fields, jsonField(name, "/"+name, icd.FloatJSONField{}))
		}
		m := jsonMessage("Command", nil, fields...)
		ic := createICD(ctx, m)
		keys := make(channel.Keys, sc.fields)
		bindings := make([]map[string]any, sc.fields)
		for i := range keys {
			ch := createVirtual(ctx, telem.Float64T)
			keys[i] = ch.Key()
			bindings[i] = channelBinding(m, i, ch)
		}
		start(
			ctx,
			mqtt.WriteTaskType,
			writeConfig(dev, messageTarget(topic, ic, m, bindings...)),
			newFactory(64),
		)
		w := MustSucceed(framerSvc.OpenWriter(ctx, framer.WriterConfig{
			Keys:  keys,
			Start: telem.Now(),
		}))
		DeferCleanup(func() { Expect(w.Close()).To(Succeed()) })
		values := make([]float64, sc.samples)
		// commands returns a frame of the commands of frame k. The writer reads a
		// frame after Write returns, so each one is new.
		commands := func(k int) frame.Frame {
			series := make([]telem.Series, len(keys))
			for i := range series {
				for j := range values {
					values[j] = benchValue(i, k*sc.samples+j)
				}
				series[i] = telem.NewSeries(values)
			}
			return frame.NewMulti(keys, series)
		}
		// The task misses the frames written before its streamer connects.
		Eventually(func() int64 {
			Expect(w.Write(commands(0))).To(BeTrue())
			return published.Load()
		}).Should(BeNumerically(">", 0))
		Eventually(func() int64 {
			before := published.Load()
			time.Sleep(50 * time.Millisecond)
			return published.Load() - before
		}).Should(BeZero())
		published.Store(0)

		frames := sc.messages() / sc.samples
		total := frames * sc.samples
		period := time.Duration(sc.samples) * time.Second / time.Duration(sc.rate)
		end := measure()
		begin := time.Now()
		for k := range frames {
			if wait := time.Until(begin.Add(time.Duration(k) * period)); wait > 0 {
				time.Sleep(wait)
			}
			authorized, err := w.Write(commands(k))
			if err != nil {
				Fail(err.Error())
			}
			if !authorized {
				Fail("the writer is not authorized")
			}
		}
		Eventually(published.Load).WithTimeout(time.Minute).
			Should(BeEquivalentTo(total))
		report(
			fmt.Sprintf(
				"MQTTWrite/fields=%d/samples=%d/rate=%d",
				sc.fields, sc.samples, sc.rate,
			),
			total,
			end(),
		)
	},
		Entry(
			"one field",
			benchScenario{
				fields:   1,
				samples:  10,
				rate:     2000,
				duration: 5 * time.Second,
			},
		),
		Entry(
			"ten fields",
			benchScenario{
				fields:   10,
				samples:  10,
				rate:     2000,
				duration: 5 * time.Second,
			},
		),
		Entry(
			"a hundred fields",
			benchScenario{
				fields:   100,
				samples:  10,
				rate:     1000,
				duration: 5 * time.Second,
			},
		),
	)
})
