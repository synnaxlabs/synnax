// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// This file provides a mixed-workload benchmark harness for gorp, oriented
// around GC behavior rather than raw operation latency. The existing
// gorp_bench_test.go and index_bench_test.go files measure per-op costs in
// isolation (ns/op, allocs/op, B/op); those numbers are useful for ranking
// alternatives but they don't answer the question that matters for a
// control system: "how much GC pause time does this workload actually
// produce, and where does it come from?"
//
// The harness here is designed to run sustained workloads against an index-
// equipped table for a fixed wall-clock duration, then report:
//
//   - Throughput: ops/sec by category (reads, writes, queries).
//   - Allocation rate: bytes and objects allocated per op.
//   - GC cycles: count and total wall time spent in GC during the run.
//   - Pause statistics: max pause, sum of pauses, pause-time as a percentage
//     of wall time. These are the numbers that matter for determinism
//     budgets.
//   - Live heap: steady-state heap footprint after the run and after a
//     forced GC, as a proxy for how much of the process memory the index
//     infrastructure contributes.
//
// Usage:
//
//	# Run the whole workload suite (long; several minutes):
//	go test -run=^$ -bench=BenchmarkWorkload -benchtime=1x ./...
//
//	# Run a single workload against one size:
//	go test -run=^$ -bench=BenchmarkWorkload_MixedReadWrite/n=100000 \
//	    -benchtime=1x ./...
//
//	# Capture a runtime/trace for one of the workload runs so we can
//	# inspect STW windows with `go tool trace`:
//	GORP_WORKLOAD_TRACE=/tmp/gorp.trace go test -run=^$ \
//	    -bench=BenchmarkWorkload_MixedReadWrite/n=100000 ./...
//	go tool trace /tmp/gorp.trace
//
//	# Capture a heap profile at the end of one workload run:
//	GORP_WORKLOAD_HEAP=/tmp/gorp.heap go test -run=^$ \
//	    -bench=BenchmarkWorkload_SteadyStateQueries/n=100000 ./...
//	go tool pprof /tmp/gorp.heap
//
// benchtime=1x forces a single sample per sub-bench so we get one run at
// whatever duration the harness picks, instead of Go's auto-scaling loop.
//
// The workloads are designed so that changing a storage implementation
// (e.g., swapping Lookup's map backing for an off-heap layout) should move
// the GC metrics in a visible, attributable way.

package gorp_test

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
)

// --- Realistic entry shape ---

// workloadChannel mirrors the layout of a distribution.channel.Channel
// closely enough that GC behavior is representative of production tables.
// Key features that matter for the measurements:
//
//   - uint32 Key (no internal pointers — good baseline for primitive keys).
//   - String Name and DataType (pointer-rich; contributes string headers
//     to the scannable heap via both the forward index storage and the
//     Lookup.reverse map).
//   - Long Expression string on a subset of entries (calculated channels
//     only), so the workload mimics the heterogeneity of real tables where
//     a minority of rows carry the bulk of the string weight.
//   - Bool flags (internal, virtual, is_index) so index-by-bool storage
//     (boolLookupStorage) can be exercised under workload conditions.
type workloadChannel struct {
	Key         uint32
	Name        string
	DataType    string
	IsIndex     bool
	IsVirtual   bool
	Internal    bool
	Leaseholder uint32
	LocalIndex  uint32
	Expression  string
}

func (c workloadChannel) GorpKey() uint32   { return c.Key }
func (c workloadChannel) SetOptions() []any { return nil }

// --- Workload-scale parameters ---

// workloadSizes spans the range where GC behavior actually changes shape.
// Under 1k rows the reverse map fits in a handful of buckets and scan cost
// is noise; above 100k the mark phase starts contributing meaningful
// wall-clock time to each GC cycle; 1M is the aspirational upper bound for
// a large cluster.
var workloadSizes = []int{1_000, 10_000, 100_000}

