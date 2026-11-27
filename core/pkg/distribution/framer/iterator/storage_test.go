// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package iterator_test

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/iterator"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/telem/testutil"
	. "github.com/synnaxlabs/x/testutil"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// sendFailContext cancels once for the first send that finds full true, and cancels
// nothing before or after that send. failed closes when the next send starts.
type sendFailContext struct {
	context.Context
	full     func() bool
	canceled atomic.Bool
	failed   chan struct{}
	report   sync.Once
}

// sendFailSignal runs each routine under a sendFailContext that wraps the routine's
// context.
type sendFailSignal struct {
	signal.Context
	full   func() bool
	failed chan struct{}
}

func (s sendFailSignal) Go(
	f func(ctx context.Context) error,
	opts ...signal.RoutineOption,
) {
	s.Context.Go(func(ctx context.Context) error {
		return f(&sendFailContext{Context: ctx, full: s.full, failed: s.failed})
	}, opts...)
}

func (c *sendFailContext) Done() <-chan struct{} {
	if c.canceled.Load() {
		c.report.Do(func() { close(c.failed) })
		return nil
	}
	if !c.full() {
		return nil
	}
	c.canceled.Store(true)
	done := make(chan struct{})
	close(done)
	return done
}

func (c *sendFailContext) Err() error {
	if c.canceled.Load() {
		return context.Canceled
	}
	return nil
}

