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
	"github.com/samber/lo"
	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/cesium/internal/virtual"
	"github.com/synnaxlabs/x/encoding/json"
	"github.com/synnaxlabs/x/io/fs"
	. "github.com/synnaxlabs/x/io/fs/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Write", func() {
	var db *virtual.DB
	BeforeEach(func(ctx SpecContext) {
		db = MustOpen(virtual.Open(ctx, virtual.Config{
			MetaCodec: json.Codec,
			Channel: channel.Channel{
				Name:     "Ray",
				Key:      2,
				DataType: telem.TimestampT,
				Virtual:  true,
			},
			FS: fs.NewMem(),
		}))
	})
	open := func(ctx SpecContext) *virtual.Writer {
		GinkgoHelper()
		return MustSucceed(db.OpenWriter(ctx, virtual.WriterConfig{
			Start: 10 * telem.SecondTS,
		}))
	}

	Describe("Write", func() {
		It("Should let every writer of a claim write", func(ctx SpecContext) {
			w1, w2 := open(ctx), open(ctx)
			a := MustSucceed(w1.Write(telem.NewSeriesSecondsTSV(10, 11, 12)))
			Expect(MustSucceed(w2.Write(telem.NewSeriesSecondsTSV(13)))).
				To(Equal(a.AddSamples(3)))
			Expect(w1.Close()).To(Succeed())
			Expect(w2.Close()).To(Succeed())
		})

		It(
			"Should give concurrent writers of one claim distinct alignments",
			func(ctx SpecContext) {
				const writes = 10_000
				writers := []*virtual.Writer{open(ctx), open(ctx)}
				alignments := make([][]telem.Alignment, len(writers))
				var wg sync.WaitGroup
				for i, w := range writers {
					wg.Go(func() {
						defer GinkgoRecover()
						for range writes {
							alignments[i] = append(
								alignments[i],
								MustSucceed(w.Write(telem.NewSeriesSecondsTSV(1))),
							)
						}
					})
				}
				wg.Wait()
				all := lo.Flatten(alignments)
				Expect(lo.Uniq(all)).To(HaveLen(len(all)))
				for _, w := range writers {
					Expect(w.Close()).To(Succeed())
				}
			},
		)

		It(
			"Should return an error when writing a series with the wrong data type",
			func(ctx SpecContext) {
				w := open(ctx)
				Expect(w.Write(telem.NewSeriesV[uint8](1, 2, 3))).
					Error().To(MatchError(validate.ErrValidation))
				Expect(w.Close()).To(Succeed())
			},
		)
	})

	Describe("Close", func() {
		It(
			"Should not return an error when the same writer is closed multiple times",
			func(ctx SpecContext) {
				w := open(ctx)
				Expect(w.Close()).To(Succeed())
				Expect(w.Close()).To(Succeed())
			},
		)

		It("Should return an error on Write when the writer is closed", func(
			ctx SpecContext,
		) {
			w := open(ctx)
			Expect(w.Close()).To(Succeed())
			Expect(w.Write(telem.NewSeriesSecondsTSV(10, 11, 12))).
				Error().To(MatchError(virtual.ErrWriterClosed))
		})
	})

	Describe("FS Interaction", func() {
		It(
			"Should not produce any data writes to the underlying FS",
			func(ctx SpecContext) {
				// Virtual channels are defined as not persisting data; this test pins
				// that contract directly by recording every Open, Read, ReadAt, Write,
				// and WriteAt against the underlying FS and asserting that nothing is
				// written during the writer's session.
				rec := NewRecorder(fs.NewMem())
				virtualDB := MustSucceed(virtual.Open(ctx, virtual.Config{
					MetaCodec: json.Codec,
					Channel: channel.Channel{
						Name:     "virtual-fs-check",
						Key:      99,
						DataType: telem.TimestampT,
						Virtual:  true,
					},
					FS: rec,
				}))
				rec.Reset()
				w := MustSucceed(virtualDB.OpenWriter(ctx, virtual.WriterConfig{
					Start: 10 * telem.SecondTS,
				}))
				MustSucceed(w.Write(telem.NewSeriesSecondsTSV(10, 11, 12, 13, 14)))
				Expect(w.Close()).To(Succeed())
				Expect(virtualDB.Close()).To(Succeed())
				Expect(rec.Count(MatchOp(OpWrite, OpWriteAt))).To(BeZero())
			},
		)
	})
})
