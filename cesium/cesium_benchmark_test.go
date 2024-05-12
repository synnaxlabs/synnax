package cesium_test

import (
	"flag"
	"fmt"
	"github.com/samber/lo"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/control"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/testutil"
	"golang.org/x/sync/semaphore"
	"sync"
	"testing"
)

type CesiumBenchmarkConfig struct {
	domainsPerChannel int
	samplesPerDomain  int
	numIndexChannels  int
	numDataChannels   int
	numRateChannels   int
	numGoRoutines     int64
	usingMemFS        bool
}

type CesiumWriteBenchmarkConfig struct {
	CesiumBenchmarkConfig
	numWriters int
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
)

func BenchmarkCesium(b *testing.B) {
	var (
		cfg CesiumWriteBenchmarkConfig
		fs  xfs.FS
		err error
	)

	cfg = CesiumWriteBenchmarkConfig{
		CesiumBenchmarkConfig: CesiumBenchmarkConfig{
			domainsPerChannel: *domainsPerChannel,
			samplesPerDomain:  *samplesPerDomain,
			numIndexChannels:  *numIndexChannels,
			numDataChannels:   *numDataChannels,
			numRateChannels:   *numRateChannels,
			numGoRoutines:     *numGoRoutines,
			usingMemFS:        *usingMemFS,
		},
		numWriters: *numWriters,
	}

	fsMaker, ok := fileSystems[lo.Ternary(cfg.usingMemFS, "memFS", "osFS")]
	fs = fsMaker()
	if !ok {
		b.Error("Cannot find osFS in file systems")
	}

	frames, channels, keys := testutil.GenerateFrameAndChannels(cfg.numIndexChannels, cfg.numDataChannels, cfg.numRateChannels, cfg.domainsPerChannel, cfg.samplesPerDomain)

	b.Run("write", func(b *testing.B) { bench_write(b, cfg, frames, channels, keys, fs) })
	b.Run("read", func(b *testing.B) { bench_read(b, frames, channels, keys, fs) })
	b.Run("stream", func(b *testing.B) { bench_stream(b, frames, channels, keys, fs) })

	err = cleanUp()
	if err != nil {
		b.Errorf("Error during cleanup: %s", err)
	}
}

func bench_write(b *testing.B, cfg CesiumWriteBenchmarkConfig, frames []cesium.Frame, channels []cesium.Channel, keys []cesium.ChannelKey, fs xfs.FS) {
	for i := 0; i < b.N; i++ {
		var (
			err                       error
			db                        *cesium.DB
			wg                        = sync.WaitGroup{}
			numIndexChannelsPerWriter = cfg.numIndexChannels / cfg.numWriters
			sem                       = semaphore.NewWeighted(cfg.numGoRoutines)
		)

		db, err = cesium.Open("benchmark_write_test", cesium.WithFS(fs))
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

			go func(writerChannels []cesium.ChannelKey, j int) {
				defer func() {
					wg.Done()
					sem.Release(1)
				}()
				var w *cesium.Writer

				w, err = db.OpenWriter(ctx, cesium.WriterConfig{
					ControlSubject: control.Subject{Key: fmt.Sprintf("bench_writer %d", j)},
					Start:          1 * telem.SecondTS,
					Channels:       writerChannels,
				})

				if err != nil {
					b.Errorf("Writer open error %s", err)
					return
				}

				for _, frame := range frames {
					ok := w.Write(frame.FilterKeys(writerChannels))
					if !ok {
						b.Error(w.Error())
						return
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
	}
}

func bench_read(b *testing.B, frames []cesium.Frame, channels []cesium.Channel, keys []cesium.ChannelKey, fs xfs.FS) {
	var (
		db  *cesium.DB
		err error
	)

	db, err = cesium.Open("benchmark_read_test", cesium.WithFS(fs))
	err = db.CreateChannel(ctx, channels...)
	if err != nil {
		b.Errorf("Error during channel creation: %s", err)
	}

	w, err := db.OpenWriter(ctx, cesium.WriterConfig{Start: 1 * telem.SecondTS, Channels: keys, ControlSubject: control.Subject{Key: "bench_reader"}})
	for _, frame := range frames {
		ok := w.Write(frame)
		if !ok {
			b.Error(w.Error())
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

func bench_stream(b *testing.B, frames []cesium.Frame, channels []cesium.Channel, keys []cesium.ChannelKey, fs xfs.FS) {
	for i := 0; i < b.N; i++ {
		var (
			db  *cesium.DB
			err error
		)
		db, err = cesium.Open("benchmark_stream_test", cesium.WithFS(fs))
		err = db.CreateChannel(ctx, channels...)
		if err != nil {
			b.Errorf("Error during channel creation: %s", err)
		}

		s, err := db.NewStreamer(ctx, cesium.StreamerConfig{Channels: keys})
		sCtx, cancel := signal.WithCancel(ctx)

		i, o := confluence.Attach(s, 1)

		s.Flow(sCtx)

		w, err := db.OpenWriter(ctx, cesium.WriterConfig{Start: 1 * telem.SecondTS, Channels: keys, ControlSubject: control.Subject{Key: "bench_reader"}})

		b.StartTimer()
		for _, frame := range frames {
			ok := w.Write(frame)
			if !ok {
				b.Error(w.Error())
			}
			<-o.Outlet()
		}

		b.StopTimer()

		if err = w.Close(); err != nil {
			b.Error(err)
		}
		i.Close()

		err = db.Close()
		if err != nil {
			b.Errorf("Error during db close: %s", err)
		}

		err = fs.Remove("benchmark_stream_test")
		if err != nil {
			b.Errorf("Error during removing directory: %s", err)
		}

		cancel()
	}
}
