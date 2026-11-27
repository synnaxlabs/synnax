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
	"path"
	"sync/atomic"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/iterator"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/x/confluence"
	xfs "github.com/synnaxlabs/x/io/fs"
	. "github.com/synnaxlabs/x/io/fs/testutil"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// readFaultFS fails each ReadAt on the files of the directory that failing names.
type readFaultFS struct {
	xfs.FS
	dir     string
	failing *atomic.Pointer[string]
}

// Sub implements xfs.FS.
func (f readFaultFS) Sub(name string) (xfs.FS, error) {
	sub, err := f.FS.Sub(name)
	if err != nil {
		return nil, err
	}
	return readFaultFS{FS: sub, dir: path.Join(f.dir, name), failing: f.failing}, nil
}

// Open implements xfs.FS.
func (f readFaultFS) Open(name string, flag int) (xfs.File, error) {
	file, err := f.FS.Open(name, flag)
	if err != nil {
		return nil, err
	}
	return readFaultFile{File: file, dir: f.dir, failing: f.failing}, nil
}

type readFaultFile struct {
	xfs.File
	dir     string
	failing *atomic.Pointer[string]
}

// ReadAt implements xfs.File.
func (f readFaultFile) ReadAt(p []byte, off int64) (int, error) {
	if dir := f.failing.Load(); dir != nil && *dir == f.dir {
		return 0, ErrFault
	}
	return f.File.ReadAt(p, off)
}

// commandStream runs requests on one flowing storage stream.
type commandStream struct {
	requests  *confluence.Stream[iterator.Request]
	responses *confluence.Stream[iterator.Response]
	sCtx      signal.Context
	seqNum    int
}

// exec sends req and returns the stamps of its data responses and its ack. It checks
// that each response carries the sequence number of req, that each data response names
// CommandNext, and that the ack names the command of req.
func (s *commandStream) exec(
	req iterator.Request,
) ([]telem.TimeStamp, iterator.Response) {
	GinkgoHelper()
	s.requests.Inlet() <- req
	var stamps []telem.TimeStamp
	for {
		var res iterator.Response
		Eventually(s.responses.Outlet()).Should(Receive(&res))
		Expect(res.SeqNum).To(Equal(s.seqNum))
		if res.Variant == iterator.ResponseVariantAck {
			Expect(res.Command).To(Equal(req.Command))
			s.seqNum++
			return stamps, res
		}
		Expect(res.Variant).To(Equal(iterator.ResponseVariantData))
		Expect(res.Command).To(Equal(iterator.CommandNext))
		for series := range res.Frame.Series() {
			if series.DataType == telem.TimestampT {
				stamps = append(stamps, series.Unmarshal[telem.TimeStamp]()...)
			}
		}
	}
}

// close ends the stream and returns the error that stopped it.
func (s *commandStream) close() error {
	s.requests.Close()
	return s.sCtx.Wait()
}

