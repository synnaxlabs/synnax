// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package benchmark_test

import (
	"context"
	"flag"
	"fmt"
	"sync"
	"testing"

	"github.com/samber/lo"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/cesium/internal/testutil"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/control"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	"golang.org/x/sync/semaphore"
)

type BenchmarkConfig struct {
	domainsPerChannel int
	samplesPerDomain  int
	numIndexChannels  int
	numDataChannels   int
	numGoRoutines     int64
	usingMemFS        bool
}

type WriteBenchmarkConfig struct {
	BenchmarkConfig
	numWriters int
	// commitInterval is the number of domains written before the writer must commit.
	// set to -1 for never commit until the end
	commitInterval int
}

type StreamBenchmarkConfig struct {
	WriteBenchmarkConfig
	streamOnly bool
}

var (
	domainsPerChannel = flag.Int("d", 10, "domains per channel")
	samplesPerDomain  = flag.Int("s", 10, "samples per domain")
	numIndexChannels  = flag.Int("index", 10, "index channel count")
	numDataChannels   = flag.Int("data", 10, "data channel count")
	usingMemFS        = flag.Bool("mem", false, "memFS")
	numWriters        = flag.Int("w", 1, "writer count")
	numGoRoutines     = flag.Int64("g", 1, "goroutine count")
	streamOnly        = flag.Bool("only_stream", false, "writer streamOnly mode")
	commitInterval    = flag.Int("commit", -1, "writer commit interval")
	ctx               = context.TODO()
)

func BenchmarkCesium(b *testing.B) {
	var (
		fs  xfs.FS
		err error
	)

	benchCfg := BenchmarkConfig{
		domainsPerChannel: *domainsPerChannel,
		samplesPerDomain:  *samplesPerDomain,
		numIndexChannels:  *numIndexChannels,
		numDataChannels:   *numDataChannels,
		numGoRoutines:     *numGoRoutines,
		usingMemFS:        *usingMemFS,
	}
	writeCfg := WriteBenchmarkConfig{
		BenchmarkConfig: benchCfg,
		numWriters:      *numWriters,
		commitInterval:  *commitInterval,
	}
	streamCfg := StreamBenchmarkConfig{
		WriteBenchmarkConfig: writeCfg,
		streamOnly:           *streamOnly,
	}

	makeFS := testutil.FileSystemsWithoutAssertion[lo.Ternary(benchCfg.usingMemFS, "memFS", "osFS")]
	fs, cleanUp := makeFS()

	dataSeries, channels, keys := testutil.GenerateDataAndChannels(
		benchCfg.numIndexChannels,
		benchCfg.numDataChannels,
		benchCfg.samplesPerDomain,
	)

	b.Run("write", func(b *testing.B) { BenchWrite(b, writeCfg, dataSeries, channels, keys, fs) })
	b.Run("read", func(b *testing.B) { BenchRead(b, benchCfg, dataSeries, channels, keys, fs) })
	b.Run("stream", func(b *testing.B) {
		BenchStream(b, streamCfg, dataSeries, channels, keys, fs)
	})

	err = cleanUp()
	if err != nil {
		b.Errorf("Error during cleanup: %s", err)
	}
}

