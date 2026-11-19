// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package streamer_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	calcgraph "github.com/synnaxlabs/synnax/pkg/service/calculation/graph"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/framer/calculation"
	"github.com/synnaxlabs/synnax/pkg/service/framer/streamer"
	"github.com/synnaxlabs/synnax/pkg/service/framer/writer"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
)

type benchStreamerEnv struct {
	ctx         context.Context
	node        testutil.Node
	closer      io.MultiCloser
	channelSvc  *channel.Service
	streamerSvc *streamer.Service
}

// mustOpen fails the benchmark with a message naming what when err is not nil.
func mustOpen[T any](b *testing.B, what string, v T, err error) T {
	b.Helper()
	if err != nil {
		b.Fatalf("failed to open %s: %v", what, err)
	}
	return v
}

func newBenchStreamerEnv(b *testing.B) *benchStreamerEnv {
	b.Helper()
	RegisterTestingT(b)
	ctx := b.Context()
	node := testutil.OpenNode(ctx)
	groupSvc, err := group.OpenService(ctx, group.ServiceConfig{DB: node.DB})
	groupSvc = mustOpen(b, "group service", groupSvc, err)
	labelSvc, err := label.OpenService(ctx, label.ServiceConfig{DB: node.DB})
	labelSvc = mustOpen(b, "label service", labelSvc, err)
	statusSvc, err := status.OpenService(ctx, status.ServiceConfig{DB: node.DB})
	statusSvc = mustOpen(b, "status service", statusSvc, err)
	channelSvc, err := channel.OpenService(ctx, channel.ServiceConfig{
		Channel: node.Channel,
		TS:      node.TS,
		DB:      node.DB,
		HostKey: node.Identity.HostKey,
		Status:  statusSvc,
	})
	channelSvc = mustOpen(b, "channel service", channelSvc, err)
	writerSvc, err := writer.NewService(writer.ServiceConfig{
		DB:      node.DB,
		Framer:  node.Framer,
		Channel: channelSvc,
	})
	writerSvc = mustOpen(b, "writer service", writerSvc, err)
	channelGraph, err := calcgraph.Open(ctx, calcgraph.Config{
		DB:      node.DB,
		Channel: channelSvc,
		Status:  statusSvc,
	})
	channelGraph = mustOpen(b, "channel calculation graph", channelGraph, err)
	calc, err := calculation.OpenService(ctx, calculation.ServiceConfig{
		DB:           node.DB,
		Framer:       node.Framer,
		Writer:       writerSvc,
		Channel:      channelSvc,
		ChannelGraph: channelGraph,
	})
	calc = mustOpen(b, "calculation service", calc, err)
	streamerSvc, err := streamer.NewService(streamer.ServiceConfig{
		Framer:      node.Framer,
		Channel:     channelSvc,
		Calculation: calc,
	})
	streamerSvc = mustOpen(b, "streamer service", streamerSvc, err)
	return &benchStreamerEnv{
		ctx:  ctx,
		node: node,
		closer: io.MultiCloser{
			calc, channelSvc, statusSvc, labelSvc, groupSvc, node,
		},
		channelSvc:  channelSvc,
		streamerSvc: streamerSvc,
	}
}

func (e *benchStreamerEnv) close(b *testing.B) {
	b.Helper()
	if err := e.closer.Close(); err != nil {
		b.Errorf("failed to close env: %v", err)
	}
}

func (e *benchStreamerEnv) createVirtualChannel(
	b *testing.B,
	name string,
) *channel.Channel {
	b.Helper()
	ch := &channel.Channel{
		Name:     name,
		DataType: telem.Float32T,
		Virtual:  true,
	}
	if err := e.node.DB.WithTx(e.ctx, func(tx gorp.Tx) error {
		return e.channelSvc.NewWriter(tx).Create(e.ctx, ch)
	}); err != nil {
		b.Fatalf("failed to create channel: %v", err)
	}
	return ch
}