// workloadDuration is the wall-clock target for each workload run. Long
// enough to trigger several dozen GC cycles at typical allocation rates,
// short enough that the whole suite fits in a few minutes.
const workloadDuration = 3 * time.Second

// calculatedExpressionFraction is the proportion of rows that carry a
// non-empty Expression string. Set to match the rough ratio of calculated
// to regular channels in production tables.
const calculatedExpressionFraction = 0.05

// nameCardinalityFraction controls the spread of the Name field. A value
// of 1.0 means every row has a unique name; 0.1 means there are only N/10
// distinct names (each matching 10 rows). Used by composition benchmarks
// to control how wide the index-backed candidate set gets.
const nameCardinalityFraction = 1.0

// dataTypeCardinality is the number of distinct DataType values. Kept
// small to mirror the handful of canonical types in telem.
const dataTypeCardinality = 8

// --- Realistic entry generation ---

func makeWorkloadChannels(n int) []workloadChannel {
	out := make([]workloadChannel, n)
	nameCardinality := max(1, int(float64(n)*nameCardinalityFraction))
	for i := range out {
		isCalculated := (i % int(1.0/calculatedExpressionFraction)) == 0
		isIndexCh := (i%17 == 0) && !isCalculated
		isVirtual := isCalculated
		internal := (i%23 == 0)
		out[i] = workloadChannel{
			Key:         uint32(i + 1),
			Name:        "channel-" + strconv.Itoa(i%nameCardinality),
			DataType:    "type-" + strconv.Itoa(i%dataTypeCardinality),
			IsIndex:     isIndexCh,
			IsVirtual:   isVirtual,
			Internal:    internal,
			Leaseholder: uint32((i % 8) + 1),
		}
		if isIndexCh {
			out[i].LocalIndex = out[i].Key
		}
		if isCalculated {
			// A representative calculated-channel expression: long enough
			// to contribute meaningful string weight to the heap without
			// being pathologically huge.
			out[i].Expression = "TEMP_" + strconv.Itoa(i) +
				" * 1.8 + 32.0 + offset_" + strconv.Itoa(i%100) +
				" - calibration_" + strconv.Itoa(i%50) +
				" / scale_factor_" + strconv.Itoa(i%25)
		}
	}
	return out
}

// --- Metrics ---

// workloadMetrics captures the GC-centric view of a workload run. It's
// computed by diffing runtime.MemStats before and after, plus counters
// incremented inside the workload itself.
type workloadMetrics struct {
	duration     time.Duration
	opCount      int64
	readCount    int64
	writeCount   int64
	queryCount   int64
	createCount  int64
	deleteCount  int64
	memStart     runtime.MemStats
	memEnd       runtime.MemStats
	memPostForce runtime.MemStats
}

func (m workloadMetrics) allocBytesDelta() uint64 {
	return m.memEnd.TotalAlloc - m.memStart.TotalAlloc
}

func (m workloadMetrics) allocObjectsDelta() uint64 {
	return m.memEnd.Mallocs - m.memStart.Mallocs
}

func (m workloadMetrics) gcCountDelta() uint32 {
	return m.memEnd.NumGC - m.memStart.NumGC
}

func (m workloadMetrics) pauseTotalDelta() time.Duration {
	return time.Duration(m.memEnd.PauseTotalNs - m.memStart.PauseTotalNs)
}

// maxPauseDuringRun walks the last 256 GC pauses in the MemStats circular
// buffer and returns the largest one that fell inside the workload
// window. A simpler "end.MaxPauseNs" doesn't exist in MemStats, and
// PauseNs is the only source of per-cycle pause data.
func (m workloadMetrics) maxPauseDuringRun() time.Duration {
	var maxPause uint64
	delta := m.memEnd.NumGC - m.memStart.NumGC
	if delta == 0 {
		return 0
	}
	// PauseNs is a 256-entry circular buffer indexed by (NumGC+255)%256.
	// Walk the entries that correspond to GCs that happened during the
	// run, bounded by 256 (older entries are overwritten).
	count := min(delta, 256)
	for i := range count {
		idx := (m.memEnd.NumGC + 255 - i) % 256
		if p := m.memEnd.PauseNs[idx]; p > maxPause {
			maxPause = p
		}
	}
	return time.Duration(maxPause)
}