var _ = Describe("Storage stream check", func() {
	// flow opens a storage stream on keys and flows it.
	flow := func(
		ctx context.Context,
		dist testutil.Node,
		keys channel.Keys,
	) *commandStream {
		GinkgoHelper()
		stream := MustSucceed(dist.Framer.NewStreamIterator(ctx, iterator.Config{
			Keys:   keys,
			Bounds: telem.TimeRangeMax,
		}))
		sCtx, cancel := signal.Isolated()
		DeferCleanup(cancel)
		s := &commandStream{
			requests:  confluence.NewStream[iterator.Request](1),
			responses: confluence.NewStream[iterator.Response](1),
			sCtx:      sCtx,
		}
		stream.InFrom(s.requests)
		stream.OutTo(s.responses)
		stream.Flow(sCtx, confluence.CloseOutputInletsOnExit())
		return s
	}
	// write writes the stamps 10 to 19 seconds to the index, first in keys, and the
	// values 0 to 9 to every other channel of keys.
	write := func(ctx context.Context, dist testutil.Node, keys channel.Keys) {
		GinkgoHelper()
		w := MustSucceed(dist.Framer.OpenWriter(ctx, writer.Config{
			Keys:  keys,
			Start: 10 * telem.SecondTS,
			Sync:  new(true),
		}))
		series := make([]telem.Series, len(keys))
		series[0] = telem.NewSeriesSecondsTSV(10, 11, 12, 13, 14, 15, 16, 17, 18, 19)
		for i := range keys[1:] {
			series[i+1] = telem.NewSeriesV[int64](0, 1, 2, 3, 4, 5, 6, 7, 8, 9)
		}
		Expect(w.Write(frame.NewMulti(keys, series))).To(BeTrue())
		MustCommit(w)
		Expect(w.Close()).To(Succeed())
	}
	seconds := func(from, to int) []telem.TimeStamp {
		var stamps []telem.TimeStamp
		for sec := from; sec <= to; sec++ {
			stamps = append(stamps, telem.TimeStamp(sec)*telem.SecondTS)
		}
		return stamps
	}

	It("Should answer each command with its ack and data", func(ctx SpecContext) {
		s := DeferClose(openScenario(ctx))
		write(ctx, s.dist, s.keys)
		stream := flow(ctx, s.dist, s.keys)
		for _, step := range []struct {
			req  iterator.Request
			ack  bool
			data []telem.TimeStamp
		}{
			{req: iterator.Request{
				Command: iterator.CommandSeekLE,
				Stamp:   5 * telem.SecondTS,
			}},
			{req: iterator.Request{
				Command: iterator.CommandSeekGE,
				Stamp:   25 * telem.SecondTS,
			}},
			{req: iterator.Request{
				Command: iterator.CommandSeekGE,
				Stamp:   5 * telem.SecondTS,
			}, ack: true},
			{req: iterator.Request{
				Command: iterator.CommandSeekLE,
				Stamp:   25 * telem.SecondTS,
			}, ack: true},
			{req: iterator.Request{Command: iterator.CommandValid}},
			{req: iterator.Request{Command: iterator.CommandSeekFirst}, ack: true},
			{
				req: iterator.Request{
					Command: iterator.CommandNext,
					Span:    iterator.AutoSpan,
				},
				ack:  true,
				data: seconds(10, 19),
			},
			{req: iterator.Request{Command: iterator.CommandValid}, ack: true},
			{req: iterator.Request{
				Command: iterator.CommandNext,
				Span:    iterator.AutoSpan,
			}},
			{req: iterator.Request{Command: iterator.CommandSeekLast}, ack: true},
			{
				req: iterator.Request{
					Command: iterator.CommandPrev,
					Span:    5 * telem.Second,
				},
				ack:  true,
				data: seconds(15, 19),
			},
			{
				req: iterator.Request{
					Command: iterator.CommandPrev,
					Span:    iterator.AutoSpan,
				},
				ack:  true,
				data: seconds(10, 14),
			},
			{
				req: iterator.Request{
					Command: iterator.CommandSetBounds,
					Bounds:  (12 * telem.SecondTS).Range(15 * telem.SecondTS),
				},
				ack: true,
			},
			{req: iterator.Request{Command: iterator.CommandSeekFirst}, ack: true},
			{
				req: iterator.Request{
					Command: iterator.CommandNext,
					Span:    10 * telem.Second,
				},
				ack:  true,
				data: seconds(12, 14),
			},
			{req: iterator.Request{Command: iterator.CommandError}},
		} {
			data, ack := stream.exec(step.req)
			Expect(ack.Error).ToNot(HaveOccurred())
			Expect(ack.Ack).To(Equal(step.ack), "ack of %v", step.req.Command)
			Expect(data).To(Equal(step.data), "data of %v", step.req.Command)
		}
		Expect(stream.close()).To(Succeed())
	})

	DescribeTable("Should reject a command outside the command set",
		func(ctx SpecContext, command iterator.Command, message string) {
			s := DeferClose(openScenario(ctx))
			write(ctx, s.dist, s.keys)
			stream := flow(ctx, s.dist, s.keys)
			data, ack := stream.exec(iterator.Request{Command: command})
			Expect(data).To(BeEmpty())
			Expect(ack.Ack).To(BeFalse())
			Expect(ack.Error).To(MatchError(validate.ErrValidation))
			Expect(ack.Error).To(MatchError(message))
			_, ack = stream.exec(iterator.Request{Command: iterator.CommandSeekFirst})
			Expect(ack.Ack).To(BeTrue())
			Expect(stream.close()).To(Succeed())
		},
		Entry(
			"zero",
			iterator.Command(0),
			"invalid iterator command 0: validation error",
		),
		Entry(
			"after SetBounds",
			iterator.CommandSetBounds+1,
			"invalid iterator command 10: validation error",
		),
	)

	DescribeTable("Should read the other channels when the read of one channel fails",
		func(ctx SpecContext, span telem.TimeSpan) {
			failing := &atomic.Pointer[string]{}
			kv := DeferClose(memkv.New())
			store := &storage.Layer{
				KV:   kv,
				Gorp: gorp.Wrap(kv),
				TS: DeferClose(MustSucceed(cesium.Open(ctx, "", cesium.WithFS(
					readFaultFS{FS: xfs.NewMem(), failing: failing},
				)))),
			}
			StartRun(store.Run)
			core, logs := observer.New(zapcore.ErrorLevel)
			logger := MustSucceed(alamos.NewLogger(alamos.LoggerConfig{
				ZapLogger: zap.New(core),
			}))
			dist := testutil.NewNode(ctx, distribution.LayerConfig{
				Storage:         store,
				Instrumentation: alamos.New("check", alamos.WithLogger(logger)),
			})
			idx := MustSucceed(dist.CreateChannels(ctx, []channel.Channel{{
				Name:     "check_time",
				IsIndex:  true,
				DataType: telem.TimestampT,
			}}))[0]
			data := MustSucceed(dist.CreateChannels(ctx, []channel.Channel{{
				Name:       "check_data",
				LocalIndex: idx.LocalKey,
				DataType:   telem.Int64T,
			}}))[0]
			keys := channel.Keys{idx.Key(), data.Key()}
			write(ctx, dist, keys)
			stream := flow(ctx, dist, keys)
			_, ack := stream.exec(iterator.Request{Command: iterator.CommandSeekFirst})
			Expect(ack.Ack).To(BeTrue())
			dir := data.Key().String()
			failing.Store(&dir)
			read, ack := stream.exec(iterator.Request{
				Command: iterator.CommandNext,
				Span:    span,
			})
			Expect(ack.Error).ToNot(HaveOccurred())
			Expect(ack.Ack).To(BeTrue())
			Expect(read).To(Equal(seconds(10, 19)))
			// The failed channel reports its failure once, until a seek.
			read, ack = stream.exec(iterator.Request{
				Command: iterator.CommandNext,
				Span:    span,
			})
			Expect(read).To(BeEmpty())
			Expect(ack.Ack).To(BeFalse())
			failures := logs.FilterMessage("read of a channel failed").All()
			Expect(failures).To(HaveLen(1))
			Expect(failures[0].Level).To(Equal(zapcore.ErrorLevel))
			fields := failures[0].ContextMap()
			Expect(fields).To(HaveKeyWithValue("channel", data.Key().String()))
			Expect(fields).To(HaveKeyWithValue(
				"error",
				"channel [check_data]<"+data.Key().String()+">: "+ErrFault.Error(),
			))
			Expect(fields).To(HaveKey("time_range"))
			_, ack = stream.exec(iterator.Request{Command: iterator.CommandError})
			Expect(ack.Error).To(MatchError(ErrFault))
			Expect(stream.close()).To(MatchError(ErrFault))
		},
		Entry("automatic span", iterator.AutoSpan),
		Entry("fixed span", 20*telem.Second),
	)
})
