// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package index_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/alamos/testutil"
	"github.com/synnaxlabs/cesium/internal/domain"
	"github.com/synnaxlabs/cesium/internal/index"
	"github.com/synnaxlabs/cesium/internal/testutil"
	xfs "github.com/synnaxlabs/x/io/fs"
	. "github.com/synnaxlabs/x/io/fs/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Cache", func() {
	var (
		rec   *Recorder
		idx   *index.Domain
		cache *index.Cache
		tr    = (2 * telem.SecondTS).Range(7 * telem.SecondTS)
	)
	BeforeEach(func(ctx SpecContext) {
		rec = NewRecorder(MustSucceed(xfs.NewMem().Sub("index")))
		db := MustOpen(domain.Open(domain.Config{
			FS:              rec,
			Instrumentation: PanicLogger(),
		}))
		Expect(testutil.WriteDomain(
			ctx,
			db,
			(1 * telem.SecondTS).Range(9*telem.SecondTS+1),
			telem.NewSeriesSecondsTSV(1, 2, 3, 5, 7, 9).Data,
		)).To(Succeed())
		idx = &index.Domain{DB: db}
		cache = index.NewCache()
		rec.Reset()
	})
	reads := func() int {
		GinkgoHelper()
		n := rec.Count(MatchOp(OpReadAt))
		rec.Reset()
		return n
	}

	Describe("Distance", func() {
		It(
			"Should resolve a repeated lookup without reading the index",
			func(ctx SpecContext) {
				expected := MustSucceed(idx.Distance(ctx, tr, index.MustBeContinuous))
				Expect(reads()).To(BeNumerically(">", 0))
				Expect(cache.Distance(ctx, idx, tr, index.MustBeContinuous)).
					To(Equal(expected))
				Expect(reads()).To(BeNumerically(">", 0))
				Expect(cache.Distance(ctx, idx, tr, index.MustBeContinuous)).
					To(Equal(expected))
				Expect(reads()).To(BeZero())
			},
		)

		It("Should not cache an error", func(ctx SpecContext) {
			missing := (20 * telem.SecondTS).Range(30 * telem.SecondTS)
			Expect(cache.Distance(ctx, idx, missing, index.MustBeContinuous)).
				Error().To(MatchError(index.ErrDiscontinuous))
			Expect(cache.Distance(ctx, idx, missing, index.MustBeContinuous)).
				Error().To(MatchError(index.ErrDiscontinuous))
		})
	})

	Describe("Stamp", func() {
		It(
			"Should resolve a repeated lookup without reading the index",
			func(ctx SpecContext) {
				expected := MustSucceed(
					idx.Stamp(ctx, 1*telem.SecondTS, 3, index.AllowDiscontinuous),
				)
				reads()
				Expect(
					cache.Stamp(
						ctx,
						idx,
						1*telem.SecondTS,
						3,
						index.AllowDiscontinuous,
					),
				).
					To(Equal(expected))
				Expect(reads()).To(BeNumerically(">", 0))
				Expect(
					cache.Stamp(
						ctx,
						idx,
						1*telem.SecondTS,
						3,
						index.AllowDiscontinuous,
					),
				).
					To(Equal(expected))
				Expect(reads()).To(BeZero())
			},
		)
	})

	Describe("AppendSegments", func() {
		It("Should append cached segments after the given ones", func(ctx SpecContext) {
			expected := MustSucceed(
				idx.AppendSegments(ctx, nil, 2*telem.SecondTS, 0, 3),
			)
			reads()
			first := index.Segment{Len: 7}
			Expect(cache.AppendSegments(ctx, idx, nil, 2*telem.SecondTS, 0, 3)).
				To(Equal(expected))
			Expect(reads()).To(BeNumerically(">", 0))
			Expect(cache.AppendSegments(
				ctx,
				idx,
				[]index.Segment{first},
				2*telem.SecondTS,
				0,
				3,
			)).To(Equal(append([]index.Segment{first}, expected...)))
			Expect(reads()).To(BeZero())
		})
	})

	Describe("Reset", func() {
		It("Should read the index again after a reset", func(ctx SpecContext) {
			MustSucceed(cache.Distance(ctx, idx, tr, index.MustBeContinuous))
			reads()
			cache.Reset()
			MustSucceed(cache.Distance(ctx, idx, tr, index.MustBeContinuous))
			Expect(reads()).To(BeNumerically(">", 0))
		})
	})

	Describe("Nil", func() {
		It("Should read the index on each lookup", func(ctx SpecContext) {
			var nilCache *index.Cache
			expected := MustSucceed(idx.Distance(ctx, tr, index.MustBeContinuous))
			reads()
			for range 2 {
				Expect(nilCache.Distance(ctx, idx, tr, index.MustBeContinuous)).
					To(Equal(expected))
				Expect(reads()).To(BeNumerically(">", 0))
			}
		})
	})
})