var _ = Describe("Storage stream", func() {
	// writeStamps writes the stamps 10 to 19 seconds to the index of keys.
	writeStamps := func(ctx context.Context, dist testutil.Node, keys channel.Keys) {
		GinkgoHelper()
		w := MustSucceed(dist.Framer.OpenWriter(ctx, writer.Config{
			Keys:  keys,
			Start: 10 * telem.SecondTS,
			Sync:  new(true),
		}))
		Expect(w.Write(frame.NewMulti(keys, []telem.Series{
			telem.NewSeriesSecondsTSV(10, 11, 12, 13, 14, 15, 16, 17, 18, 19),
		}))).To(BeTrue())
		MustCommit(w)
		Expect(w.Close()).To(Succeed())
	}
	openStream := func(
		ctx context.Context,
		dist testutil.Node,
		keys channel.Keys,
		chunkSize int64,
	) iterator.StreamIterator {
		GinkgoHelper()
		return MustSucceed(dist.Framer.NewStreamIterator(ctx, iterator.Config{
			Keys:      keys,
			Bounds:    telem.TimeRangeMax,
			ChunkSize: chunkSize,
		}))
	}

	It("Should send each automatic span of the chunk size as one data response, "+
		"then its ack", func(ctx SpecContext) {
		s := DeferClose(openScenario(ctx))
		writeStamps(ctx, s.dist, s.keys)
		stream := openStream(ctx, s.dist, s.keys, 3)
		sCtx, cancel := signal.Isolated()
		defer cancel()
		requests, responses := confluence.Attach(stream, 1)
		stream.Flow(sCtx, confluence.CloseOutputInletsOnExit())
		requests.Inlet() <- iterator.Request{Command: iterator.CommandSeekFirst}
		Eventually(responses.Outlet()).Should(Receive(HaveField("Ack", true)))
		var lens []int64
		for seq := 1; ; seq++ {
			requests.Inlet() <- iterator.Request{
				Command: iterator.CommandNext,
				Span:    iterator.AutoSpan,
			}
			var res iterator.Response
			Eventually(responses.Outlet()).Should(Receive(&res))
			Expect(res.SeqNum).To(Equal(seq))
			if res.Variant == iterator.ResponseVariantAck {
				Expect(res.Ack).To(BeFalse())
				break
			}
			lens = append(lens, res.Frame.SeriesAt(0).Len())
			Eventually(responses.Outlet()).Should(Receive(&res))
			Expect(res.Variant).To(Equal(iterator.ResponseVariantAck))
			Expect(res.Ack).To(BeTrue())
		}
		Expect(lens).To(Equal([]int64{3, 3, 3, 1}))
		requests.Close()
		Expect(sCtx.Wait()).To(Succeed())
	})

	It("Should close its iterator when it flows in a canceled context", func(
		ctx SpecContext,
	) {
		s := openScenario(ctx)
		stream := openStream(ctx, s.dist, s.keys, 0)
		sCtx, cancel := signal.Isolated()
		cancel()
		stream.InFrom(confluence.NewStream[iterator.Request]())
		stream.OutTo(confluence.NewStream[iterator.Response]())
		stream.Flow(sCtx)
		Expect(sCtx.Wait()).To(MatchError(context.Canceled))
		Expect(s.Close()).To(Succeed())
	})

	It("Should exit on cancel when nothing reads its responses", func(
		ctx SpecContext,
	) {
		s := openScenario(ctx)
		writeStamps(ctx, s.dist, s.keys)
		stream := openStream(ctx, s.dist, s.keys, 0)
		var (
			sCtx, cancel = signal.Isolated()
			requests     = confluence.NewStream[iterator.Request]()
		)
		stream.InFrom(requests)
		stream.OutTo(confluence.NewStream[iterator.Response]())
		stream.Flow(sCtx)
		requests.Inlet() <- iterator.Request{Command: iterator.CommandSeekFirst}
		// The ack of the seek has no reader.
		cancel()
		Eventually(sCtx.Stopped()).Should(BeClosed())
		Expect(sCtx.Wait()).To(MatchError(context.Canceled))
		Expect(s.Close()).To(Succeed())
	})

	DescribeTable(
		"Should not acknowledge a read whose data it failed to send",
		func(ctx SpecContext, span telem.TimeSpan) {
			s := openScenario(ctx)
			writeStamps(ctx, s.dist, s.keys)
			stream := openStream(ctx, s.dist, s.keys, 0)
			var (
				requests  = confluence.NewStream[iterator.Request](2)
				responses = confluence.NewStream[iterator.Response](1)
			)
			requests.Inlet() <- iterator.Request{Command: iterator.CommandSeekFirst}
			requests.Inlet() <- iterator.Request{
				Command: iterator.CommandNext,
				Span:    span,
			}
			// The ack of the seek fills responses for the data of the read.
			sCtx, cancel := signal.WithCancel(ctx)
			defer cancel()
			failed := make(chan struct{})
			stream.InFrom(requests)
			stream.OutTo(responses)
			stream.Flow(sendFailSignal{
				Context: sCtx,
				full: func() bool {
					return len(requests.Outlet()) == 0 &&
						len(responses.Outlet()) == cap(responses.Outlet())
				},
				failed: failed,
			}, confluence.CloseOutputInletsOnExit())
			Eventually(failed).Should(BeClosed())
			Eventually(responses.Outlet()).Should(Receive(
				HaveField("Command", iterator.CommandSeekFirst),
			))
			var ack iterator.Response
			Eventually(responses.Outlet()).Should(Receive(&ack))
			Expect(ack.Variant).To(Equal(iterator.ResponseVariantAck))
			Expect(ack.Error).To(MatchError(context.Canceled))
			Expect(ack.Ack).To(BeFalse())
			requests.Close()
			Eventually(sCtx.Stopped()).Should(BeClosed())
			Expect(sCtx.Wait()).To(Succeed())
			Expect(s.Close()).To(Succeed())
		},
		Entry("automatic span", iterator.AutoSpan),
		Entry("fixed span", telem.Second),
	)
})

