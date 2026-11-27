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

var _ = Describe("Invariants", func() {
	DescribeTable(
		"Should panic when the resolvers of a delete place a kept part out of order",
		func(
			ctx SpecContext,
			headEnd telem.TimeStamp,
			endOffset telem.Size,
			tailStart telem.TimeStamp,
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
					(12 * telem.SecondTS).Range(14*telem.SecondTS),
					fixedOffsetAndTimeStamp(2, headEnd),
					fixedOffsetAndTimeStamp(endOffset, tailStart),
				)).To(Succeed())
			}).To(PanicWith(MatchError(ContainSubstring(panicText))))
		},
		Entry(
			"a kept head that ends before it starts",
			5*telem.SecondTS,
			telem.Size(6),
			14*telem.SecondTS,
			"ends before it starts",
		),
		Entry(
			"a kept tail that ends before it starts",
			12*telem.SecondTS,
			telem.Size(6),
			25*telem.SecondTS,
			"ends before it starts",
		),
		Entry(
			"a kept head that ends after the kept tail starts",
			35*telem.SecondTS,
			telem.Size(6),
			14*telem.SecondTS,
			"are not sorted and apart",
		),
	)
})