func (e *benchStreamerEnv) createIndexedChannels(
	b *testing.B,
	prefix string,
	numDataChannels int,
) (*channel.Channel, []*channel.Channel) {
	b.Helper()
	indexCh := &channel.Channel{
		Name:     prefix + "_time",
		DataType: telem.TimestampT,
		IsIndex:  true,
	}
	if err := e.node.DB.WithTx(e.ctx, func(tx gorp.Tx) error {
		return e.channelSvc.NewWriter(tx).Create(e.ctx, indexCh)
	}); err != nil {
		b.Fatalf("failed to create index channel: %v", err)
	}
	dataChannels := make([]*channel.Channel, numDataChannels)
	for i := range numDataChannels {
		dataChannels[i] = &channel.Channel{
			Name:       fmt.Sprintf("%s_sensor_%d", prefix, i),
			DataType:   telem.Float32T,
			LocalIndex: indexCh.LocalKey,
		}
		if err := e.node.DB.WithTx(e.ctx, func(tx gorp.Tx) error {
			return e.channelSvc.NewWriter(tx).Create(e.ctx, dataChannels[i])
		}); err != nil {
			b.Fatalf("failed to create data channel: %v", err)
		}
	}
	return indexCh, dataChannels
}

func (e *benchStreamerEnv) createCalculation(
	b *testing.B,
	name, expression string,
) *channel.Channel {
	b.Helper()
	calc := &channel.Channel{
		Name:       name,
		DataType:   telem.Float32T,
		Expression: expression,
	}
	if err := e.node.DB.WithTx(e.ctx, func(tx gorp.Tx) error {
		return e.channelSvc.NewWriter(tx).Create(e.ctx, calc)
	}); err != nil {
		b.Fatalf("failed to create calculation channel: %v", err)
	}
	return calc
}

// benchPipeline is a writer on writerKeys and a running streamer that has sent its
// open ack.
type benchPipeline struct {
	w      *framer.Writer
	inlet  confluence.Inlet[streamer.Request]
	outlet confluence.Outlet[streamer.Response]
	cancel context.CancelFunc
}

// openPipeline opens a writer on writerKeys and a streamer with cfg, waits for the
// open ack, then waits for settle.
func (e *benchStreamerEnv) openPipeline(
	b *testing.B,
	writerKeys []channel.Key,
	cfg streamer.Config,
	settle time.Duration,
) *benchPipeline {
	b.Helper()
	w, err := e.node.Framer.OpenWriter(e.ctx, framer.WriterConfig{
		Start: telem.SecondTS,
		Keys:  writerKeys,
	})
	if err != nil {
		b.Fatalf("failed to open writer: %v", err)
	}
	p := &benchPipeline{w: w}
	cfg.SendOpenAck = true
	s, err := e.streamerSvc.New(e.ctx, cfg)
	if err != nil {
		p.closeWriter(b)
		b.Fatalf("failed to create streamer: %v", err)
	}
	var sCtx signal.Context
	sCtx, p.cancel = signal.Isolated()
	p.inlet, p.outlet = confluence.Attach(s)
	s.Flow(sCtx, confluence.CloseOutputInletsOnExit())
	<-p.outlet.Outlet()
	time.Sleep(settle)
	return p
}

func (p *benchPipeline) closeWriter(b *testing.B) {
	b.Helper()
	if err := p.w.Close(); err != nil {
		b.Errorf("failed to close writer: %v", err)
	}
}

func (p *benchPipeline) close(b *testing.B) {
	b.Helper()
	p.cancel()
	p.closeWriter(b)
}

// writeAndAwait writes fr and waits for one streamer response, failing the benchmark
// on a timeout.
func (p *benchPipeline) writeAndAwait(b *testing.B, fr frame.Frame, iteration int) {
	b.Helper()
	if _, err := p.w.Write(fr); err != nil {
		b.Fatalf("failed to write: %v", err)
	}
	select {
	case <-p.outlet.Outlet():
	case <-time.After(500 * time.Millisecond):
		b.Fatalf("timeout waiting for streamer response on iteration %d", iteration)
	}
}

