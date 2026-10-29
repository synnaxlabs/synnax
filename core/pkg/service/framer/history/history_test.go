// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package history_test

import (
	"context"
	"slices"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	distframer "github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/framer/history"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// openStreamer streams sy_history_change. next flushes the publisher, then reads
// published lists until one holds a change to key and returns that change.
func openStreamer(ctx context.Context, key channel.Key) (
	next func() distframer.HistoryChange,
) {
	GinkgoHelper()
	var sigCh channel.Channel
	Expect(channelSvc.NewRetrieve().
		Where(channel.MatchNames("sy_history_change")).
		Entry(&sigCh).
		Exec(ctx, nil),
	).To(Succeed())
	streamer := MustSucceed(framerSvc.NewStreamer(ctx, framer.StreamerConfig{
		Keys:        channel.Keys{sigCh.Key()},
		SendOpenAck: true,
	}))
	requests, responses := confluence.Attach(streamer, 10)
	sCtx, cancel := signal.Isolated()
	streamer.Flow(sCtx, confluence.CloseOutputInletsOnExit())
	DeferCleanup(func() {
		requests.Close()
		Eventually(responses.Outlet()).Should(BeClosed())
		cancel()
	})
	Eventually(responses.Outlet()).Should(Receive())
	return func() distframer.HistoryChange {
		GinkgoHelper()
		Eventually(clock.Pending).Should(Equal(1))
		clock.Advance(flushInterval.Duration())
		var change distframer.HistoryChange
		Eventually(func(g Gomega) {
			var res framer.StreamerResponse
			g.Expect(responses.Outlet()).To(Receive(&res))
			lists := MustSucceed(
				res.Frame.SeriesAt(0).DecodeJSON[[]distframer.HistoryChange](),
			)
			var found bool
			change, found = lo.Find(
				slices.Concat(lists...),
				func(c distframer.HistoryChange) bool { return c.Key == key },
			)
			g.Expect(found).To(BeTrue())
		}).Should(Succeed())
		return change
	}
}

func createIndex(ctx context.Context) channel.Key {
	GinkgoHelper()
	ch := channel.Channel{
		Name:     UniqueChannelName(),
		DataType: telem.TimestampT,
		IsIndex:  true,
	}
	Expect(channelSvc.NewWriter(nil).Create(ctx, &ch)).To(Succeed())
	return ch.Key()
}

func openWriter(
	ctx context.Context,
	key channel.Key,
	start telem.TimeStamp,
) *framer.Writer {
	GinkgoHelper()
	w := MustSucceed(framerSvc.OpenWriter(ctx, framer.WriterConfig{
		Keys:             channel.Keys{key},
		Start:            start,
		EnableAutoCommit: new(false),
	}))
	DeferCleanup(func() { Expect(w.Close()).To(Succeed()) })
	return w
}

func writeAndCommit(w *framer.Writer, key channel.Key, stamps ...telem.TimeStamp) {
	GinkgoHelper()
	Expect(w.Write(frame.NewUnary(key, telem.NewSeriesSecondsTSV(stamps...)))).
		To(BeTrue())
	MustSucceed(w.Commit())
}

// write writes stamps to key in one writer and closes it, releasing control so a later
// write or delete can take the region.
func write(ctx context.Context, key channel.Key, stamps ...telem.TimeStamp) {
	GinkgoHelper()
	w := MustSucceed(framerSvc.OpenWriter(ctx, framer.WriterConfig{
		Keys:  channel.Keys{key},
		Start: stamps[0] * telem.SecondTS,
	}))
	writeAndCommit(w, key, stamps...)
	Expect(w.Close()).To(Succeed())
}

func historyChange(
	key channel.Key,
	start, end telem.TimeStamp,
) distframer.HistoryChange {
	return distframer.HistoryChange{Key: key, TimeRange: start.Range(end)}
}

var _ = Describe("History", func() {
	It("Should publish a write into the past", func(ctx SpecContext) {
		key := createIndex(ctx)
		next := openStreamer(ctx, key)
		write(ctx, key, 20, 21)
		write(ctx, key, 10, 11)
		Expect(next()).To(Equal(
			historyChange(key, 10*telem.SecondTS, 11*telem.SecondTS+1),
		))
	})

	It(
		"Should merge changes to one channel within a flush into one change",
		func(ctx SpecContext) {
			key := createIndex(ctx)
			next := openStreamer(ctx, key)
			write(ctx, key, 20, 21)
			write(ctx, key, 10, 11)
			write(ctx, key, 5, 6)
			Expect(next()).To(Equal(
				historyChange(key, 5*telem.SecondTS, 11*telem.SecondTS+1),
			))
		},
	)

	It(
		"Should not publish writes at the end of the channel",
		func(ctx SpecContext) {
			key := createIndex(ctx)
			w := openWriter(ctx, key, 10*telem.SecondTS)
			writeAndCommit(w, key, 10, 11)
			writeAndCommit(w, key, 12, 13)
			Consistently(clock.Pending).Should(BeZero())
		},
	)

	It("Should publish a delete", func(ctx SpecContext) {
		key := createIndex(ctx)
		next := openStreamer(ctx, key)
		write(ctx, key, 10, 11, 12, 13)
		tr := (11 * telem.SecondTS).Range(13 * telem.SecondTS)
		Expect(framerSvc.DeleteTimeRange(ctx, channel.Keys{key}, tr)).To(Succeed())
		Expect(next()).To(Equal(distframer.HistoryChange{Key: key, TimeRange: tr}))
	})

	DescribeTable(
		"Should reject an invalid config",
		func(ctx SpecContext, cfg history.Config, msg string) {
			Expect(history.Publish(ctx, cfg)).Error().To(MatchError(msg))
		},
		Entry(
			"without a framer or signals",
			history.Config{},
			"framer: must be non-nil\nsignals: must be non-nil",
		),
		Entry(
			"with a negative flush interval",
			history.Config{FlushInterval: -telem.Millisecond},
			"framer: must be non-nil\nsignals: must be non-nil\n"+
				"flush_interval: must be positive",
		),
	)
})
