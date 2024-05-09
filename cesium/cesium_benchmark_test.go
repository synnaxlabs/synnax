package cesium_test

import (
	"github.com/samber/lo"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/x/control"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/testutil"
	"testing"
)

type CesiumBenchmarkConfig struct {
	domainsPerChannel int
	samplesPerDomain  int
	numIndexChannels  int
	numDataChannels   int
	numRateChannels   int
	numGoRoutines     int
	usingMemFS        bool
}

type CesiumWriteBenchmarkConfig struct {
	CesiumBenchmarkConfig
	numWriters int
}

var cfg CesiumWriteBenchmarkConfig

func BenchmarkCesium(b *testing.B) {
	cfg = CesiumWriteBenchmarkConfig{
		CesiumBenchmarkConfig: CesiumBenchmarkConfig{
			domainsPerChannel: 100,
			samplesPerDomain:  1000,
			numIndexChannels:  10,
			numDataChannels:   400,
			numRateChannels:   90,
			numGoRoutines:     0,
			usingMemFS:        false,
		},
		numWriters: 1,
	}

	b.Run("write", benchmarkCesiumWrite)
}

func benchmarkCesiumWrite(b *testing.B) {
	var (
		db       *cesium.DB
		frames   []cesium.Frame
		channels []cesium.Channel
		fs       xfs.FS
		w        *cesium.Writer
		keys     []cesium.ChannelKey
	)
	fileSystems, cleanUp, err = testutil.FileSystems()
	if err != nil {
		b.Fatalf("Error during setup: %s", err)
	}

	fsMaker, ok := fileSystems[lo.Ternary(cfg.usingMemFS, "memFS", "osFS")]
	fs = fsMaker()
	if !ok {
		b.Fatal("Cannot find osFS in file systems")
	}

	frames, channels, keys = testutil.GenerateFrameAndChannels(cfg.numIndexChannels, cfg.numDataChannels, cfg.numRateChannels, cfg.domainsPerChannel, cfg.samplesPerDomain)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		db, err = cesium.Open("benchmark_test", cesium.WithFS(fs))
		if err != nil {
			b.Fatalf("Error during DB creation: %s", err)
		}

		err = db.CreateChannel(ctx, channels...)
		if err != nil {
			b.Fatalf("Error during channel creation: %s", err)
		}

		//// we have cfg.numWriters writers and cfg.domainsPerChannel frames.
		//// each writer will write cfg.domainsPerChannel / cfg.numWriters frames.
		//// One writer will write everything remaining.
		//framesPerWriter := cfg.domainsPerChannel / cfg.numWriters

		b.StartTimer()
		for j := 0; j < cfg.numWriters; j++ {
			//var framesToWrite []cesium.Frame
			//if j == cfg.numWriters-1 {
			//	framesToWrite = frames[j*framesPerWriter:]
			//} else {
			//	framesToWrite = frames[j*framesPerWriter : (j+1)*framesPerWriter]
			//}

			w = testutil.MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
				ControlSubject: control.Subject{Key: "bench writer"},
				Start:          1 * telem.SecondTS,
				Channels:       keys,
			}))

			for _, frame := range frames {
				ok = w.Write(frame)
				if !ok {
					b.Log(w.Error())
					break
				}
			}
			_, ok = w.Commit()
			if !ok {
				b.Log("Commit failed")
				b.Log(w.Error())
			}
		}

		b.StopTimer()

		err = db.Close()
		if err != nil {
			b.Fatalf("Error during db close: %s", err)
		}

		err = fs.Remove("benchmark_test")
		if err != nil {
			b.Fatalf("Error during removing directory: %s", err)
		}
	}

	err = cleanUp()
	if err != nil {
		b.Fatalf("Error during cleanup: %s", err)
	}
}
