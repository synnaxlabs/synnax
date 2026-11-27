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
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/kafka"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/payload"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kgo"
)

// benchShape is how a scenario lays its channels out across records.
type benchShape string

const (
	// wide produces one record per tick that carries every channel.
	wide benchShape = "wide"
	// narrow produces one record per channel per tick, keyed by channel.
	narrow benchShape = "narrow"
)

type benchScenario struct {
	shape    benchShape
	channels int
	rate     int
	duration time.Duration
	// kafkaTime stamps samples with the Kafka timestamp instead of a body field.
	kafkaTime bool
	// backlog produces every record before the task starts, so the run measures how
	// fast the task drains a backlog with no producer competing for CPU.
	backlog bool
}

func (s benchScenario) name() string {
	stamp := "body"
	if s.kafkaTime {
		stamp = "kafka"
	}
	name := fmt.Sprintf("%s-%dch-%dhz-%s", s.shape, s.channels, s.rate, stamp)
	if s.backlog {
		name += "-backlog"
	}
	return name
}

// benchValue is the sample channel i carries at tick k. It spans many magnitudes and
// uses every mantissa bit, so any rounding shows.
func benchValue(i, k int) float64 {
	return math.Sin(float64(k)*0.001+float64(i)) * math.Pow(10, float64(i%12-6))
}

type benchResult struct {
	Scenario          string  `json:"scenario"`
	Records           int64   `json:"records"`
	Samples           int64   `json:"samples"`
	ProduceSeconds    float64 `json:"produce_s"`
	AchievedRate      float64 `json:"achieved_rate_hz"`
	DrainMs           float64 `json:"drain_ms"`
	LatencyP50Ms      float64 `json:"latency_p50_ms"`
	LatencyP95Ms      float64 `json:"latency_p95_ms"`
	LatencyP99Ms      float64 `json:"latency_p99_ms"`
	LatencyMaxMs      float64 `json:"latency_max_ms"`
	LatencyFirstSecMs float64 `json:"latency_first_s_p50_ms"`
	LatencyLastSecMs  float64 `json:"latency_last_s_p50_ms"`
	Missing           int64   `json:"missing"`
	Extra             int64   `json:"extra"`
	ValueMismatches   int64   `json:"value_mismatches"`
	StampMismatches   int64   `json:"stamp_mismatches"`
	StampErrP50Ms     float64 `json:"stamp_err_p50_ms"`
	StampErrP99Ms     float64 `json:"stamp_err_p99_ms"`
	StampErrMaxMs     float64 `json:"stamp_err_max_ms"`
	IntervalJitterMs  float64 `json:"interval_jitter_stddev_ms"`
	CPUCores          float64 `json:"cpu_cores"`
	AllocMBPerSec     float64 `json:"alloc_mb_per_s"`
	TaskStatus        string  `json:"task_status"`
	DrainSamplesPerS  float64 `json:"drain_samples_per_s"`
	DrainRecordsPerS  float64 `json:"drain_records_per_s"`
}

// benchProfile records one scenario's CPU and allocation profiles when
// KAFKA_BENCH_PPROF names a directory. The allocation profile holds exact counts, so
// compare it to its base with pprof -base, or the setup shows up in it. Exact counts
// slow the task, so the timings of a profiled run are not comparable.
type benchProfile struct {
	root *os.Root
	name string
	cpu  *os.File
}

func startBenchProfile(name string) *benchProfile {
	GinkgoHelper()
	dir := os.Getenv("KAFKA_BENCH_PPROF")
	if dir == "" {
		return &benchProfile{}
	}
	root := MustSucceed(os.OpenRoot(dir))
	DeferCleanup(root.Close)
	p := &benchProfile{root: root, name: name}
	rate := runtime.MemProfileRate
	DeferCleanup(func() { runtime.MemProfileRate = rate })
	runtime.MemProfileRate = 1
	p.writeAllocs("allocs.base")
	p.cpu = MustSucceed(root.Create(name + ".cpu.pprof"))
	Expect(pprof.StartCPUProfile(p.cpu)).To(Succeed())
	DeferCleanup(p.stop)
	return p
}

