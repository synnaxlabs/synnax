package cesium_test

import (
	"flag"
	"fmt"
	"github.com/samber/lo"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/cesium/internal/testutil"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/control"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	"golang.org/x/sync/semaphore"
	"sync"
	"testing"
)

type BenchmarkConfig struct {
	domainsPerChannel int
	samplesPerDomain  int
	numIndexChannels  int
	numDataChannels   int
	numRateChannels   int
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
	numIndexChannels  = flag.Int("index", 0, "index channel count")
	numDataChannels   = flag.Int("data", 0, "data channel count")
	numRateChannels   = flag.Int("rate", 0, "rate channel count")
	usingMemFS        = flag.Bool("mem", false, "memFS")
	numWriters        = flag.Int("w", 1, "writer count")
	numGoRoutines     = flag.Int64("g", 1, "goroutine count")
	streamOnly        = flag.Bool("only_stream", false, "writer streamOnly mode")
	commitInterval    = flag.Int("commit", -1, "writer commit interval")
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
		numRateChannels:   *numRateChannels,
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

	dataSeries, channels, keys := testutil.GenerateDataAndChannels(benchCfg.numIndexChannels, benchCfg.numDataChannels, benchCfg.numRateChannels, benchCfg.samplesPerDomain)

	b.Run("write", func(b *testing.B) { bench_write(b, writeCfg, dataSeries, channels, keys, fs) })
	b.Run("read", func(b *testing.B) { bench_read(b, benchCfg, dataSeries, channels, keys, fs) })
	b.Run("stream", func(b *testing.B) {
		bench_stream(b, streamCfg, dataSeries, channels, keys, fs)
	})

	err = cleanUp()
	if err != nil {
		b.Errorf("Error during cleanup: %s", err)
	}
}