func (m workloadMetrics) pauseFraction() float64 {
	if m.duration == 0 {
		return 0
	}
	return float64(m.pauseTotalDelta()) / float64(m.duration)
}

func (m workloadMetrics) opsPerSecond() float64 {
	if m.duration == 0 {
		return 0
	}
	return float64(m.opCount) / m.duration.Seconds()
}

// liveHeapAfterForcedGC reports the heap size after a full GC cycle at
// the end of the run. This is the closest proxy to "how big is the
// permanently reachable state" without dumping a full heap profile.
func (m workloadMetrics) liveHeapAfterForcedGC() uint64 {
	return m.memPostForce.HeapAlloc
}

// report writes every metric through b.ReportMetric so benchstat can
// diff runs across implementations.
func (m workloadMetrics) report(b *testing.B) {
	b.Helper()
	b.ReportMetric(m.opsPerSecond(), "ops/s")
	b.ReportMetric(float64(m.readCount), "reads")
	b.ReportMetric(float64(m.writeCount), "writes")
	b.ReportMetric(float64(m.queryCount), "queries")
	b.ReportMetric(float64(m.createCount), "creates")
	b.ReportMetric(float64(m.deleteCount), "deletes")
	b.ReportMetric(float64(m.allocBytesDelta())/float64(m.opCount), "B/op")
	b.ReportMetric(float64(m.allocObjectsDelta())/float64(m.opCount), "allocs/op")
	b.ReportMetric(float64(m.gcCountDelta()), "gcs")
	b.ReportMetric(float64(m.pauseTotalDelta().Microseconds()), "pauseμs")
	b.ReportMetric(float64(m.maxPauseDuringRun().Microseconds()), "maxμs")
	b.ReportMetric(m.pauseFraction()*100, "pause%")
	b.ReportMetric(float64(m.liveHeapAfterForcedGC())/(1<<20), "heapMB")
}

// --- Workload context ---

// workloadContext bundles the state a workload implementation needs to
// drive operations against a pre-populated table. Workloads run until
// ctx is cancelled, incrementing the counters on every op so the harness
// can compute throughput afterwards.
type workloadContext struct {
	ctx     context.Context
	db      *gorp.DB
	table   *gorp.Table[uint32, workloadChannel]
	nameIdx *gorp.Lookup[uint32, workloadChannel, string]
	typeIdx *gorp.Lookup[uint32, workloadChannel, string]
	// prefilled is the set of keys we created during setup, used by
	// workloads that need to read/delete from a known population. Read
	// in a round-robin fashion via a per-worker cursor.
	prefilled []uint32
	// stop is closed by the harness to signal workload workers to exit.
	stop chan struct{}
	// Atomic counters — workers increment these on every operation.
	opCount     atomic.Int64
	readCount   atomic.Int64
	writeCount  atomic.Int64
	queryCount  atomic.Int64
	createCount atomic.Int64
	deleteCount atomic.Int64
}

// --- Workload runner ---

type workloadFn func(wc *workloadContext, workerID int)

type workloadConfig struct {
	// prefillSize is the number of rows created before the workload
	// starts. Measured in workloadChannel entries.
	prefillSize int
	// duration is the wall-clock window over which the workload runs.
	duration time.Duration
	// workers is the number of goroutines driving the workload
	// concurrently. Each gets a unique ID passed to the workloadFn.
	workers int
	// indexes controls which indexes are registered on the table. Used
	// to baseline "no indexes" runs against "indexes registered" runs,
	// so we can attribute GC cost to the index infrastructure specifically.
	indexes func() (
		name *gorp.Lookup[uint32, workloadChannel, string],
		typ *gorp.Lookup[uint32, workloadChannel, string],
	)
}

