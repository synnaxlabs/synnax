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
		"Should panic when the resolvers of a delete keep a part that ends before it starts",
		func(ctx SpecContext, headEnd, tailStart telem.TimeStamp) {
			db := DeferClose(MustSucceed(domain.Open(domain.Config{
				FS:              xfs.NewMem(),
				Instrumentation: PanicLogger(),
			})))
			Expect(testutil.WriteDomain(
				ctx,
				db,
				(10 * telem.SecondTS).SpanRange(10*telem.Second),
				make([]byte, 10),
			)).To(Succeed())
			Expect(func() {
				MustSucceed(db.Delete(
					ctx,
					(12 * telem.SecondTS).Range(14*telem.SecondTS),
					fixedOffsetAndTimeStamp(2, headEnd),
					fixedOffsetAndTimeStamp(6, tailStart),
				))
			}).To(PanicWith(MatchError(ContainSubstring("ends before it starts"))))
		},
		Entry("a kept head", 5*telem.SecondTS, 14*telem.SecondTS),
		Entry("a kept tail", 12*telem.SecondTS, 25*telem.SecondTS),
	)
})