// stop ends the profile. A spec calls it to bound the measurement, and a failed spec
// that never reached that call stops it in cleanup.
func (p *benchProfile) stop() {
	GinkgoHelper()
	if p.cpu == nil {
		return
	}
	pprof.StopCPUProfile()
	Expect(p.cpu.Close()).To(Succeed())
	p.cpu = nil
	p.writeAllocs("allocs")
}

func (p *benchProfile) writeAllocs(suffix string) {
	GinkgoHelper()
	// An allocation reaches the profile only after the sweep that follows it.
	runtime.GC()
	runtime.GC()
	f := MustSucceed(p.root.Create(p.name + "." + suffix + ".pprof"))
	Expect(pprof.Lookup("allocs").WriteTo(f, 0)).To(Succeed())
	Expect(f.Close()).To(Succeed())
}

func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return -1
	}
	i := int(math.Ceil(p*float64(len(sorted)))) - 1
	return sorted[max(i, 0)]
}

func cpuTime() time.Duration {
	GinkgoHelper()
	var ru syscall.Rusage
	Expect(syscall.Getrusage(syscall.RUSAGE_SELF, &ru)).To(Succeed())
	return time.Duration(ru.Utime.Nano() + ru.Stime.Nano())
}

var _ = Describe("Read task benchmark", Serial, func() {
	BeforeEach(func() {
		if os.Getenv("KAFKA_BENCH") == "" {
			Skip("set KAFKA_BENCH to run the read task benchmark, and " +
				"KAFKA_BENCH_BROKERS to run it against a real cluster")
		}
	})

	DescribeTable("Should store every sample with its exact value and stamp",
		func(ctx SpecContext, sc benchScenario) {
			if d, err := time.ParseDuration(
				os.Getenv("KAFKA_BENCH_DURATION"),
			); err == nil {
				sc.duration = d
			}
			res := runBench(ctx, sc)
			b := MustSucceed(json.Marshal(res))
			GinkgoWriter.Printf("BENCH %s\n", b)
			if out := os.Getenv("KAFKA_BENCH_OUT"); out != "" {
				f := MustSucceed(
					os.OpenFile(
						filepath.Clean(out),
						os.O_APPEND|os.O_CREATE|os.O_WRONLY,
						0o644,
					),
				)
				MustSucceed(f.Write(append(b, '\n')))
				Expect(f.Close()).To(Succeed())
			}
			Expect(res.Missing).To(BeZero())
			Expect(res.Extra).To(BeZero())
			Expect(res.ValueMismatches).To(BeZero())
			if !sc.kafkaTime {
				Expect(res.StampMismatches).To(BeZero())
			}
		},
		benchEntries(),
	)
})

func benchEntries() []TableEntry {
	var entries []TableEntry
	add := func(sc benchScenario) {
		entries = append(entries, Entry(sc.name(), sc))
	}
	d := 10 * time.Second
	for _, ch := range []int{1, 10, 50, 200, 500, 1000} {
		for _, rate := range []int{50, 200, 1000} {
			add(benchScenario{shape: wide, channels: ch, rate: rate, duration: d})
		}
	}
	for _, ch := range []int{10, 50, 200} {
		for _, rate := range []int{50, 200, 1000} {
			add(benchScenario{shape: narrow, channels: ch, rate: rate, duration: d})
		}
	}
	add(
		benchScenario{
			shape:     wide,
			channels:  200,
			rate:      200,
			duration:  d,
			kafkaTime: true,
		},
	)
	add(
		benchScenario{
			shape:     narrow,
			channels:  200,
			rate:      200,
			duration:  d,
			kafkaTime: true,
		},
	)
	for _, sc := range []benchScenario{
		{shape: wide, channels: 200, rate: 200},
		{shape: wide, channels: 1000, rate: 200},
		{shape: wide, channels: 10, rate: 1000},
		{shape: narrow, channels: 200, rate: 200},
		{shape: narrow, channels: 10, rate: 1000},
	} {
		sc.duration, sc.backlog = d, true
		add(sc)
	}
	return entries
}