func runWorkload(
	b *testing.B,
	cfg workloadConfig,
	fn workloadFn,
) {
	b.Helper()
	// Optional runtime/trace capture, controlled by env var. One file per
	// process invocation; sub-benches overwrite.
	if tracePath := os.Getenv("GORP_WORKLOAD_TRACE"); tracePath != "" {
		f, err := os.Create(tracePath)
		if err != nil {
			b.Fatalf("create trace file: %v", err)
		}
		if err := trace.Start(f); err != nil {
			b.Fatalf("start trace: %v", err)
		}
		defer func() {
			trace.Stop()
			_ = f.Close()
		}()
	}

	db := gorp.Wrap(memkv.New())
	defer func() { _ = db.Close() }()
	setupCtx := context.Background()

	var (
		nameIdx *gorp.Lookup[uint32, workloadChannel, string]
		typeIdx *gorp.Lookup[uint32, workloadChannel, string]
	)
	var indexList []gorp.Index[uint32, workloadChannel]
	if cfg.indexes != nil {
		nameIdx, typeIdx = cfg.indexes()
		if nameIdx != nil {
			indexList = append(indexList, nameIdx)
		}
		if typeIdx != nil {
			indexList = append(indexList, typeIdx)
		}
	}
	table, err := gorp.OpenTable(setupCtx, gorp.TableConfig[uint32, workloadChannel]{
		DB:      db,
		Indexes: indexList,
	})
	if err != nil {
		b.Fatal(err)
	}
	defer func() { _ = table.Close() }()

	// Prefill. This population lives for the whole run and is what the
	// steady-state GC scan has to walk — anything we want in the scan
	// cost gets created here, before the measurement window opens.
	prefilled := makeWorkloadChannels(cfg.prefillSize)
	if err := gorp.NewCreate[uint32, workloadChannel]().
		Entries(&prefilled).
		Exec(setupCtx, db); err != nil {
		b.Fatal(err)
	}
	prefilledKeys := make([]uint32, len(prefilled))
	for i, c := range prefilled {
		prefilledKeys[i] = c.Key
	}
	// Drop the local slice reference so the prefill slice itself doesn't
	// keep the scan cost artificially inflated. The KV store still holds
	// the entries via its own codec path.
	prefilled = nil

	runCtx, cancel := context.WithTimeout(context.Background(), cfg.duration)
	defer cancel()
	wc := &workloadContext{
		ctx:       runCtx,
		db:        db,
		table:     table,
		nameIdx:   nameIdx,
		typeIdx:   typeIdx,
		prefilled: prefilledKeys,
		stop:      make(chan struct{}),
	}

	// Stabilize before measurement: force a full GC so the mark phase
	// isn't still catching up on prefill allocations when we start.
	runtime.GC()
	runtime.GC()

	var metrics workloadMetrics
	runtime.ReadMemStats(&metrics.memStart)
	start := time.Now()

	var wg sync.WaitGroup
	wg.Add(cfg.workers)
	for w := 0; w < cfg.workers; w++ {
		go func(id int) {
			defer wg.Done()
			fn(wc, id)
		}(w)
	}
	<-runCtx.Done()
	close(wc.stop)
	wg.Wait()

	metrics.duration = time.Since(start)
	runtime.ReadMemStats(&metrics.memEnd)

	// Force a GC and re-read MemStats so HeapAlloc reflects live state,
	// not garbage that hasn't been swept yet. This is the "how much of
	// the heap is actually load-bearing" snapshot.
	runtime.GC()
	runtime.ReadMemStats(&metrics.memPostForce)

	// Optional heap profile dump.
	if heapPath := os.Getenv("GORP_WORKLOAD_HEAP"); heapPath != "" {
		f, err := os.Create(heapPath)
		if err != nil {
			b.Fatalf("create heap file: %v", err)
		}
		if err := pprof.WriteHeapProfile(f); err != nil {
			_ = f.Close()
			b.Fatalf("write heap profile: %v", err)
		}
		_ = f.Close()
	}

	metrics.opCount = wc.opCount.Load()
	metrics.readCount = wc.readCount.Load()
	metrics.writeCount = wc.writeCount.Load()
	metrics.queryCount = wc.queryCount.Load()
	metrics.createCount = wc.createCount.Load()
	metrics.deleteCount = wc.deleteCount.Load()
	metrics.report(b)
}