func bench_write(b *testing.B, cfg WriteBenchmarkConfig, dataSeries telem.Series, channels []cesium.Channel, keys []cesium.ChannelKey, fs xfs.FS) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		var (
			wg                        = sync.WaitGroup{}
			numIndexChannelsPerWriter = cfg.numIndexChannels / cfg.numWriters
			sem                       = semaphore.NewWeighted(cfg.numGoRoutines)
		)

		db, err := cesium.Open("benchmark_write_test", cesium.WithFS(fs))
		if err != nil {
			b.Errorf("Error during DB creation: %s", err)
		}

		err = db.CreateChannel(ctx, channels...)
		if err != nil {
			b.Errorf("Error during channel creation: %s", err)
		}

		b.StartTimer()

		for j := 0; j < cfg.numWriters; j++ {
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
			for k := cfg.numIndexChannels + cfg.numDataChannels + 1; k < cfg.numIndexChannels+cfg.numDataChannels+cfg.numRateChannels+1; k++ {
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
					commitCount                 = 0
					hwm         telem.TimeStamp = 0
					indexData                   = make([]telem.TimeStamp, cfg.samplesPerDomain)
					frame       cesium.Frame
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

				for k := 0; k < cfg.domainsPerChannel; k++ {
					// Generate the index data for this frame.
					for l := 0; l < cfg.samplesPerDomain; l++ {
						if l == 0 && k == 0 {
							indexData[l] = 0
							continue
						}
						indexData[l] = hwm + telem.TimeStamp(l)*telem.SecondTS
					}
					hwm += telem.TimeStamp(cfg.samplesPerDomain-1) * telem.SecondTS

					// Add the index data into frame / modify the index data in the frame
					if k == 0 {
						for _, ch := range writerChannels {
							if ch <= cesium.ChannelKey(cfg.numIndexChannels) {
								frame = frame.Append(ch, telem.NewSeries[telem.TimeStamp](indexData))
							}
						}
					} else {
						indexDataSeries := telem.NewSeries[telem.TimeStamp](indexData)
						for l := len(frame.Keys) - 1; l >= 0; l-- {
							if l > cfg.numIndexChannels {
								break
							}
							frame.Series[l] = indexDataSeries
						}
					}

					ok := w.Write(frame)
					if !ok {
						b.Error(w.Error())
						return
					}

					if cfg.commitInterval != -1 {
						commitCount += 1
						if commitCount >= cfg.commitInterval {
							_, ok = w.Commit()
							if !ok {
								b.Error(w.Error())
								return
							}
							commitCount = 0
						}
					}
				}

				_, ok := w.Commit()
				if !ok {
					b.Error("Commit failed")
					b.Error(w.Error())
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

func bench_read(b *testing.B, cfg BenchmarkConfig, dataSeries telem.Series, channels []cesium.Channel, keys []cesium.ChannelKey, fs xfs.FS) {
	var (
		db        *cesium.DB
		err       error
		frame     cesium.Frame
		indexData                 = make([]telem.TimeStamp, cfg.samplesPerDomain)
		hwm       telem.TimeStamp = 0
	)

	db, err = cesium.Open("benchmark_read_test", cesium.WithFS(fs))
	err = db.CreateChannel(ctx, channels...)
	if err != nil {
		b.Errorf("Error during channel creation: %s", err)
	}

	w, err := db.OpenWriter(ctx, cesium.WriterConfig{
		ControlSubject: control.Subject{Key: fmt.Sprintf("bench_reader")},
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

	for k := 0; k < cfg.domainsPerChannel; k++ {
		// Generate the index data for this frame.
		for l := 0; l < cfg.samplesPerDomain; l++ {
			if l == 0 && k == 0 {
				indexData[l] = 0
				continue
			}
			indexData[l] = hwm + telem.TimeStamp(l)*telem.SecondTS
		}
		hwm += telem.TimeStamp(cfg.samplesPerDomain-1) * telem.SecondTS

		// Add the index data into frame / modify the index data in the frame
		if k == 0 {
			for _, ch := range keys {
				if ch <= cesium.ChannelKey(cfg.numIndexChannels) {
					frame = frame.Append(ch, telem.NewSeries[telem.TimeStamp](indexData))
				}
			}
		} else {
			indexDataSeries := telem.NewSeries[telem.TimeStamp](indexData)
			for l := len(frame.Keys) - 1; l >= 0; l-- {
				if l > cfg.numIndexChannels {
					break
				}
				frame.Series[l] = indexDataSeries
			}
		}

		ok := w.Write(frame)
		if !ok {
			b.Error(w.Error())
			return
		}
	}

	_, ok := w.Commit()
	if !ok {
		b.Error(w.Error())
	}

	if err = w.Close(); err != nil {
		b.Error(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
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

func bench_stream(b *testing.B, cfg StreamBenchmarkConfig, dataSeries telem.Series, channels []cesium.Channel, keys []cesium.ChannelKey, fs xfs.FS) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		var (
			wg                        = sync.WaitGroup{}
			numIndexChannelsPerWriter = cfg.numIndexChannels / cfg.numWriters
			sem                       = semaphore.NewWeighted(cfg.numGoRoutines)
		)

		db, err := cesium.Open("benchmark_stream_test", cesium.WithFS(fs))
		if err != nil {
			b.Errorf("Error during DB creation: %s", err)
		}

		err = db.CreateChannel(ctx, channels...)
		if err != nil {
			b.Errorf("Error during channel creation: %s", err)
		}

		b.StartTimer()

		for j := 0; j < cfg.numWriters; j++ {
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
			for k := cfg.numIndexChannels + cfg.numDataChannels + 1; k < cfg.numIndexChannels+cfg.numDataChannels+cfg.numRateChannels+1; k++ {
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
					commitCount                 = 0
					hwm         telem.TimeStamp = 0
					indexData                   = make([]telem.TimeStamp, cfg.samplesPerDomain)
					frame       cesium.Frame
					w           *cesium.Writer
					s           cesium.Streamer
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

				for k := 0; k < cfg.domainsPerChannel; k++ {
					// Generate the index data for this frame.
					for l := 0; l < cfg.samplesPerDomain; l++ {
						if l == 0 && k == 0 {
							indexData[l] = 0
							continue
						}
						indexData[l] = hwm + telem.TimeStamp(l)*telem.SecondTS
					}
					hwm += telem.TimeStamp(cfg.samplesPerDomain-1) * telem.SecondTS

					// Add the index data into frame / modify the index data in the frame
					if k == 0 {
						for _, ch := range writerChannels {
							if ch <= cesium.ChannelKey(cfg.numIndexChannels) {
								frame = frame.Append(ch, telem.NewSeries[telem.TimeStamp](indexData))
							}
						}
					} else {
						indexDataSeries := telem.NewSeries[telem.TimeStamp](indexData)
						for l := len(frame.Keys) - 1; l >= 0; l-- {
							if l > cfg.numIndexChannels {
								break
							}
							frame.Series[l] = indexDataSeries
						}
					}

					ok := w.Write(frame)
					if !ok {
						b.Error(w.Error())
						return
					}

					if cfg.commitInterval != -1 {
						commitCount += 1
						if commitCount >= cfg.commitInterval {
							_, ok = w.Commit()
							if !ok {
								b.Error(w.Error())
								return
							}
							commitCount = 0
						}
					}

					<-oStream.Outlet()
				}

				_, ok := w.Commit()
				if !ok {
					b.Error("Commit failed")
					b.Error(w.Error())
					return
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