// benchGroup is one record key's channels: an index and its data channels.
type benchGroup struct {
	key   string
	index channel.Channel
	data  []channel.Channel
	// first is the position of this group's first channel among all channels.
	first int
}

func createBenchGroups(ctx context.Context, sc benchScenario) []benchGroup {
	GinkgoHelper()
	suffix := strconv.Itoa(int(time.Now().UnixNano() % 1e9))
	newIndex := func(name string) channel.Channel {
		GinkgoHelper()
		ch := channel.Channel{Name: name, DataType: telem.TimestampT, IsIndex: true}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channelSvc.NewWriter(tx).Create(ctx, &ch)
		})).To(Succeed())
		return ch
	}
	newData := func(name string, idx channel.Channel) channel.Channel {
		GinkgoHelper()
		ch := channel.Channel{
			Name:       name,
			DataType:   telem.Float64T,
			LocalIndex: idx.LocalKey,
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channelSvc.NewWriter(tx).Create(ctx, &ch)
		})).To(Succeed())
		return ch
	}
	if sc.shape == wide {
		idx := newIndex("bench_idx_" + suffix)
		g := benchGroup{key: "tick", index: idx}
		data := make([]channel.Channel, sc.channels)
		for i := range data {
			data[i] = channel.Channel{
				Name:       fmt.Sprintf("bench_%s_%d", suffix, i),
				DataType:   telem.Float64T,
				LocalIndex: idx.LocalKey,
			}
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channelSvc.NewWriter(tx).CreateMany(ctx, &data)
		})).To(Succeed())
		g.data = data
		return []benchGroup{g}
	}
	groups := make([]benchGroup, sc.channels)
	for i := range groups {
		idx := newIndex(fmt.Sprintf("bench_idx_%s_%d", suffix, i))
		groups[i] = benchGroup{
			key:   fmt.Sprintf("ch_%d", i),
			index: idx,
			data: []channel.Channel{
				newData(fmt.Sprintf("bench_%s_%d", suffix, i), idx),
			},
			first: i,
		}
	}
	return groups
}

func benchRecords(
	ctx context.Context,
	groups []benchGroup,
	sc benchScenario,
) []kafka.ReadRecord {
	GinkgoHelper()
	unixNs := payload.TimeFormatUnixNs
	recs := make([]kafka.InlineRecord, len(groups))
	for gi, g := range groups {
		rec := kafka.InlineRecord{Key: fmt.Sprintf("r%d", gi), RecordKey: g.key}
		if !sc.kafkaTime {
			rec.Index = "ts"
			rec.Fields = append(rec.Fields, kafka.ReadField{
				Key:        "ts",
				Channel:    g.index.Key(),
				Pointer:    "/ts",
				DataType:   telem.TimestampT,
				TimeFormat: &unixNs,
			})
		}
		for i, ch := range g.data {
			ptr := "/value"
			if sc.shape == wide {
				ptr = "/c" + strconv.Itoa(i)
			}
			rec.Fields = append(rec.Fields, kafka.ReadField{
				Key:      "f" + strconv.Itoa(i),
				Channel:  ch.Key(),
				Pointer:  ptr,
				DataType: telem.Float64T,
			})
		}
		recs[gi] = rec
	}
	return lift(ctx, recs...)
}

// latencyLog collects end-to-end latency samples, keyed by when they were seen.
type latencyLog struct {
	mu sync.Mutex
	// seen[i] is when sample i arrived, since production started.
	seen []time.Duration
	// lat[i] is the latency of sample i in milliseconds.
	lat []float64
}