// --- Workload implementations ---

// steadyStateQueriesIndexed runs a single-value indexed lookup in a tight
// loop. Every op walks: Filter construction → execKeys → GetMany → decode.
// The prefilled set is read round-robin per worker so cache hits don't
// artificially compress the working set.
func steadyStateQueriesIndexed(wc *workloadContext, workerID int) {
	cursor := workerID
	for {
		select {
		case <-wc.stop:
			return
		default:
		}
		key := wc.prefilled[cursor%len(wc.prefilled)]
		cursor++
		name := "channel-" + strconv.FormatUint(uint64(key-1), 10)
		var out []workloadChannel
		if err := wc.table.NewRetrieve().
			Where(wc.nameIdx.Filter(name)).
			Entries(&out).
			Exec(wc.ctx, wc.db); err != nil {
			return
		}
		wc.opCount.Add(1)
		wc.queryCount.Add(1)
	}
}

// steadyStateQueriesScan is the baseline for steadyStateQueriesIndexed:
// same logical query, but routed through gorp.Match so it falls back to
// execFilter (full table scan). Lets us quantify the GC cost of the
// index fast path relative to the scan path on the same workload shape.
func steadyStateQueriesScan(wc *workloadContext, workerID int) {
	cursor := workerID
	for {
		select {
		case <-wc.stop:
			return
		default:
		}
		key := wc.prefilled[cursor%len(wc.prefilled)]
		cursor++
		name := "channel-" + strconv.FormatUint(uint64(key-1), 10)
		var out []workloadChannel
		if err := wc.table.NewRetrieve().
			Where(gorp.Match(func(_ gorp.Context, c *workloadChannel) (bool, error) {
				return c.Name == name, nil
			})).
			Entries(&out).
			Exec(wc.ctx, wc.db); err != nil {
			return
		}
		wc.opCount.Add(1)
		wc.queryCount.Add(1)
	}
}

// steadyStateReadsByKey exercises the WhereKeys path, which is the fastest
// retrieval route in gorp (no filter, direct KV point reads). Baseline for
// "best case" GC pressure on the read path — anything the index path
// adds over this is attributable to index infrastructure.
func steadyStateReadsByKey(wc *workloadContext, workerID int) {
	cursor := workerID
	for {
		select {
		case <-wc.stop:
			return
		default:
		}
		key := wc.prefilled[cursor%len(wc.prefilled)]
		cursor++
		var out workloadChannel
		if err := wc.table.NewRetrieve().
			WhereKeys(key).
			Entry(&out).
			Exec(wc.ctx, wc.db); err != nil {
			return
		}
		wc.opCount.Add(1)
		wc.readCount.Add(1)
	}
}

// writeHeavy creates fresh rows as fast as possible. Each op allocates a
// new entry, goes through the codec, and triggers observer fan-out into
// every registered index (name + type). Dominant allocation path:
// keyCodec encode, value encode, observer dispatch, index set.
func writeHeavy(wc *workloadContext, workerID int) {
	// Use a worker-local counter so writers don't collide on keys.
	next := uint32(workerID*10_000_000) + uint32(len(wc.prefilled)) + 1
	for {
		select {
		case <-wc.stop:
			return
		default:
		}
		entry := workloadChannel{
			Key:         next,
			Name:        "new-" + strconv.FormatUint(uint64(next), 10),
			DataType:    "type-" + strconv.Itoa(int(next%dataTypeCardinality)),
			Leaseholder: 1,
		}
		next++
		if err := gorp.NewCreate[uint32, workloadChannel]().
			Entry(&entry).
			Exec(wc.ctx, wc.db); err != nil {
			return
		}
		wc.opCount.Add(1)
		wc.writeCount.Add(1)
		wc.createCount.Add(1)
	}
}

