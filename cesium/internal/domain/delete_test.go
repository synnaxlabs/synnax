package domain_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/domain"
	"github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/testutil"
)

// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

var _ = Describe("Delete", Ordered, func() {
	var db *domain.DB
	BeforeEach(func() {
		db = testutil.MustSucceed(domain.Open(domain.Config{FS: fs.NewMem()}))
	})
	AfterEach(func() { Expect(db.Close()).To(Succeed()) })
	Describe("Delete a Timerange", func() {
		BeforeEach(func() {
			Expect(domain.Write(ctx, db, (10 * telem.SecondTS).SpanRange(10*telem.Second), []byte{10, 11, 12, 13, 14, 15, 16})).To(Succeed())
			Expect(domain.Write(ctx, db, (20 * telem.SecondTS).SpanRange(10*telem.Second), []byte{20, 21, 22, 23, 24, 25, 26, 27, 28, 29})).To(Succeed())
			Expect(domain.Write(ctx, db, (30 * telem.SecondTS).SpanRange(7*telem.Second), []byte{30, 31, 32, 33, 34, 35, 36})).To(Succeed())
		})
		Describe("timerange and no offset", func() {
			Expect(db.Delete(ctx, telem.TimeRange{Start: 15 * telem.SecondTS, End: 32 * telem.SecondTS}, 0, 0)).To(Succeed())
			iter := db.NewIterator(domain.IteratorConfig{Bounds: telem.TimeRange{
				Start: 10 * telem.SecondTS,
				End:   36 * telem.SecondTS,
			}})
			Expect(iter.SeekFirst(ctx)).To(BeTrue())
			Expect(iter.TimeRange()).To(Equal((10 * telem.SecondTS).Range(15 * telem.SecondTS)))
			Expect(iter.Next()).To(BeTrue())
			Expect(iter.TimeRange()).To(Equal((32 * telem.SecondTS).Range(37 * telem.SecondTS)))
		})
	})
})
