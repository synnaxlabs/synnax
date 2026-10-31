// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package domain_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/domain"
	"github.com/synnaxlabs/cesium/internal/testutil"
	"github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("OnHistoryChange", func() {
	var (
		db      *domain.DB
		changes []telem.TimeRange
	)
	BeforeEach(func() {
		changes = nil
		db = MustOpen(domain.Open(domain.Config{
			FS: fs.NewMem(),
			OnHistoryChange: func(_ context.Context, tr telem.TimeRange) {
				changes = append(changes, tr)
			},
		}))
	})
	write := func(ctx context.Context, start, end telem.TimeStamp) {
		GinkgoHelper()
		tr := (start * telem.SecondTS).Range(end * telem.SecondTS)
		Expect(testutil.WriteDomain(ctx, db, tr, []byte{1, 2, 3})).To(Succeed())
	}

	It("Should not report domains written at the end", func(ctx SpecContext) {
		write(ctx, 10, 20)
		write(ctx, 30, 40)
		Expect(changes).To(BeEmpty())
	})

	It("Should report a domain written before the last", func(ctx SpecContext) {
		write(ctx, 30, 40)
		write(ctx, 10, 20)
		Expect(changes).To(Equal([]telem.TimeRange{
			(10 * telem.SecondTS).Range(20 * telem.SecondTS),
		}))
	})

	It("Should report a commit to a domain before the last", func(ctx SpecContext) {
		cfg := domain.WriterConfig{Start: 10 * telem.SecondTS}
		w := MustOpen(db.OpenWriter(ctx, cfg))
		MustSucceed(w.Write([]byte{1, 2}))
		Expect(w.Commit(ctx, 15*telem.SecondTS)).To(Succeed())
		write(ctx, 30, 40)
		MustSucceed(w.Write([]byte{3, 4}))
		Expect(w.Commit(ctx, 20*telem.SecondTS)).To(Succeed())
		Expect(changes).To(Equal([]telem.TimeRange{
			(15 * telem.SecondTS).Range(20 * telem.SecondTS),
		}))
	})

	It("Should report a delete", func(ctx SpecContext) {
		write(ctx, 10, 20)
		tr := (12 * telem.SecondTS).Range(15 * telem.SecondTS)
		Expect(db.Delete(ctx, tr, fixedOffset(1), fixedOffset(2))).To(Succeed())
		Expect(changes).To(Equal([]telem.TimeRange{tr}))
	})
})