func runBench(ctx SpecContext, sc benchScenario) benchResult {
	GinkgoHelper()
	f := newFactory()
	groups := createBenchGroups(ctx, sc)
	brokers := benchBrokers()
	dev := createDevice(ctx, kafka.Properties{Brokers: brokers})
	topic := uniqueTopic()
	producer := openBenchProducer(ctx, brokers, topic)
	t, startTask := configureBenchTask(ctx, f, dev, topic, groups, sc)
	if !sc.backlog {
		startTask()
	}
	outlet, isIndex, closeRequests := openBenchStreamer(ctx, groups)

	var (
		ll       latencyLog
		received atomic.Int64
		t0       time.Time
	)
	done := make(chan struct{})
	go func() {
		defer close(done)
		ll.collect(outlet, isIndex, &received, &t0)
	}()

	ticks := int(sc.duration.Seconds() * float64(sc.rate))
	bp := newBenchProducer(producer, topic, sc, groups, ticks)

	var m benchMeasure
	if !sc.backlog {
		m.begin(sc.name())
	}
	t0 = time.Now()
	period := time.Second / time.Duration(sc.rate)
	base := telem.Now() - telem.TimeStamp(sc.duration)
	bp.produceTicks(ctx, ticks, t0, period, base)
	produceSecs := time.Since(t0).Seconds()
	bp.flush(ctx)
	if sc.backlog {
		m.begin(sc.name())
		startTask()
	}
	drain := waitReceived(&received, int64(ticks*len(groups)))
	elapsed, cpu, alloc := m.end()
	closeRequests()
	Eventually(done).Should(BeClosed())

	res := newBenchResult(sc, ticks, produceSecs, drain, elapsed, cpu, alloc)
	res.TaskStatus = statusText(taskStatus(ctx, t))

	ll.summarize(&res, produceSecs)
	intervals := checkStored(ctx, sc, groups, bp.produced, ticks, &res)
	res.IntervalJitterMs = stddevFrom(intervals, float64(period)/1e6)
	return res
}

func statusText(stat task.Status) string {
	return fmt.Sprintf("%s: %s %s", stat.Variant, stat.Message, stat.Description)
}

// waitReceived waits until received reaches want and returns how long it took.
func waitReceived(received *atomic.Int64, want int64) time.Duration {
	GinkgoHelper()
	start := time.Now()
	Eventually(received.Load).WithTimeout(5 * time.Minute).
		WithPolling(5 * time.Millisecond).Should(BeNumerically(">=", want))
	return time.Since(start)
}

// newBenchResult returns the throughput and cost of a scenario that produced ticks
// ticks in produceSecs seconds, drained in drain, and used cpu and alloc bytes in
// elapsed.
func newBenchResult(
	sc benchScenario,
	ticks int,
	produceSecs float64,
	drain, elapsed, cpu time.Duration,
	alloc uint64,
) benchResult {
	res := benchResult{
		Scenario:       sc.name(),
		Records:        producedRecords(sc, ticks),
		Samples:        int64(ticks * sc.channels),
		ProduceSeconds: produceSecs,
		AchievedRate:   float64(ticks) / produceSecs,
		DrainMs:        float64(drain) / 1e6,
		CPUCores:       cpu.Seconds() / elapsed.Seconds(),
		AllocMBPerSec:  float64(alloc) / 1e6 / elapsed.Seconds(),
	}
	if sc.backlog {
		res.DrainSamplesPerS = float64(res.Samples) / drain.Seconds()
		res.DrainRecordsPerS = float64(res.Records) / drain.Seconds()
	}
	return res
}

// benchBrokers returns the brokers that KAFKA_BENCH_BROKERS names, or else the brokers
// of the fake cluster.
func benchBrokers() []string {
	if b := os.Getenv("KAFKA_BENCH_BROKERS"); b != "" {
		return strings.Split(b, ",")
	}
	return cluster.ListenAddrs()
}