func BenchWrite(b *testing.B, cfg WriteBenchmarkConfig, dataSeries telem.Series, channels []cesium.Channel, keys []cesium.ChannelKey, fs xfs.FS) {
	for i := range b.N {
		b.StopTimer()
		var (
			wg                        = sync.WaitGroup{}
			numIndexChannelsPerWriter = cfg.numIndexChannels / cfg.numWriters
			sem                       = semaphore.NewWeighted(cfg.numGoRoutines)
		)

		db, err := cesium.Open(ctx, "benchmark_write_test", cesium.WithFS(fs))
		if err != nil {
			b.Errorf("Error during DB creation: %s", err)
		}

		err = db.CreateChannel(ctx, channels...)
		if err != nil {
			b.Errorf("Error during channel creation: %s", err)
		}

		b.StartTimer()

		for j := range cfg.numWriters {
			var writerChannels []cesium.ChannelKey

			// Filter out the index channels we are writing to
			if j == cfg.numWriters-1 {
				writerChannels = make([]cesium.ChannelKey, len(keys[j*numIndexChannelsPerWriter:cfg.numIndexChannels]))
				copy(writerChannels, keys[j*numIndexChannelsPerWriter:cfg.numIndexChannels])
			} else {
				writerChannels = make([]cesium.ChannelKey, numIndexChannelsPerWriter)
				copy(writerChannels, keys[j*numIndexChannelsPerWriter:(j+1)*numIndexChannelsPerWriter])
			}

			// Then find all the channels indexed by these index channels
			for k := cfg.numIndexChannels + 1; k < cfg.numIndexChannels+cfg.numDataChannels; k++ {
				if lo.Contains(writerChannels, cesium.ChannelKey(k%cfg.numIndexChannels+1)) {
					writerChannels = append(writerChannels, cesium.ChannelKey(k))
				}
			}

			// Then arbitrarily assign rate channels
			for k := cfg.numIndexChannels + cfg.numDataChannels + 1; k < cfg.numIndexChannels+cfg.numDataChannels+1; k++ {
				if k%cfg.numWriters == j {
					writerChannels = append(writerChannels, cesium.ChannelKey(k))
				}
			}

			wg.Add(1)
			if err = sem.Acquire(ctx, 1); err != nil {
				b.Errorf("Semaphore error %s", err)
			}

			if err != nil {
				b.Errorf("pprof error %s", err)
			}
			go func(writerChannels []cesium.ChannelKey, j int) {
				defer func() {
					wg.Done()
					sem.Release(1)
				}()
				var (
					commitCount                   = 0
					highWaterMark telem.TimeStamp = 0
					indexData                     = make([]telem.TimeStamp, cfg.samplesPerDomain)
					frame         cesium.Frame
				)

				w, err := db.OpenWriter(ctx, cesium.WriterConfig{
					ControlSubject: control.Subject{Key: fmt.Sprintf("bench_writer %d", j)},
					Start:          1 * telem.SecondTS,
					Channels:       writerChannels,
				})

				if err != nil {
					b.Errorf("Writer open error %s", err)
					return
				}

				// Prepare the frame for writing to channels
				for _, ch := range writerChannels {
					if ch > cesium.ChannelKey(cfg.numIndexChannels) {
						frame = frame.Append(ch, dataSeries)
					}
				}

				for k := range cfg.domainsPerChannel {
					// Generate the index data for this frame.
					for l := range cfg.samplesPerDomain {
						if l == 0 && k == 0 {
							indexData[l] = 0
							continue
						}
						indexData[l] = highWaterMark + telem.TimeStamp(l)*telem.SecondTS
					}
					highWaterMark += telem.TimeStamp(cfg.samplesPerDomain-1) * telem.SecondTS

					// Add the index data into frame / modify the index data in the frame
					if k == 0 {
						for _, ch := range writerChannels {
							if ch <= cesium.ChannelKey(cfg.numIndexChannels) {
								frame = frame.Append(ch, telem.NewSeries[telem.TimeStamp](indexData))
							}
						}
					} else {
						indexDataSeries := telem.NewSeries[telem.TimeStamp](indexData)
						for l := len(frame.KeysSlice()) - 1; l >= 0; l-- {
							if l > cfg.numIndexChannels {
								break
							}
							frame.SetSeriesAt(i, indexDataSeries)
						}
					}

					if _, err := w.Write(frame); err != nil {
						b.Error(err)
						return
					}

					if cfg.commitInterval != -1 {
						commitCount += 1
						if commitCount >= cfg.commitInterval {
							if _, err = w.Commit(); err != nil {
								b.Error(err)
								return
							}
							commitCount = 0
						}
					}
				}

				if _, err := w.Commit(); err != nil {
					b.Error("Commit failed")
					b.Error(err)
					return
				}

				err = w.Close()
				if err != nil {
					b.Errorf("Close error %s", err)
				}
			}(writerChannels, j)
		}

		wg.Wait()
		b.StopTimer()

		err = db.Close()
		if err != nil {
			b.Errorf("Error during db close: %s", err)
		}

		err = fs.Remove("benchmark_write_test")
		if err != nil {
			b.Errorf("Error during removing directory: %s", err)
		}

		b.StartTimer()
	}
}

