package calculated_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/service/framer/calculated"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
)

func TestCalculatedService(t *testing.T) {
	ctx := context.Background()
	dist := mock.NewBuilder().New(ctx)

	// Create base channel
	baseCH := channel.Channel{Name: "base", DataType: telem.Int64T, Virtual: true}
	assert.NoError(t, dist.Channel.Create(ctx, &baseCH))

	// Create calculated channel
	calcCH := channel.Channel{
		Name:       "calculated",
		DataType:   telem.Int64T,
		Virtual:    true,
		Requires:   []channel.Key{baseCH.Key()},
		Expression: "result = base * 2",
	}
	assert.NoError(t, dist.Channel.Create(ctx, &calcCH))

	// Open the calculated service
	service := calculated.MustSucceed(calculated.Open(calculated.Config{
		Framer:  dist.Framer,
		Channel: dist.Channel,
	}))
	defer service.Close()

	// Simulate writing data to the base channel
	go func() {
		w, err := dist.Framer.NewStreamWriter(ctx, framer.WriterConfig{
			Start: telem.Now(),
			Keys:  []channel.Key{baseCH.Key()},
		})
		if err != nil {
			t.Fatal(err)
		}
		wInlet, _ := confluence.Attach[framer.WriterRequest, framer.WriterResponse](w, 1, 1)
		w.Flow(signal.NewContext(ctx))

		// Write base values
		wInlet.Inlet() <- framer.WriterRequest{
			Command: writer.Data,
			Frame: framer.Frame{
				Keys:   channel.Keys{baseCH.Key()},
				Series: []telem.Series{telem.NewSeriesV[int64](1, 2)},
			},
		}
	}()

	// Allow some time for processing
	time.Sleep(100 * time.Millisecond)

	// Read from the calculated channel
	streamer, err := dist.Framer.NewStreamer(ctx, framer.StreamerConfig{
		Keys: []channel.Key{calcCH.Key()},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, sOutlet := confluence.Attach[framer.StreamerRequest, framer.StreamerResponse](streamer, 1, 1)
	streamer.Flow(signal.NewContext(ctx))

	// Wait for the calculated result
	var res framer.StreamerResponse
	select {
	case <-time.After(5 * time.Second):
		t.Fatal("Timed out waiting for calculated result")
	case res = <-sOutlet.Outlet():
	}

	// Validate the result
	assert.Equal(t, channel.Keys{calcCH.Key()}, res.Frame.Keys)
	assert.Equal(t, int64(2), telem.ValueAt[int64](res.Frame.Series[0], 0)) // Expecting 2 (1 * 2)
	assert.Equal(t, int64(4), telem.ValueAt[int64](res.Frame.Series[0], 1)) // Expecting 4 (2 * 2)
}

func TestCalculatedServiceErrorHandling(t *testing.T) {
	ctx := context.Background()
	dist := mock.NewBuilder().New(ctx)

	// Create base channel
	baseCH := channel.Channel{Name: "base", DataType: telem.Int64T, Virtual: true}
	assert.NoError(t, dist.Channel.Create(ctx, &baseCH))

	// Create calculated channel with an invalid expression
	calcCH := channel.Channel{
		Name:       "calculated",
		DataType:   telem.Int64T,
		Virtual:    true,
		Requires:   []channel.Key{baseCH.Key()},
		Expression: "result = base + fake", // Invalid expression
	}
	assert.NoError(t, dist.Channel.Create(ctx, &calcCH))

	// Open the calculated service
	service := calculated.MustSucceed(calculated.Open(calculated.Config{
		Framer:  dist.Framer,
		Channel: dist.Channel,
	}))
	defer service.Close()

	// Simulate writing data to the base channel
	go func() {
		w, err := dist.Framer.NewStreamWriter(ctx, framer.WriterConfig{
			Start: telem.Now(),
			Keys:  []channel.Key{baseCH.Key()},
		})
		if err != nil {
			t.Fatal(err)
		}
		wInlet, _ := confluence.Attach[framer.WriterRequest, framer.WriterResponse](w, 1, 1)
		w.Flow(signal.NewContext(ctx))

		// Write base values
		wInlet.Inlet() <- framer.WriterRequest{
			Command: writer.Data,
			Frame: framer.Frame{
				Keys:   channel.Keys{baseCH.Key()},
				Series: []telem.Series{telem.NewSeriesV[int64](1, 2)},
			},
		}
	}()

	// Allow some time for processing
	time.Sleep(100 * time.Millisecond)

	// Read from the calculated channel
	streamer, err := dist.Framer.NewStreamer(ctx, framer.StreamerConfig{
		Keys: []channel.Key{calcCH.Key()},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, sOutlet := confluence.Attach[framer.StreamerRequest, framer.StreamerResponse](streamer, 1, 1)
	streamer.Flow(signal.NewContext(ctx))

	// Wait for the calculated result
	select {
	case <-time.After(5 * time.Second):
		t.Fatal("Timed out waiting for calculated result")
	case res := <-sOutlet.Outlet():
		// Expect an error due to invalid expression
		assert.Error(t, res.Error)
	}
}
