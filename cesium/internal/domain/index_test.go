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
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/alamos/testutil"
	"github.com/synnaxlabs/cesium/internal/domain"
	. "github.com/synnaxlabs/x/io/fs/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Index", func() {
	Describe("Write conflicts", func() {
		const (
			commits    = 2000
			conflictAt = 10 * telem.SecondTS
		)
		var db *domain.DB

		openWriter := func(ctx context.Context, start telem.TimeStamp) *domain.Writer {
			w := MustSucceed(db.OpenWriter(ctx, domain.WriterConfig{
				Start:                    start,
				AutoIndexPersistInterval: telem.Hour,
			}))
			MustSucceed(w.Write([]byte{1}))
			DeferCleanup(func() { Expect(w.Close()).To(Succeed()) })
			return w
		}

		BeforeEach(func() {
			db = MustSucceed(domain.Open(domain.Config{
				FS:              FileSystems["memFS"](),
				FileSize:        telem.Megabyte,
				Instrumentation: PanicLogger(),
			}))
			DeferCleanup(func() { Expect(db.Close()).To(Succeed()) })
		})

		// Each failed commit builds its error from the domain another commit is
		// growing, so the race detector fails these specs if that read leaves the lock.
		commitDuringConflicts := func(ctx context.Context, conflicting *domain.Writer) {
			grown := openWriter(ctx, conflictAt)
			Expect(grown.Commit(ctx, conflictAt+1)).To(Succeed())
			var wg sync.WaitGroup
			wg.Go(func() {
				defer GinkgoRecover()
				for range commits {
					Expect(
						conflicting.Commit(ctx, conflictAt+telem.SecondTS),
					).To(MatchError(domain.ErrWriteConflict))
				}
			})
			wg.Go(func() {
				defer GinkgoRecover()
				for i := range commits {
					Expect(grown.Commit(
						ctx,
						conflictAt+telem.TimeStamp(i)+2,
					)).To(Succeed())
				}
			})
			wg.Wait()
		}

		It("Should read the conflict of an insert under the lock", func(
			ctx SpecContext,
		) {
			commitDuringConflicts(ctx, openWriter(ctx, 5*telem.SecondTS))
		})

		It("Should read the conflict of an update under the lock", func(
			ctx SpecContext,
		) {
			w := openWriter(ctx, 5*telem.SecondTS)
			Expect(w.Commit(ctx, 6*telem.SecondTS)).To(Succeed())
			commitDuringConflicts(ctx, w)
		})
	})
})