// mixedReadWrite runs the most realistic workload: 80% indexed queries,
// 20% writes. Each worker picks its role by workerID parity so the split
// is deterministic regardless of scheduler timing.
func mixedReadWrite(wc *workloadContext, workerID int) {
	// Worker 0-3: readers. Worker 4: writer. This gives 80/20 at the
	// per-worker level, which is a close approximation of the op-level
	// ratio for workloads where reads are much cheaper than writes.
	if workerID < 4 {
		steadyStateQueriesIndexed(wc, workerID)
		return
	}
	writeHeavy(wc, workerID)
}

// churn keeps the table population constant by alternating create and
// delete. Designed to expose the "Go maps never shrink" behavior: over
// many cycles, the reverse map and forward storage accumulate bucket
// capacity proportional to the peak size, even though the live row count
// stays flat. Live heap after forced GC should stay roughly constant
// across runs; if it grows, that's evidence of map bloat.
func churn(wc *workloadContext, workerID int) {
	next := uint32(workerID*10_000_000) + uint32(len(wc.prefilled)) + 1
	const batchSize = 64
	for {
		select {
		case <-wc.stop:
			return
		default:
		}
		// Create a batch.
		batch := make([]workloadChannel, batchSize)
		for i := range batch {
			batch[i] = workloadChannel{
				Key:         next,
				Name:        "churn-" + strconv.FormatUint(uint64(next), 10),
				DataType:    "type-" + strconv.Itoa(int(next%dataTypeCardinality)),
				Leaseholder: 1,
			}
			next++
		}
		if err := gorp.NewCreate[uint32, workloadChannel]().
			Entries(&batch).
			Exec(wc.ctx, wc.db); err != nil {
			return
		}
		// Immediately delete the batch we just created. Net population
		// stays at the prefill size, but the maps see 2N observer
		// operations (N sets + N deletes) per cycle.
		keys := make([]uint32, batchSize)
		for i, c := range batch {
			keys[i] = c.Key
		}
		if err := gorp.NewDelete[uint32, workloadChannel]().
			WhereKeys(keys...).
			Exec(wc.ctx, wc.db); err != nil {
			return
		}
		wc.opCount.Add(int64(2 * batchSize))
		wc.createCount.Add(int64(batchSize))
		wc.deleteCount.Add(int64(batchSize))
		wc.writeCount.Add(int64(2 * batchSize))
	}
}

// compositionQueries runs Where(And(nameIdx, typeIdx)) in a tight loop so
// we can measure the allocation cost of intersectKeys + rebuildMembership
// under load. Uses a narrow name and a wider type so the intersection
// has to actually walk, not trivially empty out.
func compositionQueries(wc *workloadContext, workerID int) {
	cursor := workerID
	for {
		select {
		case <-wc.stop:
			return
		default:
		}
		key := wc.prefilled[cursor%len(wc.prefilled)]
		cursor++
		name := "channel-" + strconv.FormatUint(uint64(key-1), 10)
		typ := "type-" + strconv.Itoa(int((key-1)%dataTypeCardinality))
		var out []workloadChannel
		if err := wc.table.NewRetrieve().
			Where(gorp.And(
				wc.nameIdx.Filter(name),
				wc.typeIdx.Filter(typ),
			)).
			Entries(&out).
			Exec(wc.ctx, wc.db); err != nil {
			return
		}
		wc.opCount.Add(1)
		wc.queryCount.Add(1)
	}
}

