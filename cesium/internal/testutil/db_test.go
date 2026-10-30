// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package testutil_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/cesium/internal/domain"
	"github.com/synnaxlabs/cesium/internal/testutil"
	"github.com/synnaxlabs/cesium/internal/unary"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/encoding/json"
	"github.com/synnaxlabs/x/io/fs"
	fstestutil "github.com/synnaxlabs/x/io/fs/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Domain", func() {
	var (
		faulty *fstestutil.FaultyFS
		db     *domain.DB
	)
	BeforeEach(func() {
		faulty = fstestutil.WrapFaultyFS(fs.NewMem())
		db = MustOpen(domain.Open(domain.Config{FS: faulty}))
	})

	Describe("WriteDomain", func() {
		It("Should write the data as one domain", func(ctx SpecContext) {
			tr := (10 * telem.SecondTS).SpanRange(10 * telem.Second)
			Expect(testutil.WriteDomain(ctx, db, tr, []byte{1, 2, 3})).To(Succeed())
			Expect(MustSucceed(db.HasDataFor(ctx, tr))).To(BeTrue())
		})

		It(
			"Should return an error when the range overlaps a domain",
			func(ctx SpecContext) {
				tr := (10 * telem.SecondTS).SpanRange(10 * telem.Second)
				Expect(testutil.WriteDomain(ctx, db, tr, []byte{1})).To(Succeed())
				Expect(testutil.WriteDomain(ctx, db, tr, []byte{2})).
					To(MatchError(domain.ErrWriteConflict))
			},
		)
	})

	Describe("ReadDomain", func() {
		It(
			"Should return the data of each overlapping domain in time order",
			func(ctx SpecContext) {
				second := (20 * telem.SecondTS).SpanRange(10 * telem.Second)
				first := (10 * telem.SecondTS).SpanRange(10 * telem.Second)
				Expect(
					testutil.WriteDomain(ctx, db, second, []byte{3, 4}),
				).To(Succeed())
				Expect(testutil.WriteDomain(ctx, db, first, []byte{1, 2})).To(Succeed())
				Expect(testutil.ReadDomain(ctx, db, telem.TimeRangeMax)).
					To(Equal([]byte{1, 2, 3, 4}))
			},
		)

		It("Should return no data for a range with no domains", func(ctx SpecContext) {
			Expect(testutil.ReadDomain(ctx, db, telem.TimeRangeMax)).To(BeEmpty())
		})

		It(
			"Should return an error when the data cannot be read",
			func(ctx SpecContext) {
				tr := (10 * telem.SecondTS).SpanRange(10 * telem.Second)
				Expect(testutil.WriteDomain(ctx, db, tr, []byte{1, 2})).To(Succeed())
				faulty.SetOptions(fstestutil.WithFailReadAt())
				Expect(testutil.ReadDomain(ctx, db, tr)).Error().
					To(MatchError(fstestutil.ErrFault))
			},
		)
	})
})

var _ = Describe("ReadUnary", func() {
	open := func(ctx SpecContext) *unary.DB {
		GinkgoHelper()
		return MustSucceed(unary.Open(ctx, unary.Config{
			FS:        fs.NewMem(),
			MetaCodec: json.Codec,
			Channel: channel.Channel{
				Name:     "time",
				Key:      1,
				DataType: telem.TimestampT,
				IsIndex:  true,
				Index:    1,
			},
		}))
	}

	It("Should return the samples that overlap the range", func(ctx SpecContext) {
		db := DeferClose(open(ctx))
		w, _ := MustSucceed2(db.OpenWriter(ctx, unary.WriterConfig{
			Start:     10 * telem.SecondTS,
			Authority: control.AuthorityAbsolute,
			Subject:   control.Subject{Key: "writer"},
		}))
		MustSucceed(w.Write(telem.NewSeriesSecondsTSV(10, 11, 12)))
		MustSucceed(w.Commit(ctx))
		MustSucceed(w.Close())
		frame := MustSucceed(testutil.ReadUnary(
			ctx,
			db,
			(11 * telem.SecondTS).Range(13*telem.SecondTS),
		))
		Expect(frame.SeriesAt(0).Data).
			To(Equal(telem.NewSeriesSecondsTSV(11, 12).Data))
	})

	It("Should return an empty frame when the DB has no data", func(ctx SpecContext) {
		db := DeferClose(open(ctx))
		Expect(testutil.ReadUnary(ctx, db, telem.TimeRangeMax)).To(BeZero())
	})

	It("Should return an error when the DB is closed", func(ctx SpecContext) {
		db := open(ctx)
		Expect(db.Close()).To(Succeed())
		Expect(testutil.ReadUnary(ctx, db, telem.TimeRangeMax)).Error().
			To(MatchError(unary.ErrDBClosed))
	})
})