// openBenchProducer opens a Kafka producer on brokers and creates topic.
func openBenchProducer(
	ctx context.Context,
	brokers []string,
	topic string,
) *kgo.Client {
	GinkgoHelper()
	producer := MustSucceed(kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.ProducerLinger(time.Millisecond),
		kgo.MaxBufferedRecords(1<<20),
		kgo.ProducerBatchMaxBytes(1_000_000),
		kgo.RecordPartitioner(kgo.ManualPartitioner()),
	))
	DeferCleanup(producer.Close)
	MustSucceed(kadm.NewClient(producer).CreateTopic(ctx, 1, 1, nil, topic))
	return producer
}

// configureBenchTask configures a read task of topic on dev and returns it with a
// function that starts it.
func configureBenchTask(
	ctx context.Context,
	f driver.Factory,
	dev device.Device,
	topic string,
	groups []benchGroup,
	sc benchScenario,
) (task.Task, func()) {
	GinkgoHelper()
	cfg := kafka.ReadConfig{
		Device:      dev.Key,
		Topics:      oneTopic(topic, benchRecords(ctx, groups, sc)),
		StartOffset: kafka.StartOffsetEarliest,
	}
	t := newTask(ctx, kafka.ReadTaskType, cfg)
	dt := MustSucceed(f.ConfigureTask(ctx, t, "cmd-1"))
	DeferCleanup(func() { Expect(dt.Stop(false)).To(Succeed()) })
	startTask := func() {
		GinkgoHelper()
		Expect(dt.Exec(ctx, task.Command{Task: t.Key, Type: "start", Key: "cmd-2"})).
			To(Succeed())
	}
	return t, startTask
}

// openBenchStreamer streams the index channels of groups. It returns the responses, the
// set of streamed keys, and a function that closes the requests.
func openBenchStreamer(
	ctx context.Context,
	groups []benchGroup,
) (<-chan framer.StreamerResponse, set.Set[channel.Key], func()) {
	GinkgoHelper()
	streamKeys := make(channel.Keys, 0, len(groups))
	isIndex := make(set.Set[channel.Key])
	for _, g := range groups {
		streamKeys = append(streamKeys, g.index.Key())
		isIndex.Add(g.index.Key())
	}
	streamer := MustSucceed(framerSvc.NewStreamer(ctx, framer.StreamerConfig{
		Keys:        streamKeys,
		SendOpenAck: true,
	}))
	sCtx, cancel := signal.Isolated()
	requests, responses := confluence.Attach(streamer, 1000)
	streamer.Flow(sCtx, confluence.CloseOutputInletsOnExit())
	// A streamer left open blocks the relay's close at suite teardown.
	DeferCleanup(func() {
		requests.Close()
		cancel()
		Expect(sCtx.Wait()).To(Or(Succeed(), MatchError(context.Canceled)))
	})
	Eventually(responses.Outlet()).Should(Receive())
	return responses.Outlet(), isIndex, requests.Close
}

// benchMeasure measures the CPU time and allocations of a scenario.
type benchMeasure struct {
	profile   *benchProfile
	msBefore  runtime.MemStats
	cpuBefore time.Duration
	measureAt time.Time
}

func (m *benchMeasure) begin(name string) {
	m.profile = startBenchProfile(name)
	runtime.ReadMemStats(&m.msBefore)
	m.cpuBefore = cpuTime()
	m.measureAt = time.Now()
}

// end returns the time, CPU time, and bytes allocated since begin.
func (m *benchMeasure) end() (elapsed, cpu time.Duration, alloc uint64) {
	elapsed = time.Since(m.measureAt)
	cpu = cpuTime() - m.cpuBefore
	var msAfter runtime.MemStats
	runtime.ReadMemStats(&msAfter)
	m.profile.stop()
	return elapsed, cpu, msAfter.TotalAlloc - m.msBefore.TotalAlloc
}

