package index_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/index"
	"github.com/synnaxlabs/cesium/internal/position"
	"github.com/synnaxlabs/x/array"
)

func newRollingBinarySearchIndex(cap int) *index.BinarySearch {
	return &index.BinarySearch{Array: array.Searchable[index.Alignment]{
		Array: array.NewRolling[index.Alignment](cap),
	}}
}

var _ = Describe("Compound", func() {
	Context("Exact Match in first index", func() {
		Describe("SeekP", func() {
			It("Should return a certain approximation", func() {
				i1 := newRollingBinarySearchIndex(10)
				i2 := newRollingBinarySearchIndex(10)
				c := index.CompoundSearcher{i1, i2}
				Expect(i1.Write([]index.Alignment{{0, 0}, {2, 4}, {6, 8}})).To(Succeed())
				Expect(i2.Write([]index.Alignment{{0, 0}, {2, 4}, {6, 8}})).To(Succeed())
				Expect(c.SearchP(4, position.Uncertain)).To(Equal(position.ExactlyAt(2)))
			})
		})
	})
	Context("Exact Match in second index", func() {
		Describe("SeekP", func() {
			It("Should return a certain approximation", func() {
				i1 := newRollingBinarySearchIndex(10)
				i2 := newRollingBinarySearchIndex(10)
				c := index.CompoundSearcher{i1, i2}
				Expect(i1.Write([]index.Alignment{{6, 8}, {10, 12}, {7, 14}})).To(Succeed())
				Expect(i2.Write([]index.Alignment{{0, 0}, {2, 4}, {6, 8}})).To(Succeed())
				Expect(c.SearchP(8, position.Uncertain)).To(Equal(position.ExactlyAt(6)))
			})
		})
	})
	Context("Inexact Match in both indexes", func() {
		Describe("SeekP", func() {
			It("Should return the least uncertain approximation", func() {
				i1 := newRollingBinarySearchIndex(10)
				i2 := newRollingBinarySearchIndex(10)
				c := index.CompoundSearcher{i1, i2}
				Expect(i1.Write([]index.Alignment{{2, 4}, {6, 8}, {7, 14}})).To(Succeed())
				Expect(i2.Write([]index.Alignment{{0, 0}, {3, 5}, {6, 8}})).To(Succeed())
				Expect(c.SearchP(7, position.Uncertain)).To(Equal(position.Between(3, 6)))
			})
		})
	})
})
