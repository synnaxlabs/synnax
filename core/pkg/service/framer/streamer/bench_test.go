// Copyright 2025 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/framer/calculation"
	"github.com/synnaxlabs/synnax/pkg/service/framer/streamer"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	svcstatus "github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
)

type benchStreamerEnv struct {
	ctx         context.Context
	builder     *mock.Cluster
	dist        mock.Node
	streamerSvc *streamer.Service
}

func newBenchStreamerEnv(b *testing.B) *benchStreamerEnv {
	RegisterTestingT(b)
	ctx := context.Background()
	builder := mock.NewCluster()
	dist := builder.Provision(ctx)

	labelSvc, err := label.OpenService(ctx, label.Config{
		DB:       dist.DB,
		Ontology: dist.Ontology,
		Group:    dist.Group,
		Signals:  dist.Signals,
	})
	if err != nil {
		b.Fatalf("failed to open label service: %v", err)
	}

	statusSvc, err := svcstatus.OpenService(ctx, svcstatus.ServiceConfig{
		DB:       dist.DB,
		Label:    labelSvc,
		Ontology: dist.Ontology,
		Group:    dist.Group,
		Signals:  dist.Signals,
	})
	if err != nil {
		b.Fatalf("failed to open status service: %v", err)
	}

	arcSvc, err := arc.OpenService(ctx, arc.ServiceConfig{
		Channel:  dist.Channel,
		Ontology: dist.Ontology,
		DB:       dist.DB,
		Framer:   dist.Framer,
		Status:   statusSvc,
		Signals:  dist.Signals,
	})
	if err != nil {
		b.Fatalf("failed to open arc service: %v", err)
	}

	calc, err := calculation.OpenService(ctx, calculation.ServiceConfig{
		DB:                dist.DB,
		Arc:               arcSvc,
		Framer:            dist.Framer,
		Channels:          dist.Channel,
		ChannelObservable: dist.Channel.NewObservable(),
	})
	if err != nil {
		b.Fatalf("failed to open calculation service: %v", err)
	}

	streamerSvc, err := streamer.NewService(streamer.ServiceConfig{
		DistFramer:  dist.Framer,
		Channel:     dist.Channel,
		Calculation: calc,
	})
	if err != nil {
		b.Fatalf("failed to open streamer service: %v", err)
	}

	return &benchStreamerEnv{
		ctx:         ctx,
		builder:     builder,
		dist:        dist,
		streamerSvc: streamerSvc,
	}
}

func (e *benchStreamerEnv) close(b *testing.B) {
	if err := e.builder.Close(); err != nil {
		b.Errorf("failed to close cluster: %v", err)
	}
}

func (e *benchStreamerEnv) createVirtualChannel(b *testing.B, name string) *channel.Channel {
	ch := &channel.Channel{
		Name:     name,
		DataType: telem.Float32T,
		Virtual:  true,
	}
	if err := e.dist.Channel.Create(e.ctx, ch); err != nil {
		b.Fatalf("failed to create channel: %v", err)
	}
	return ch
}

func (e *benchStreamerEnv) createIndexedChannels(
	b *testing.B,
	prefix string,
	numDataChannels int,
) (*channel.Channel, []*channel.Channel) {
	indexCh := &channel.Channel{
		Name:     prefix + "_time",
		DataType: telem.TimeStampT,
		IsIndex:  true,
	}
	if err := e.dist.Channel.Create(e.ctx, indexCh); err != nil {
		b.Fatalf("failed to create index channel: %v", err)
	}
	dataChannels := make([]*channel.Channel, numDataChannels)
	for i := 0; i < numDataChannels; i++ {
		dataChannels[i] = &channel.Channel{
			Name:       fmt.Sprintf("%s_sensor_%d", prefix, i),
			DataType:   telem.Float32T,
			LocalIndex: indexCh.LocalKey,
		}
		if err := e.dist.Channel.Create(e.ctx, dataChannels[i]); err != nil {
			b.Fatalf("failed to create data channel: %v", err)
		}
	}
	return indexCh, dataChannels
}

func (e *benchStreamerEnv) createCalculation(b *testing.B, name, expression string) *channel.Channel {
	calc := &channel.Channel{
		Name:       name,
		DataType:   telem.Float32T,
		Expression: expression,
	}
	if err := e.dist.Channel.Create(e.ctx, calc); err != nil {
		b.Fatalf("failed to create calculation channel: %v", err)
	}
	return calc
}