var _ = Describe("Storage stream read failures", func() {
	var (
		dist      testutil.Node
		logs      *observer.ObservedLogs
		good, bad channel.Key
	)
	BeforeEach(func(ctx SpecContext) {
		dir := GinkgoT().TempDir()
		layer := MustSucceed(storage.OpenLayer(ctx, storage.LayerConfig{
			Dirname: dir,
			Version: "0.0.0",
		}))
		stop := GoRun(GinkgoT().Context(), layer.Run)
		DeferCleanup(
			func() { Expect(errors.Join(stop(), layer.Close())).To(Succeed()) },
		)
		var core zapcore.Core
		core, logs = observer.New(zapcore.ErrorLevel)
		logger := MustSucceed(alamos.NewLogger(alamos.LoggerConfig{
			ZapLogger: zap.New(core),
		}))
		dist = testutil.NewNode(ctx, distribution.LayerConfig{
			Storage:         layer,
			Instrumentation: alamos.New("test", alamos.WithLogger(logger)),
		})
		idx := MustSucceed(dist.CreateChannels(ctx, []channel.Channel{{
			Name:     "time",
			IsIndex:  true,
			DataType: telem.TimestampT,
		}}))[0]
		data := MustSucceed(dist.CreateChannels(ctx, []channel.Channel{
			{Name: "good", LocalIndex: idx.LocalKey, DataType: telem.Int64T},
			{Name: "bad", LocalIndex: idx.LocalKey, DataType: telem.Int64T},
		}))
		good, bad = data[0].Key(), data[1].Key()
		keys := channel.Keys{idx.Key(), good, bad}
		w := MustSucceed(dist.Framer.OpenWriter(ctx, writer.Config{
			Keys:  keys,
			Start: 10 * telem.SecondTS,
			Sync:  new(true),
		}))
		Expect(w.Write(frame.NewMulti(keys, []telem.Series{
			telem.NewSeriesSecondsTSV(10, 11, 12),
			telem.NewSeriesV[int64](1, 2, 3),
			telem.NewSeriesV[int64](4, 5, 6),
		}))).To(BeTrue())
		MustCommit(w)
		Expect(w.Close()).To(Succeed())
		Expect(os.Remove(filepath.Join(dir, "cesium", bad.String(), "1.domain"))).
			To(Succeed())
	})

	DescribeTable("Should acknowledge a read where one channel fails, send the "+
		"other channels whole, and log the failure",
		func(ctx SpecContext, span telem.TimeSpan) {
			stream := MustSucceed(dist.Framer.NewStreamIterator(ctx, iterator.Config{
				Keys:   channel.Keys{good, bad},
				Bounds: telem.TimeRangeMax,
			}))
			sCtx, cancel := signal.Isolated()
			defer cancel()
			requests, responses := confluence.Attach(stream, 1)
			stream.Flow(sCtx, confluence.CloseOutputInletsOnExit())
			requests.Inlet() <- iterator.Request{Command: iterator.CommandSeekFirst}
			Eventually(responses.Outlet()).Should(Receive(HaveField("Ack", true)))
			requests.Inlet() <- iterator.Request{Command: iterator.CommandNext, Span: span}
			var res iterator.Response
			Eventually(responses.Outlet()).Should(Receive(&res))
			Expect(res.Variant).To(Equal(iterator.ResponseVariantData))
			Expect(res.Frame.Get(bad).Series).To(BeEmpty())
			Expect(res.Frame.Get(good).Series).To(HaveLen(1))
			Expect(res.Frame.Get(good).Series[0]).To(MatchSeriesDataV[int64](1, 2, 3))
			Eventually(responses.Outlet()).Should(Receive(&res))
			Expect(res.Variant).To(Equal(iterator.ResponseVariantAck))
			Expect(res.Ack).To(BeTrue())
			Expect(res.Error).ToNot(HaveOccurred())
			failures := logs.FilterMessage("read of a channel failed").All()
			Expect(failures).To(HaveLen(1))
			Expect(
				failures[0].ContextMap(),
			).To(HaveKeyWithValue("channel", bad.String()))
			Expect(failures[0].ContextMap()).To(HaveKeyWithValue(
				"error",
				ContainSubstring("1.domain: no such file or directory"),
			))
			requests.Close()
			Expect(sCtx.Wait()).To(MatchError(ContainSubstring("no such file")))
		},
		Entry("of an automatic span", iterator.AutoSpan),
		Entry("of a fixed span", 10*telem.Second),
	)
})