func BenchmarkStreamerCalc_Throughput(b *testing.B) {
	env := newBenchStreamerEnv(b)
	defer env.close(b)

	ch := env.createVirtualChannel(b, "throughput")
	keys := []channel.Key{ch.Key()}

	w, err := env.node.Framer.OpenWriter(env.ctx, framer.WriterConfig{
		Start: telem.SecondTS,
		Keys:  keys,
	})
	if err != nil {
		b.Fatalf("failed to open writer: %v", err)
	}
	defer func() {
		if err := w.Close(); err != nil {
			b.Errorf("failed to close writer: %v", err)
		}
	}()

	s, err := env.streamerSvc.New(env.ctx, streamer.Config{
		Keys:        keys,
		SendOpenAck: true,
	})
	if err != nil {
		b.Fatalf("failed to create streamer: %v", err)
	}

	sCtx, cancel := signal.Isolated()
	defer cancel()
	inlet, outlet := confluence.Attach(s)
	s.Flow(sCtx, confluence.CloseOutputInletsOnExit())
	<-outlet.Outlet()
	time.Sleep(5 * time.Millisecond)

	data := make([]float32, 100)
	for j := range 100 {
		data[j] = float32(j)
	}
	fr := frame.NewUnary(ch.Key(), telem.NewSeriesV(data...))

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if _, err := w.Write(fr); err != nil {
			b.Fatalf("failed to write: %v", err)
		}
		<-outlet.Outlet()
	}
	b.StopTimer()
	inlet.Close()
	b.ReportMetric(float64(100*b.N)/b.Elapsed().Seconds(), "samples/sec")
}

func BenchmarkStreamerCalc_WithDownsample(b *testing.B) {
	for _, factor := range []uint32{2, 10} {
		b.Run(fmt.Sprintf("factor=%d", factor), func(b *testing.B) {
			env := newBenchStreamerEnv(b)
			defer env.close(b)

			ch := env.createVirtualChannel(b, fmt.Sprintf("ds%d", factor))
			keys := []channel.Key{ch.Key()}

			w, err := env.node.Framer.OpenWriter(env.ctx, framer.WriterConfig{
				Start: telem.SecondTS,
				Keys:  keys,
			})
			if err != nil {
				b.Fatalf("failed to open writer: %v", err)
			}
			defer func() {
				if err := w.Close(); err != nil {
					b.Errorf("failed to close writer: %v", err)
				}
			}()

			s, err := env.streamerSvc.New(env.ctx, streamer.Config{
				Keys:             keys,
				SendOpenAck:      true,
				DownsampleFactor: factor,
			})
			if err != nil {
				b.Fatalf("failed to create streamer: %v", err)
			}

			sCtx, cancel := signal.Isolated()
			defer cancel()
			inlet, outlet := confluence.Attach(s)
			s.Flow(sCtx, confluence.CloseOutputInletsOnExit())
			<-outlet.Outlet()
			time.Sleep(5 * time.Millisecond)

			data := make([]float32, 1000)
			for j := range 1000 {
				data[j] = float32(j)
			}
			fr := frame.NewUnary(ch.Key(), telem.NewSeriesV(data...))

			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				if _, err := w.Write(fr); err != nil {
					b.Fatalf("failed to write: %v", err)
				}
				<-outlet.Outlet()
			}
			b.StopTimer()
			inlet.Close()
			b.ReportMetric(float64(1000*b.N)/b.Elapsed().Seconds(), "samples/sec")
		})
	}
}

func BenchmarkStreamerCalc_WithCalculation(b *testing.B) {
	env := newBenchStreamerEnv(b)
	defer env.close(b)

	indexCh, dataChannels := env.createIndexedChannels(b, "calc", 2)
	calc := env.createCalculation(b, "calc_sum", "return calc_sensor_0 + calc_sensor_1")
	keys := []channel.Key{indexCh.Key(), dataChannels[0].Key(), dataChannels[1].Key()}

	p := env.openPipeline(
		b,
		keys,
		streamer.Config{Keys: []channel.Key{calc.Key()}},
		100*time.Millisecond,
	)
	defer p.close(b)

	const samplesPerFrame = 100
	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		baseTS := telem.TimeStamp(i*samplesPerFrame+1) * telem.SecondTS
		timestamps := make([]telem.TimeStamp, samplesPerFrame)
		data1 := make([]float32, samplesPerFrame)
		data2 := make([]float32, samplesPerFrame)
		for j := range samplesPerFrame {
			timestamps[j] = baseTS + telem.TimeStamp(j)*telem.SecondTS
			data1[j] = float32(j)
			data2[j] = float32(j * 2)
		}
		p.writeAndAwait(b, frame.NewMulti(keys, []telem.Series{
			telem.NewSeriesV(timestamps...),
			telem.NewSeriesV(data1...),
			telem.NewSeriesV(data2...),
		}), i)
	}
	b.StopTimer()
	p.inlet.Close()
	b.ReportMetric(float64(samplesPerFrame*b.N)/b.Elapsed().Seconds(), "samples/sec")
}