func BenchmarkStreamerCalc_Throughput(b *testing.B) {
	env := newBenchStreamerEnv(b)
	defer env.close(b)

	ch := env.createVirtualChannel(b, "throughput")
	keys := []channel.Key{ch.Key()}

	w, err := env.dist.Framer.OpenWriter(env.ctx, framer.WriterConfig{
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
	for j := 0; j < 100; j++ {
		data[j] = float32(j)
	}
	fr := core.UnaryFrame(ch.Key(), telem.NewSeriesV(data...))

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
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
	for _, factor := range []int{2, 10} {
		b.Run(fmt.Sprintf("factor=%d", factor), func(b *testing.B) {
			env := newBenchStreamerEnv(b)
			defer env.close(b)

			ch := env.createVirtualChannel(b, fmt.Sprintf("ds%d", factor))
			keys := []channel.Key{ch.Key()}

			w, err := env.dist.Framer.OpenWriter(env.ctx, framer.WriterConfig{
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
			for j := 0; j < 1000; j++ {
				data[j] = float32(j)
			}
			fr := core.UnaryFrame(ch.Key(), telem.NewSeriesV(data...))

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
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

	w, err := env.dist.Framer.OpenWriter(env.ctx, framer.WriterConfig{
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
		Keys:        []channel.Key{calc.Key()},
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
	time.Sleep(100 * time.Millisecond)

	const samplesPerFrame = 100
	timeout := 500 * time.Millisecond
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		baseTS := telem.TimeStamp(i*samplesPerFrame+1) * telem.SecondTS
		timestamps := make([]telem.TimeStamp, samplesPerFrame)
		data1 := make([]float32, samplesPerFrame)
		data2 := make([]float32, samplesPerFrame)
		for j := 0; j < samplesPerFrame; j++ {
			timestamps[j] = baseTS + telem.TimeStamp(j)*telem.SecondTS
			data1[j] = float32(j)
			data2[j] = float32(j * 2)
		}
		fr := core.MultiFrame(keys, []telem.Series{
			telem.NewSeriesV(timestamps...),
			telem.NewSeriesV(data1...),
			telem.NewSeriesV(data2...),
		})
		if _, err := w.Write(fr); err != nil {
			b.Fatalf("failed to write: %v", err)
		}
		select {
		case <-outlet.Outlet():
		case <-time.After(timeout):
			b.Fatalf("timeout waiting for streamer response on iteration %d", i)
		}
	}
	b.StopTimer()
	inlet.Close()
	b.ReportMetric(float64(samplesPerFrame*b.N)/b.Elapsed().Seconds(), "samples/sec")
}

func BenchmarkStreamerCalc_FrameSize(b *testing.B) {
	for _, size := range []int{100, 1000, 10000} {
		b.Run(fmt.Sprintf("samples=%d", size), func(b *testing.B) {
			env := newBenchStreamerEnv(b)
			defer env.close(b)

			ch := env.createVirtualChannel(b, fmt.Sprintf("size%d", size))
			keys := []channel.Key{ch.Key()}

			w, err := env.dist.Framer.OpenWriter(env.ctx, framer.WriterConfig{
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
			for j := 0; j < size; j++ {
				data[j] = float32(j)
			}
			fr := core.UnaryFrame(ch.Key(), telem.NewSeriesV(data...))

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
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

			indexCh, dataChannels := env.createIndexedChannels(b, fmt.Sprintf("chain%d", length), 1)

			var finalCalc *channel.Channel
			prevName := fmt.Sprintf("chain%d_sensor_0", length)
			for i := 0; i < length; i++ {
				name := fmt.Sprintf("chain%d_calc_%d", length, i)
				finalCalc = env.createCalculation(b, name, fmt.Sprintf("return %s + 1", prevName))
				prevName = name
			}

			keys := []channel.Key{indexCh.Key(), dataChannels[0].Key()}

			w, err := env.dist.Framer.OpenWriter(env.ctx, framer.WriterConfig{
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
				Keys:        []channel.Key{finalCalc.Key()},
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
			time.Sleep(100 * time.Millisecond)

			const samplesPerFrame = 100
			timeout := 500 * time.Millisecond
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				baseTS := telem.TimeStamp(i*samplesPerFrame+1) * telem.SecondTS
				timestamps := make([]telem.TimeStamp, samplesPerFrame)
				data := make([]float32, samplesPerFrame)
				for j := 0; j < samplesPerFrame; j++ {
					timestamps[j] = baseTS + telem.TimeStamp(j)*telem.SecondTS
					data[j] = float32(j)
				}
				fr := core.MultiFrame(keys, []telem.Series{
					telem.NewSeriesV(timestamps...),
					telem.NewSeriesV(data...),
				})
				if _, err := w.Write(fr); err != nil {
					b.Fatalf("failed to write: %v", err)
				}
				select {
				case <-outlet.Outlet():
				case <-time.After(timeout):
					b.Fatalf("timeout waiting for streamer response on iteration %d", i)
				}
			}
			b.StopTimer()
			inlet.Close()
			b.ReportMetric(float64(samplesPerFrame*b.N)/b.Elapsed().Seconds(), "samples/sec")
		})
	}
}