// --- Index constructors for runWorkload.indexes ---

// withBothIndexes returns a closure that constructs a fresh pair of
// (name, type) Lookup indexes. Used by any workload that needs the full
// index machinery registered.
func withBothIndexes() (
	*gorp.Lookup[uint32, workloadChannel, string],
	*gorp.Lookup[uint32, workloadChannel, string],
) {
	return gorp.NewLookup[uint32, workloadChannel, string](
			"name",
			func(c *workloadChannel) string { return c.Name },
		),
		gorp.NewLookup[uint32, workloadChannel, string](
			"type",
			func(c *workloadChannel) string { return c.DataType },
		)
}

// withNameIndexOnly returns just a name index. Used by workloads that
// don't exercise the type index so we isolate the scan/allocation cost
// of a single index.
func withNameIndexOnly() (
	*gorp.Lookup[uint32, workloadChannel, string],
	*gorp.Lookup[uint32, workloadChannel, string],
) {
	return gorp.NewLookup[uint32, workloadChannel, string](
		"name",
		func(c *workloadChannel) string { return c.Name },
	), nil
}

// --- Benchmark entry points ---
//
// Each benchmark follows the same structure: iterate over workloadSizes,
// construct a workloadConfig, hand it to runWorkload. The workload fn
// determines what operations run inside the measurement window.

func BenchmarkWorkload_SteadyStateQueriesIndexed(b *testing.B) {
	for _, size := range workloadSizes {
		b.Run(fmt.Sprintf("n=%d", size), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				runWorkload(b, workloadConfig{
					prefillSize: size,
					duration:    workloadDuration,
					workers:     4,
					indexes: func() (
						*gorp.Lookup[uint32, workloadChannel, string],
						*gorp.Lookup[uint32, workloadChannel, string],
					) {
						return withNameIndexOnly()
					},
				}, steadyStateQueriesIndexed)
			}
		})
	}
}

func BenchmarkWorkload_SteadyStateQueriesScan(b *testing.B) {
	for _, size := range workloadSizes {
		b.Run(fmt.Sprintf("n=%d", size), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				runWorkload(b, workloadConfig{
					prefillSize: size,
					duration:    workloadDuration,
					workers:     4,
					indexes:     nil,
				}, steadyStateQueriesScan)
			}
		})
	}
}

func BenchmarkWorkload_SteadyStateReadsByKey(b *testing.B) {
	for _, size := range workloadSizes {
		b.Run(fmt.Sprintf("n=%d", size), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				runWorkload(b, workloadConfig{
					prefillSize: size,
					duration:    workloadDuration,
					workers:     4,
					indexes: func() (
						*gorp.Lookup[uint32, workloadChannel, string],
						*gorp.Lookup[uint32, workloadChannel, string],
					) {
						return withNameIndexOnly()
					},
				}, steadyStateReadsByKey)
			}
		})
	}
}

func BenchmarkWorkload_WriteHeavy(b *testing.B) {
	for _, size := range workloadSizes {
		b.Run(fmt.Sprintf("n=%d", size), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				runWorkload(b, workloadConfig{
					prefillSize: size,
					duration:    workloadDuration,
					workers:     2,
					indexes: func() (
						*gorp.Lookup[uint32, workloadChannel, string],
						*gorp.Lookup[uint32, workloadChannel, string],
					) {
						return withBothIndexes()
					},
				}, writeHeavy)
			}
		})
	}
}

func BenchmarkWorkload_WriteHeavyNoIndexes(b *testing.B) {
	// Baseline: same write workload with no indexes registered. The
	// delta between this and BenchmarkWorkload_WriteHeavy is the GC cost
	// of the observer-driven index maintenance path.
	for _, size := range workloadSizes {
		b.Run(fmt.Sprintf("n=%d", size), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				runWorkload(b, workloadConfig{
					prefillSize: size,
					duration:    workloadDuration,
					workers:     2,
					indexes:     nil,
				}, writeHeavy)
			}
		})
	}
}