func BenchRead(
	b *testing.B,
	cfg BenchmarkConfig,
	dataSeries telem.Series,
	channels []cesium.Channel,
	keys []cesium.ChannelKey,
	fs xfs.FS,
) {
	var (
		db            *cesium.DB
		err           error
		frame         cesium.Frame
		indexData                     = make([]telem.TimeStamp, cfg.samplesPerDomain)
		highWaterMark telem.TimeStamp = 0
	)

	db, err = cesium.Open(ctx, "benchmark_read_test", cesium.WithFS(fs))
	if err != nil {
		b.Errorf("Error during DB creation: %s", err)
	}
	err = db.CreateChannel(ctx, channels...)
	if err != nil {
		b.Errorf("Error during channel creation: %s", err)
	}

	w, err := db.OpenWriter(ctx, cesium.WriterConfig{
		ControlSubject: control.Subject{Key: "bench_reader"},
		Start:          1 * telem.SecondTS,
		Channels:       keys,
	})

	if err != nil {
		b.Errorf("Writer open error %s", err)
		return
	}

	// Prepare the frame for writing to channels
	for _, ch := range keys {
		if ch > cesium.ChannelKey(cfg.numIndexChannels) {
			frame = frame.Append(ch, dataSeries)
		}
	}

	for k := range cfg.domainsPerChannel {
		// Generate the index data for this frame.
		for l := range cfg.samplesPerDomain {
			if l == 0 && k == 0 {
				indexData[l] = 0
				continue
			}
			indexData[l] = highWaterMark + telem.TimeStamp(l)*telem.SecondTS
		}
		highWaterMark += telem.TimeStamp(cfg.samplesPerDomain-1) * telem.SecondTS

		// Add the index data into frame / modify the index data in the frame
		if k == 0 {
			for _, ch := range keys {
				if ch <= cesium.ChannelKey(cfg.numIndexChannels) {
					frame = frame.Append(ch, telem.NewSeries[telem.TimeStamp](indexData))
				}
			}
		} else {
			indexDataSeries := telem.NewSeries[telem.TimeStamp](indexData)
			for l := len(frame.KeysSlice()) - 1; l >= 0; l-- {
				if l > cfg.numIndexChannels {
					break
				}
				frame.SetSeriesAt(l, indexDataSeries)
			}
		}

		if _, err := w.Write(frame); err != nil {
			b.Error(err)
			return
		}
	}

	if _, err := w.Commit(); err != nil {
		b.Error("Commit failed")
		b.Error(err)
		return
	}

	if err = w.Close(); err != nil {
		b.Error(err)
	}

	for b.Loop() {
		_, err = db.Read(ctx, telem.TimeRangeMax, keys...)

		if err != nil {
			b.Error("Read failed")
			break
		}
	}

	err = db.Close()
	if err != nil {
		b.Errorf("Error during db close: %s", err)
	}

	err = fs.Remove("benchmark_read_test")
	if err != nil {
		b.Errorf("Error during removing directory: %s", err)
	}
}

