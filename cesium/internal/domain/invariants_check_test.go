// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//go:build invariants || race

package domain_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/alamos/testutil"
	"github.com/synnaxlabs/cesium/internal/domain"
	"github.com/synnaxlabs/cesium/internal/testutil"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Invariants check", func() {
	DescribeTable(
		"Should panic when a delete keeps one part that overlaps an old neighbor",
		func(
			ctx SpecContext,
			tr telem.TimeRange,
			startOffset, endOffset telem.Size,
			kept telem.TimeStamp,
			panicText string,
		) {
			db := DeferClose(MustSucceed(domain.Open(domain.Config{
				FS:              xfs.NewMem(),
				Instrumentation: PanicLogger(),
			})))
			starts := []telem.TimeStamp{10 * telem.SecondTS, 30 * telem.SecondTS}
			for _, start := range starts {
				Expect(testutil.WriteDomain(
					ctx,
					db,
					start.SpanRange(10*telem.Second),
					make([]byte, 10),
				)).To(Succeed())
			}
			Expect(func() {
				Expect(db.Delete(
					ctx,
					tr,
					fixedOffsetAndTimeStamp(startOffset, kept),
					fixedOffsetAndTimeStamp(endOffset, kept),
				)).To(Succeed())
			}).To(PanicWith(MatchError(ContainSubstring(panicText))))
		},
		// The delete ends in the gap, so only the head of the first domain stays, and
		// it ends inside the second.
		Entry(
			"a kept head that ends after the next domain starts",
			(15*telem.SecondTS).Range(25*telem.SecondTS),
			telem.Size(5),
			telem.Size(0),
			35*telem.SecondTS,
			"domain pointers 0 and 1 are not sorted and apart",
		),
		// The delete starts in the gap, so only the tail of the second domain stays,
		// and it starts inside the first.
		Entry(
			"a kept tail that starts before the previous domain ends",
			(25*telem.SecondTS).Range(33*telem.SecondTS),
			telem.Size(0),
			telem.Size(3),
			15*telem.SecondTS,
			"domain pointers 0 and 1 are not sorted and apart",
		),
	)
})
