package cesium_test

import (
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/x/testutil"
	"testing"
)

// BenchmarkCesiumWrite runs a benchmark test with 500 channels (10 index, 490 data).
// Each index channel indexes 49 float32 data channels.
// Data is stored on MemFS.
func BenchmarkCesiumWriteMemFS(b *testing.B) {
	const samplesPerChannel = 1e6
	var (
		db       *cesium.DB
		frame    cesium.Frame
		channels []cesium.Channel
		keys     []cesium.ChannelKey
	)
	fileSystems, cleanUp, err = testutil.FileSystems()
	if err != nil {
		b.Fatalf("Error during setup: %s", err)
	}

	fs, ok := fileSystems["memFS"]
	if !ok {
		b.Fatal("Cannot find osFS in file systems")
	}

	db, err = cesium.Open("benchmark_test", cesium.WithFS(fs()))
	if !ok {
		b.Fatalf("Error during DB creation: %s", err)
	}

	frame, channels, keys = testutil.GenerateFrameAndChannels(10, 490, samplesPerChannel)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err = db.CreateChannel(ctx, channels...)
		if err != nil {
			b.Fatalf("Error during channel creation: %s", err)
		}

		b.StartTimer()
		err = db.Write(ctx, 0, frame)
		if err != nil {
			b.StopTimer()
			b.Fatalf("Error during write: %s", err)
		}

		b.StopTimer()
		//Rollback
		err = db.DeleteChannels(keys)
		if err != nil {
			b.Fatalf("Error during deleting db: %s", err)
		}

	}

	err = db.Close()
	if err != nil {
		b.Fatalf("Error during db close: %s", err)
	}

	err = cleanUp()
	if err != nil {
		b.Fatalf("Error during cleanup: %s", err)
	}
}

// BenchmarkCesiumWrite runs a benchmark test with 500 channels (10 index, 490 data).
// Each index channel indexes 49 float32 data channels.
// Data is stored on osFS.
func BenchmarkCesiumWriteOsFS(b *testing.B) {
	const samplesPerChannel = 1e5
	var (
		db       *cesium.DB
		frame    cesium.Frame
		channels []cesium.Channel
		keys     []cesium.ChannelKey
	)
	fileSystems, cleanUp, err = testutil.FileSystems()
	if err != nil {
		b.Fatalf("Error during setup: %s", err)
	}

	fs, ok := fileSystems["osFS"]
	if !ok {
		b.Fatal("Cannot find osFS in file systems")
	}

	db, err = cesium.Open("benchmark_test", cesium.WithFS(fs()))
	if !ok {
		b.Fatalf("Error during DB creation: %s", err)
	}

	frame, channels, keys = testutil.GenerateFrameAndChannels(10, 490, samplesPerChannel)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err = db.CreateChannel(ctx, channels...)
		if err != nil {
			b.Fatalf("Error during channel creation: %s", err)
		}

		b.StartTimer()
		err = db.Write(ctx, 0, frame)
		if err != nil {
			b.StopTimer()
			b.Fatalf("Error during write: %s", err)
		}

		b.StopTimer()
		//Rollback
		err = db.DeleteChannels(keys)
		if err != nil {
			b.Fatalf("Error during deleting db: %s", err)
		}

	}

	err = db.Close()
	if err != nil {
		b.Fatalf("Error during db close: %s", err)
	}

	err = cleanUp()
	if err != nil {
		b.Fatalf("Error during cleanup: %s", err)
	}
}