// checkStored compares the stored index and data channels of each group to what was
// produced, records the mismatches and stamp errors in res, and returns the intervals
// between stored index stamps in milliseconds.
func checkStored(
	ctx context.Context,
	sc benchScenario,
	groups []benchGroup,
	produced [][]telem.TimeStamp,
	ticks int,
	res *benchResult,
) []float64 {
	GinkgoHelper()
	var stampErrs, intervals []float64
	for gi, g := range groups {
		idx := stored(ctx, g.index.Key())
		got := make([]telem.TimeStamp, idx.Len())
		for i := range got {
			got[i] = idx.ValueAt[telem.TimeStamp](i)
		}
		want := produced[gi]
		if n := int64(len(got)) - int64(len(want)); n > 0 {
			res.Extra += n
		} else {
			res.Missing -= n
		}
		for i := range min(len(got), len(want)) {
			if got[i] != want[i] {
				res.StampMismatches++
			}
			stampErrs = append(stampErrs, math.Abs(float64(got[i]-want[i]))/1e6)
			if i > 0 {
				intervals = append(intervals, float64(got[i]-got[i-1])/1e6)
			}
		}
		checkStoredValues(ctx, sc, g, ticks, res)
	}
	slices.Sort(stampErrs)
	res.StampErrP50Ms = percentile(stampErrs, 0.5)
	res.StampErrP99Ms = percentile(stampErrs, 0.99)
	if len(stampErrs) > 0 {
		res.StampErrMaxMs = stampErrs[len(stampErrs)-1]
	}
	return intervals
}

// checkStoredValues compares the stored data channels of g to the values produced in
// ticks ticks and records the mismatches in res.
func checkStoredValues(
	ctx context.Context,
	sc benchScenario,
	g benchGroup,
	ticks int,
	res *benchResult,
) {
	GinkgoHelper()
	for ci, ch := range g.data {
		s := stored(ctx, ch.Key())
		chIndex := g.first + ci
		if sc.shape == wide {
			chIndex = ci
		}
		for k := range min(int(s.Len()), ticks) {
			if math.Float64bits(s.ValueAt[float64](k)) !=
				math.Float64bits(benchValue(chIndex, k)) {
				res.ValueMismatches++
			}
		}
		if n := s.Len() - int64(ticks); n > 0 {
			res.Extra += n
		} else {
			res.Missing -= n
		}
	}
}

// benchProducer produces the records of a bench scenario.
type benchProducer struct {
	client *kgo.Client
	topic  string
	sc     benchScenario
	groups []benchGroup
	// body is the reused record body of the wide shape.
	body map[string]any
	// produced[i] holds the stamps produced for group i.
	produced [][]telem.TimeStamp
	promise  func(*kgo.Record, error)
	// produceErr holds the first error that a produce returned.
	produceErr atomic.Pointer[error]
}

// newBenchProducer returns a producer of ticks ticks of records on client.
func newBenchProducer(
	client *kgo.Client,
	topic string,
	sc benchScenario,
	groups []benchGroup,
	ticks int,
) *benchProducer {
	bp := &benchProducer{
		client:   client,
		topic:    topic,
		sc:       sc,
		groups:   groups,
		body:     make(map[string]any, sc.channels+1),
		produced: make([][]telem.TimeStamp, len(groups)),
	}
	for i := range bp.produced {
		bp.produced[i] = make([]telem.TimeStamp, 0, ticks)
	}
	bp.promise = func(_ *kgo.Record, err error) {
		if err != nil {
			bp.produceErr.CompareAndSwap(nil, &err)
		}
	}
	return bp
}

// flush waits for every record to be produced and fails on the first produce error.
func (bp *benchProducer) flush(ctx context.Context) {
	GinkgoHelper()
	Expect(bp.client.Flush(ctx)).To(Succeed())
	if p := bp.produceErr.Load(); p != nil {
		Fail((*p).Error())
	}
}