func BenchStream(
	b *testing.B,
	cfg StreamBenchmarkConfig,
	dataSeries telem.Series,
	channels []cesium.Channel,
	keys []cesium.ChannelKey,
	fs xfs.FS,
) {
	for b.Loop() {
		b.StopTimer()
		var (
			wg                        = sync.WaitGroup{}
			numIndexChannelsPerWriter = cfg.numIndexChannels / cfg.numWriters
			sem                       = semaphore.NewWeighted(cfg.numGoRoutines)
		)

		db, err := cesium.Open(ctx, "benchmark_stream_test", cesium.WithFS(fs))
		if err != nil {
			b.Errorf("Error during DB creation: %s", err)
		}

		err = db.CreateChannel(ctx, channels...)
		if err != nil {
			b.Errorf("Error during channel creation: %s", err)
		}

		b.StartTimer()

		for j := range cfg.numWriters {
			var writerChannels []cesium.ChannelKey

			// Filter out the index channels we are writing to
			if j == cfg.numWriters-1 {
				writerChannels = make([]cesium.ChannelKey, len(keys[j*numIndexChannelsPerWriter:cfg.numIndexChannels]))
				copy(writerChannels, keys[j*numIndexChannelsPerWriter:cfg.numIndexChannels])
			} else {
				writerChannels = make([]cesium.ChannelKey, numIndexChannelsPerWriter)
				copy(writerChannels, keys[j*numIndexChannelsPerWriter:(j+1)*numIndexChannelsPerWriter])
			}

			// Then find all the channels indexed by these index channels
			for k := cfg.numIndexChannels + 1; k < cfg.numIndexChannels+cfg.numDataChannels; k++ {
				if lo.Contains(writerChannels, cesium.ChannelKey(k%cfg.numIndexChannels+1)) {
					writerChannels = append(writerChannels, cesium.ChannelKey(k))
				}
			}

			// Then arbitrarily assign rate channels
			for k := cfg.numIndexChannels + cfg.numDataChannels + 1; k < cfg.numIndexChannels+cfg.numDataChannels; k++ {
				if k%cfg.numWriters == j {
					writerChannels = append(writerChannels, cesium.ChannelKey(k))
				}
			}

			wg.Add(1)
			if err = sem.Acquire(ctx, 1); err != nil {
				b.Errorf("Semaphore error %s", err)
			}

			if err != nil {
				b.Errorf("pprof error %s", err)
			}
			go func(writerChannels []cesium.ChannelKey, j int) {
				defer func() {
					wg.Done()
					sem.Release(1)
				}()
				var (
					commitCount                   = 0
					highWaterMark telem.TimeStamp = 0
					indexData                     = make([]telem.TimeStamp, cfg.samplesPerDomain)
					frame         cesium.Frame
					w             *cesium.Writer
					s             cesium.Streamer[cesium.StreamerRequest, cesium.StreamerResponse]
				)

				w, err := db.OpenWriter(ctx, cesium.WriterConfig{
					ControlSubject: control.Subject{Key: fmt.Sprintf("bench_writer %d", j)},
					Start:          1 * telem.SecondTS,
					Channels:       writerChannels,
				})

				if err != nil {
					b.Errorf("Writer open error %s", err)
					return
				}

				s, err = db.NewStreamer(ctx, cesium.StreamerConfig{Channels: writerChannels})
				if err != nil {
					b.Errorf("Steramer open error")
				}

				iStream, oStream := confluence.Attach(s, 1)
				sCtx, cancel := signal.WithCancel(ctx)
				s.Flow(sCtx)

				// Prepare the frame for writing to channels
				for _, ch := range writerChannels {
					if ch > cesium.ChannelKey(cfg.numIndexChannels) {
						frame = frame.Append(ch, dataSeries)
					}
				}

				for k := range cfg.domainsPerChannel {
					// Generate the index data for this frame.
					for l := range cfg.samplesPerDomain {
						if l == 0 && k == 0 {
							indexData[l] = 0
							continue
						}
						indexData[l] = highWaterMark + telem.TimeStamp(l)*telem.SecondTS
					}
					highWaterMark += telem.TimeStamp(cfg.samplesPerDomain-1) * telem.SecondTS

					// Add the index data into frame / modify the index data in the frame
					if k == 0 {
						for _, ch := range writerChannels {
							if ch <= cesium.ChannelKey(cfg.numIndexChannels) {
								frame = frame.Append(ch, telem.NewSeries[telem.TimeStamp](indexData))
							}
						}
					} else {
						indexDataSeries := telem.NewSeries[telem.TimeStamp](indexData)
						for l := len(frame.KeysSlice()) - 1; l >= 0; l-- {
							if l > cfg.numIndexChannels {
								break
							}
							frame.SetSeriesAt(l, indexDataSeries)
						}
					}

					if _, err := w.Write(frame); err != nil {
						b.Error(err)
					}

					if cfg.commitInterval != -1 {
						commitCount += 1
						if commitCount >= cfg.commitInterval {
							if _, err = w.Commit(); err != nil {
								b.Error(err)
							}
							commitCount = 0
						}
					}

					<-oStream.Outlet()
				}

				if _, err := w.Commit(); err != nil {
					b.Error(err)
				}

				err = w.Close()
				if err != nil {
					b.Errorf("Close error %s", err)
				}

				iStream.Close()
				cancel()
			}(writerChannels, j)
		}

		wg.Wait()
		b.StopTimer()

		err = db.Close()
		if err != nil {
			b.Errorf("Error during db close: %s", err)
		}

		err = fs.Remove("benchmark_stream_test")
		if err != nil {
			b.Errorf("Error during removing directory: %s", err)
		}

		b.StartTimer()
	}
}
