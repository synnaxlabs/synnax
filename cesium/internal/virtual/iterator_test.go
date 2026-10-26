// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package virtual_test

import (
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/cesium/internal/virtual"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/encoding/json"
	"github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Iterator", func() {
	const key channel.Key = 3
	var (
		db    *virtual.DB
		stamp = 10 * telem.SecondTS
	)
	stamped := func(at telem.TimeStamp, values ...int64) telem.Series {
		s := telem.NewSeriesV[int64](values...)
		s.TimeRange = at.SpanRange(telem.Nanosecond)
		return s
	}
	openWriter := func(ctx SpecContext, subject string) *virtual.Writer {
		GinkgoHelper()
		w, _ := MustSucceed2(db.OpenWriter(ctx, virtual.WriterConfig{
			Start:     telem.SecondTS,
			Authority: control.AuthorityAbsolute,
			Subject:   control.Subject{Key: subject},
		}))
		return w
	}
	write := func(ctx SpecContext, s telem.Series) telem.Alignment {
		GinkgoHelper()
		w := openWriter(ctx, "writer")
		a := MustSucceed(w.Write(s))
		MustSucceed(w.Close())
		return a
	}
	openIterator := func(bounds telem.TimeRange) *virtual.Iterator {
		GinkgoHelper()
		return MustSucceed(db.OpenIterator(virtual.IteratorConfig{Bounds: bounds}))
	}
	BeforeEach(func(ctx SpecContext) {
		db = MustOpen(virtual.Open(ctx, virtual.Config{
			MetaCodec: json.Codec,
			Channel: channel.Channel{
				Name:     "Retained",
				Key:      key,
				DataType: telem.Int64T,
				Virtual:  true,
			},
			FS: fs.NewMem(),
		}))
	})
	It("Should return the retained sample on SeekLast then Prev", func(
		ctx SpecContext,
	) {
		s := stamped(stamp, 1, 2, 3)
		a := write(ctx, s)
		i := openIterator(telem.TimeRangeMax)
		Expect(i.SeekLast(ctx)).To(BeTrue())
		Expect(i.Prev(ctx, channel.AutoSpan)).To(BeTrue())
		f := i.Value()
		Expect(f.Count()).To(Equal(1))
		got := f.Get(key).Series[0]
		Expect(got).To(telem.MatchSeriesDataV[int64](3))
		Expect(got.Alignment).To(Equal(a.AddSamples(2)))
		Expect(got.TimeRange).To(Equal(s.TimeRange))
		Expect(i.Prev(ctx, channel.AutoSpan)).To(BeFalse())
		Expect(i.Valid()).To(BeFalse())
		Expect(i.Close()).To(Succeed())
	})

	It("Should return the retained sample on SeekFirst then Next", func(
		ctx SpecContext,
	) {
		write(ctx, stamped(stamp, 1, 2, 3))
		i := openIterator(telem.TimeRangeMax)
		Expect(i.SeekFirst(ctx)).To(BeTrue())
		Expect(i.Next(ctx, channel.AutoSpan)).To(BeTrue())
		Expect(i.Value().Get(key).Series[0]).To(telem.MatchSeriesDataV[int64](3))
		Expect(i.Next(ctx, channel.AutoSpan)).To(BeFalse())
		Expect(i.Close()).To(Succeed())
	})

	It("Should step over the sample with an explicit span", func(ctx SpecContext) {
		write(ctx, stamped(stamp, 7))
		i := openIterator(telem.TimeRange{Start: 0, End: 20 * telem.SecondTS})
		Expect(i.SeekFirst(ctx)).To(BeTrue())
		Expect(i.Next(ctx, telem.Second)).To(BeTrue())
		Expect(i.Value().Get(key).Series[0]).To(telem.MatchSeriesDataV[int64](7))
		Expect(i.Next(ctx, telem.Second)).To(BeFalse())
		Expect(i.Prev(ctx, telem.Second)).To(BeTrue())
		Expect(i.Close()).To(Succeed())
	})

	It("Should keep only the newest write", func(ctx SpecContext) {
		write(ctx, stamped(stamp, 1, 2))
		second := stamped(stamp+telem.TimeStamp(telem.Second), 3, 4)
		write(ctx, second)
		i := openIterator(telem.TimeRangeMax)
		Expect(i.SeekLast(ctx)).To(BeTrue())
		Expect(i.Prev(ctx, channel.AutoSpan)).To(BeTrue())
		got := i.Value().Get(key).Series[0]
		Expect(got).To(telem.MatchSeriesDataV[int64](4))
		Expect(got.TimeRange).To(Equal(second.TimeRange))
		Expect(i.Close()).To(Succeed())
	})

	It("Should not retain an empty write", func(ctx SpecContext) {
		write(ctx, stamped(stamp, 1, 2))
		write(ctx, stamped(stamp+telem.TimeStamp(telem.Second)))
		i := openIterator(telem.TimeRangeMax)
		Expect(i.SeekLast(ctx)).To(BeTrue())
		Expect(i.Prev(ctx, channel.AutoSpan)).To(BeTrue())
		Expect(i.Value().Get(key).Series[0]).To(telem.MatchSeriesDataV[int64](2))
		Expect(i.Close()).To(Succeed())
	})

	It("Should see a write made after the iterator opened", func(ctx SpecContext) {
		i := openIterator(telem.TimeRangeMax)
		Expect(i.SeekLast(ctx)).To(BeFalse())
		write(ctx, stamped(stamp, 5))
		Expect(i.SeekLast(ctx)).To(BeTrue())
		Expect(i.Prev(ctx, channel.AutoSpan)).To(BeTrue())
		Expect(i.Value().Get(key).Series[0]).To(telem.MatchSeriesDataV[int64](5))
		Expect(i.Close()).To(Succeed())
	})

	DescribeTable("SeekLE", func(ctx SpecContext, offset telem.TimeSpan, ok bool) {
		write(ctx, stamped(stamp, 1))
		i := openIterator(telem.TimeRangeMax)
		Expect(i.SeekLE(ctx, stamp+telem.TimeStamp(offset))).To(Equal(ok))
		if ok {
			Expect(i.Prev(ctx, channel.AutoSpan) || i.Next(ctx, channel.AutoSpan)).
				To(BeTrue())
			Expect(i.Value().Get(key).Series[0]).To(telem.MatchSeriesDataV[int64](1))
		}
		Expect(i.Close()).To(Succeed())
	},
		Entry("before the sample", -telem.Nanosecond, false),
		Entry("at the sample", telem.TimeSpan(0), true),
		Entry("after the sample", telem.Second, true),
	)

	DescribeTable("SeekGE", func(ctx SpecContext, offset telem.TimeSpan, ok bool) {
		write(ctx, stamped(stamp, 1))
		i := openIterator(telem.TimeRangeMax)
		Expect(i.SeekGE(ctx, stamp+telem.TimeStamp(offset))).To(Equal(ok))
		if ok {
			Expect(i.Next(ctx, channel.AutoSpan)).To(BeTrue())
			Expect(i.Value().Get(key).Series[0]).To(telem.MatchSeriesDataV[int64](1))
		}
		Expect(i.Close()).To(Succeed())
	},
		Entry("before the sample", -telem.Second, true),
		Entry("at the sample", telem.TimeSpan(0), true),
		Entry("after the sample", telem.Nanosecond, false),
	)

	It("Should read the retained sample through MoveTo and Continue", func(
		ctx SpecContext,
	) {
		write(ctx, stamped(stamp, 4))
		i := openIterator(telem.TimeRangeMax)
		Expect(i.Bounds()).To(Equal(telem.TimeRangeMax))
		Expect(i.IndexKey()).To(Equal(key))
		from := stamp - telem.TimeStamp(telem.Second)
		i.MoveTo(ctx, from)
		Expect(i.View()).To(Equal(from.SpanRange(0)))
		Expect(i.EndAfter(ctx, 10)).To(Equal(stamp + 1))
		Expect(i.Continue(ctx, stamp+1)).To(BeTrue())
		Expect(i.View()).To(Equal(from.Range(stamp + 1)))
		Expect(i.Value().Get(key).Series[0]).To(telem.MatchSeriesDataV[int64](4))
		Expect(i.Close()).To(Succeed())
	})

	It("Should report no end past the retained sample", func(ctx SpecContext) {
		write(ctx, stamped(stamp, 4))
		i := openIterator(telem.TimeRangeMax)
		i.MoveTo(ctx, stamp+1)
		Expect(i.EndAfter(ctx, 10)).To(Equal(telem.TimeStampMax))
		Expect(i.Continue(ctx, stamp+telem.TimeStamp(telem.Second))).To(BeFalse())
		Expect(i.Close()).To(Succeed())
	})

	It("Should be invalid when the sample is outside the bounds", func(
		ctx SpecContext,
	) {
		write(ctx, stamped(stamp, 1))
		i := openIterator(telem.TimeRange{Start: 0, End: 5 * telem.SecondTS})
		Expect(i.SeekFirst(ctx)).To(BeFalse())
		Expect(i.SeekLast(ctx)).To(BeFalse())
		Expect(i.Next(ctx, channel.AutoSpan)).To(BeFalse())
		Expect(i.Error()).ToNot(HaveOccurred())
		Expect(i.Close()).To(Succeed())
	})

	It("Should be invalid with no error before any write", func(ctx SpecContext) {
		i := openIterator(telem.TimeRangeMax)
		Expect(i.SeekLast(ctx)).To(BeFalse())
		Expect(i.Prev(ctx, channel.AutoSpan)).To(BeFalse())
		Expect(i.Valid()).To(BeFalse())
		Expect(i.Error()).ToNot(HaveOccurred())
		Expect(i.Close()).To(Succeed())
	})

	It("Should reject operations after Close", func(ctx SpecContext) {
		write(ctx, stamped(stamp, 1))
		i := openIterator(telem.TimeRangeMax)
		Expect(i.Close()).To(Succeed())
		Expect(i.SeekLast(ctx)).To(BeFalse())
		Expect(i.Error()).To(MatchError(virtual.ErrIteratorClosed))
		Expect(i.Close()).To(Succeed())
	})

	It("Should not open on a closed DB", func(ctx SpecContext) {
		Expect(db.Close()).To(Succeed())
		Expect(db.OpenIterator(virtual.IteratorConfig{})).Error().
			To(MatchError(virtual.ErrDBClosed))
	})

	It("Should retain a sample from concurrent writers", func(ctx SpecContext) {
		first := openWriter(ctx, "first")
		second := openWriter(ctx, "second")
		var wg sync.WaitGroup
		for _, w := range []*virtual.Writer{first, second} {
			wg.Go(func() {
				defer GinkgoRecover()
				for n := range int64(50) {
					MustSucceed(w.Write(stamped(stamp, n)))
				}
			})
		}
		wg.Wait()
		MustSucceed(first.Close())
		MustSucceed(second.Close())
		i := openIterator(telem.TimeRangeMax)
		Expect(i.SeekLast(ctx)).To(BeTrue())
		Expect(i.Prev(ctx, channel.AutoSpan)).To(BeTrue())
		Expect(i.Value().Get(key).Series[0]).To(telem.MatchSeriesDataV[int64](49))
		Expect(i.Close()).To(Succeed())
	})
})