// produce produces the records of tick k, stamped ts.
func (bp *benchProducer) produce(ctx context.Context, k int, ts telem.TimeStamp) {
	GinkgoHelper()
	if bp.sc.shape == wide {
		clear(bp.body)
		bp.body["ts"] = int64(ts)
		for i := range bp.sc.channels {
			bp.body["c"+strconv.Itoa(i)] = benchValue(i, k)
		}
		b := MustSucceed(json.Marshal(bp.body))
		bp.client.Produce(ctx, &kgo.Record{
			Topic:     bp.topic,
			Key:       []byte("tick"),
			Timestamp: time.Unix(0, int64(ts)),
			Value:     b,
		}, bp.promise)
		bp.produced[0] = append(bp.produced[0], ts)
		return
	}
	for gi, g := range bp.groups {
		b := MustSucceed(json.Marshal(map[string]any{
			"ts":    int64(ts),
			"value": benchValue(gi, k),
		}))
		bp.client.Produce(ctx, &kgo.Record{
			Topic:     bp.topic,
			Key:       []byte(g.key),
			Timestamp: time.Unix(0, int64(ts)),
			Value:     b,
		}, bp.promise)
		bp.produced[gi] = append(bp.produced[gi], ts)
	}
}

// produceTicks produces ticks ticks of records, period apart from t0. A backlog
// scenario stamps them period apart from base without waiting.
func (bp *benchProducer) produceTicks(
	ctx context.Context,
	ticks int,
	t0 time.Time,
	period time.Duration,
	base telem.TimeStamp,
) {
	GinkgoHelper()
	var last telem.TimeStamp
	for k := range ticks {
		var ts telem.TimeStamp
		if bp.sc.backlog {
			ts = base + telem.TimeStamp(time.Duration(k)*period)
		} else {
			if wait := time.Until(t0.Add(time.Duration(k) * period)); wait > 0 {
				time.Sleep(wait)
			}
			ts = max(telem.Now(), last+1)
		}
		last = ts
		bp.produce(ctx, k, ts)
	}
}

// collect records the latency of each index sample in the streamer responses, and adds
// the count of each index series to received. t0 is when production started.
func (l *latencyLog) collect(
	responses <-chan framer.StreamerResponse,
	isIndex set.Set[channel.Key],
	received *atomic.Int64,
	t0 *time.Time,
) {
	for res := range responses {
		now := telem.Now()
		since := time.Since(*t0)
		l.mu.Lock()
		for key, s := range res.Frame.Entries() {
			if !isIndex.Contains(key) {
				continue
			}
			for i := range int(s.Len()) {
				ts := s.ValueAt[telem.TimeStamp](i)
				l.lat = append(l.lat, float64(now-ts)/1e6)
				l.seen = append(l.seen, since)
			}
			received.Add(s.Len())
		}
		l.mu.Unlock()
	}
}

// summarize records the latency percentiles in res, overall and for the first and last
// second of a production that took produceSecs.
func (l *latencyLog) summarize(res *benchResult, produceSecs float64) {
	l.mu.Lock()
	sorted := slices.Clone(l.lat)
	slices.Sort(sorted)
	res.LatencyP50Ms = percentile(sorted, 0.5)
	res.LatencyP95Ms = percentile(sorted, 0.95)
	res.LatencyP99Ms = percentile(sorted, 0.99)
	if len(sorted) > 0 {
		res.LatencyMaxMs = sorted[len(sorted)-1]
	}
	var first, lastSec []float64
	for i, s := range l.seen {
		if s < time.Second {
			first = append(first, l.lat[i])
		}
		if s > time.Duration(produceSecs*float64(time.Second))-time.Second {
			lastSec = append(lastSec, l.lat[i])
		}
	}
	l.mu.Unlock()
	slices.Sort(first)
	slices.Sort(lastSec)
	res.LatencyFirstSecMs = percentile(first, 0.5)
	res.LatencyLastSecMs = percentile(lastSec, 0.5)
}

func producedRecords(sc benchScenario, ticks int) int64 {
	if sc.shape == wide {
		return int64(ticks)
	}
	return int64(ticks * sc.channels)
}

// stddevFrom is the root mean square deviation of xs from nominal.
func stddevFrom(xs []float64, nominal float64) float64 {
	if len(xs) == 0 {
		return -1
	}
	var sum float64
	for _, x := range xs {
		sum += (x - nominal) * (x - nominal)
	}
	return math.Sqrt(sum / float64(len(xs)))
}