func BenchmarkWorkload_MixedReadWrite(b *testing.B) {
	for _, size := range workloadSizes {
		b.Run(fmt.Sprintf("n=%d", size), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				runWorkload(b, workloadConfig{
					prefillSize: size,
					duration:    workloadDuration,
					workers:     5, // 4 readers + 1 writer, per mixedReadWrite
					indexes: func() (
						*gorp.Lookup[uint32, workloadChannel, string],
						*gorp.Lookup[uint32, workloadChannel, string],
					) {
						return withNameIndexOnly()
					},
				}, mixedReadWrite)
			}
		})
	}
}

func BenchmarkWorkload_Churn(b *testing.B) {
	for _, size := range workloadSizes {
		b.Run(fmt.Sprintf("n=%d", size), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				runWorkload(b, workloadConfig{
					prefillSize: size,
					duration:    workloadDuration,
					workers:     2,
					indexes: func() (
						*gorp.Lookup[uint32, workloadChannel, string],
						*gorp.Lookup[uint32, workloadChannel, string],
					) {
						return withBothIndexes()
					},
				}, churn)
			}
		})
	}
}

func BenchmarkWorkload_Composition(b *testing.B) {
	for _, size := range workloadSizes {
		b.Run(fmt.Sprintf("n=%d", size), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				runWorkload(b, workloadConfig{
					prefillSize: size,
					duration:    workloadDuration,
					workers:     4,
					indexes: func() (
						*gorp.Lookup[uint32, workloadChannel, string],
						*gorp.Lookup[uint32, workloadChannel, string],
					) {
						return withBothIndexes()
					},
				}, compositionQueries)
			}
		})
	}
}

// BenchmarkWorkload_IndexesRegisteredNoTraffic is the cleanest attribution
// test for "how much does registering indexes cost at steady state with
// no traffic". Prefills the table, registers indexes, then sits idle for
// the measurement window. Any GC work that happens comes from mark-phase
// scans over the reverse maps and storage, nothing else. Useful as the
// denominator when computing "GC cost attributable to index
// infrastructure" for a given table size.
func BenchmarkWorkload_IndexesRegisteredNoTraffic(b *testing.B) {
	for _, size := range workloadSizes {
		b.Run(fmt.Sprintf("n=%d", size), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				runWorkload(b, workloadConfig{
					prefillSize: size,
					duration:    workloadDuration,
					workers:     1,
					indexes: func() (
						*gorp.Lookup[uint32, workloadChannel, string],
						*gorp.Lookup[uint32, workloadChannel, string],
					) {
						return withBothIndexes()
					},
				}, idle)
			}
		})
	}
}

// BenchmarkWorkload_NoIndexesNoTraffic is the comparison baseline for
// BenchmarkWorkload_IndexesRegisteredNoTraffic. Same prefill, same idle
// period, no indexes registered. Diff the two to isolate the steady-state
// GC cost of the index infrastructure itself.
func BenchmarkWorkload_NoIndexesNoTraffic(b *testing.B) {
	for _, size := range workloadSizes {
		b.Run(fmt.Sprintf("n=%d", size), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				runWorkload(b, workloadConfig{
					prefillSize: size,
					duration:    workloadDuration,
					workers:     1,
					indexes:     nil,
				}, idle)
			}
		})
	}
}

// idle sits waiting for the stop signal. Used by the "no traffic"
// benchmarks to measure pure steady-state GC cost.
func idle(wc *workloadContext, _ int) {
	// Force background GC pressure from some other goroutine to not
	// skew results: this worker itself allocates nothing. MemStats
	// still captures whatever the runtime does on its own (pprof
	// goroutines, runtime scheduler, etc.) but there's no workload
	// noise.
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-wc.stop:
			return
		case <-ticker.C:
			wc.opCount.Add(1)
		}
	}
}
