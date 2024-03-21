package domain_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/domain"
	"github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
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
		db = MustSucceed(domain.Open(domain.Config{FS: fs.NewMem()}))
	})
	AfterEach(func() { Expect(db.Close()).To(Succeed()) })
	Describe("Delete a Timerange", func() {
		BeforeEach(func() {
			Expect(domain.Write(ctx, db, (10 * telem.SecondTS).SpanRange(10*telem.Second), []byte{10, 11, 12, 13, 14, 15, 16})).To(Succeed())
			Expect(domain.Write(ctx, db, (20 * telem.SecondTS).SpanRange(10*telem.Second), []byte{20, 21, 22, 23, 24, 25, 26, 27, 28, 29})).To(Succeed())
			Expect(domain.Write(ctx, db, (30 * telem.SecondTS).SpanRange(7*telem.Second), []byte{30, 31, 32, 33, 34, 35, 36})).To(Succeed())
		})
		It("Should delete a timerange", func() {
			// Note that we do not need to multiply by Density here since we are
			// directly writing bytes.
			Expect(db.Delete(ctx, 0, 2, int64(6), int64(5), telem.TimeRange{Start: 15 * telem.SecondTS, End: 32 * telem.SecondTS}, true)).To(Succeed())
			iter := db.NewIterator(domain.IteratorConfig{Bounds: telem.TimeRange{
				Start: 10 * telem.SecondTS,
				End:   36 * telem.SecondTS,
			}})
			Expect(iter.SeekFirst(ctx)).To(BeTrue())
			Expect(iter.TimeRange()).To(Equal((10 * telem.SecondTS).Range(15 * telem.SecondTS)))
			r := MustSucceed(iter.NewReader(ctx))
			p := make([]byte, 5)
			_, err := r.ReadAt(p, 0)
			Expect(err).ToNot(HaveOccurred())
			Expect(p[:5]).To(Equal([]byte{10, 11, 12, 13, 14}))

			Expect(iter.Next()).To(BeTrue())
			Expect(iter.TimeRange()).To(Equal((32 * telem.SecondTS).Range(37 * telem.SecondTS)))
			r = MustSucceed(iter.NewReader(ctx))
			p = make([]byte, 5)
			_, err = r.ReadAt(p, 0)
			Expect(err).ToNot(HaveOccurred())
			Expect(p[:5]).To(Equal([]byte{32, 33, 34, 35, 36}))
		})
		It("Should delete a timerange on the same pointer", func() {
			Expect(db.Delete(ctx, 1, 1, int64(2), int64(4), telem.TimeRange{Start: 22 * telem.SecondTS, End: 26 * telem.SecondTS}, true)).To(Succeed())
			iter := db.NewIterator(domain.IteratorConfig{Bounds: telem.TimeRange{
				Start: 20 * telem.SecondTS,
				End:   36 * telem.SecondTS,
			}})
			Expect(iter.SeekFirst(ctx)).To(BeTrue())
			Expect(iter.TimeRange()).To(Equal((20 * telem.SecondTS).Range(22 * telem.SecondTS)))
			r := MustSucceed(iter.NewReader(ctx))
			p := make([]byte, 2)
			_, err := r.ReadAt(p, 0)
			Expect(err).ToNot(HaveOccurred())
			Expect(p[:2]).To(Equal([]byte{20, 21}))

			Expect(iter.Next()).To(BeTrue())
			Expect(iter.TimeRange()).To(Equal((26 * telem.SecondTS).Range(30 * telem.SecondTS)))
			r = MustSucceed(iter.NewReader(ctx))
			p = make([]byte, 4)
			_, err = r.ReadAt(p, 0)
			Expect(err).ToNot(HaveOccurred())
			Expect(p[:4]).To(Equal([]byte{26, 27, 28, 29}))
		})
		Context("Edge cases", func() {
			It("Should return errors when the start pointer is invalid", func() {
				err := db.Delete(ctx, -1, 1, int64(2), int64(3), telem.TimeRange{Start: 22 * telem.SecondTS, End: 30 * telem.SecondTS}, true)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("starting at invalid position"))
			})

			It("Should return errors when the end pointer is invalid", func() {
				err := db.Delete(ctx, 1, 3, int64(2), int64(3), telem.TimeRange{Start: 22 * telem.SecondTS, End: 30 * telem.SecondTS}, true)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("ending at invalid position"))
			})

			It("Should return errors when the start pointer is greater than the end pointer", func() {
				err := db.Delete(ctx, 3, 1, int64(2), int64(3), telem.TimeRange{Start: 22 * telem.SecondTS, End: 30 * telem.SecondTS}, true)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("starting position cannot be greater than ending position"))
			})

			It("Should return errors when the start offset is greater than the length of the pointer", func() {
				err := db.Delete(ctx, 0, 1, int64(10), int64(3), telem.TimeRange{Start: 22 * telem.SecondTS, End: 30 * telem.SecondTS}, true)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("start offset cannot be greater than"))
			})

			It("Should return errors when the end offset is greater than the length of the pointer", func() {
				err := db.Delete(ctx, 0, 1, int64(2), int64(11), telem.TimeRange{Start: 22 * telem.SecondTS, End: 30 * telem.SecondTS}, true)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("end offset cannot be greater than"))
			})

			It("Should return errors when the startOffset is after the endOffset for same pointer deletion", func() {
				err := db.Delete(ctx, 0, 0, int64(4), int64(5), telem.TimeRange{Start: 22 * telem.SecondTS, End: 30 * telem.SecondTS}, true)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("exceed the length"))
			})

		})
	})
})