func BenchmarkStreamerCalc_FrameSize(b *testing.B) {
	for _, size := range []int{100, 1000, 10000} {
		b.Run(fmt.Sprintf("samples=%d", size), func(b *testing.B) {
			env := newBenchStreamerEnv(b)
			defer env.close(b)

			ch := env.createVirtualChannel(b, fmt.Sprintf("size%d", size))
			keys := []channel.Key{ch.Key()}

			w, err := env.node.Framer.OpenWriter(env.ctx, framer.WriterConfig{
				Start: telem.SecondTS,
				Keys:  keys,
			})
			if err != nil {
				b.Fatalf("failed to open writer: %v", err)
			}
			defer func() {
				if err := w.Close(); err != nil {
					b.Errorf("failed to close writer: %v", err)
				}
			}()

			s, err := env.streamerSvc.New(env.ctx, streamer.Config{
				Keys:        keys,
				SendOpenAck: true,
			})
			if err != nil {
				b.Fatalf("failed to create streamer: %v", err)
			}

			sCtx, cancel := signal.Isolated()
			defer cancel()
			inlet, outlet := confluence.Attach(s)
			s.Flow(sCtx, confluence.CloseOutputInletsOnExit())
			<-outlet.Outlet()
			time.Sleep(5 * time.Millisecond)

			data := make([]float32, size)
			for j := range size {
				data[j] = float32(j)
			}
			fr := frame.NewUnary(ch.Key(), telem.NewSeriesV(data...))

			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				if _, err := w.Write(fr); err != nil {
					b.Fatalf("failed to write: %v", err)
				}
				<-outlet.Outlet()
			}
			b.StopTimer()
			inlet.Close()
			b.ReportMetric(float64(size*b.N)/b.Elapsed().Seconds(), "samples/sec")
		})
	}
}

func BenchmarkStreamerCalc_CalculationChain(b *testing.B) {
	for _, length := range []int{1, 3} {
		b.Run(fmt.Sprintf("chain=%d", length), func(b *testing.B) {
			env := newBenchStreamerEnv(b)
			defer env.close(b)

			indexCh, dataChannels := env.createIndexedChannels(
				b,
				fmt.Sprintf("chain%d", length),
				1,
			)

			var finalCalc *channel.Channel
			prevName := fmt.Sprintf("chain%d_sensor_0", length)
			for i := range length {
				name := fmt.Sprintf("chain%d_calc_%d", length, i)
				finalCalc = env.createCalculation(
					b,
					name,
					fmt.Sprintf("return %s + 1", prevName),
				)
				prevName = name
			}

			keys := []channel.Key{indexCh.Key(), dataChannels[0].Key()}

			p := env.openPipeline(
				b,
				keys,
				streamer.Config{Keys: []channel.Key{finalCalc.Key()}},
				100*time.Millisecond,
			)
			defer p.close(b)

			const samplesPerFrame = 100
			b.ReportAllocs()
			b.ResetTimer()
			for i := range b.N {
				baseTS := telem.TimeStamp(i*samplesPerFrame+1) * telem.SecondTS
				timestamps := make([]telem.TimeStamp, samplesPerFrame)
				data := make([]float32, samplesPerFrame)
				for j := range samplesPerFrame {
					timestamps[j] = baseTS + telem.TimeStamp(j)*telem.SecondTS
					data[j] = float32(j)
				}
				p.writeAndAwait(b, frame.NewMulti(keys, []telem.Series{
					telem.NewSeriesV(timestamps...),
					telem.NewSeriesV(data...),
				}), i)
			}
			b.StopTimer()
			p.inlet.Close()
			b.ReportMetric(
				float64(samplesPerFrame*b.N)/b.Elapsed().Seconds(),
				"samples/sec",
			)
		})
	}
}
