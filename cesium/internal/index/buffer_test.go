package index_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/index"
	"github.com/synnaxlabs/cesium/internal/position"
	"github.com/synnaxlabs/x/array"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("ThresholdBuffer", func() {
	Describe("Add and Search", func() {
		It("Should correctly write and search for a position in a set of alignments", func() {
			buf := index.ThresholdBuffer{}
			Expect(buf.Add([]index.Alignment{
				{
					Pos:   1,
					Stamp: 1,
				},
				{
					Pos:   2,
					Stamp: 3,
				},
				{
					Pos:   3,
					Stamp: 5,
				},
			})).To(Succeed())
			Expect(buf.Add([]index.Alignment{
				{
					Pos:   4,
					Stamp: 7,
				},
				{
					Pos:   5,
					Stamp: 9,
				},
			})).To(Succeed())
			p := MustSucceed(buf.SearchP(8, position.Uncertain))
			Expect(p).To(Equal(position.Between(4, 5)))
			p2 := MustSucceed(buf.SearchP(2, position.Uncertain))
			Expect(p2).To(Equal(position.Between(1, 2)))
		})
	})
	Describe("WriteToBelowThreshold", func() {
		It("Should flush any unneeded array chunks to the wrapped index", func() {
			underlying := &index.BinarySearch{
				Array: array.Searchable[index.Alignment]{
					Array: array.NewRolling[index.Alignment](10),
				},
			}
			buf := index.ThresholdBuffer{}
			Expect(buf.Add([]index.Alignment{
				{
					Pos:   1,
					Stamp: 1,
				},
				{
					Pos:   2,
					Stamp: 3,
				},
				{
					Pos:   3,
					Stamp: 5,
				},
			})).To(Succeed())
			Expect(buf.Add([]index.Alignment{
				{
					Pos:   4,
					Stamp: 7,
				},
				{
					Pos:   5,
					Stamp: 9,
				},
			})).To(Succeed())
			Expect(buf.WriteToBelowThreshold(2, underlying)).To(Succeed())
			Expect(underlying.Array.Len()).To(Equal(0))
			Expect(buf.WriteToBelowThreshold(3, underlying)).To(Succeed())
			Expect(underlying.Array.Len()).To(Equal(3))
		})
	})
})
