package index_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/index"
	"github.com/synnaxlabs/cesium/internal/position"
	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("BinarySearch", func() {
	var bsi *index.BinarySearch
	Context("Empty", func() {
		BeforeEach(func() {
			bsi = newRollingBinarySearchIndex(0)
		})
		Describe("SearchP", func() {
			It("Should return a completely uncertain approximation", func() {
				Expect(bsi.SearchP(0, position.Uncertain)).To(Equal(position.Uncertain))
			})
		})
		Describe("SearchTS", func() {
			It("Should return a completely uncertain approximation", func() {
				Expect(bsi.SearchTS(0, telem.Uncertain)).To(Equal(telem.Uncertain))
			})
		})
	})
	Context("Exact Match", func() {
		BeforeEach(func() {
			bsi = newRollingBinarySearchIndex(10)
			Expect(bsi.Write([]index.Alignment{{0, 0}, {2, 4}, {6, 8}})).To(Succeed())
		})
		Describe("SearchP", func() {
			It("Should return a completely certain approximation", func() {
				Expect(bsi.SearchP(0, position.Uncertain)).To(Equal(position.ExactlyAt(0)))
				Expect(bsi.SearchP(4, position.Uncertain)).To(Equal(position.ExactlyAt(2)))
			})
		})
		Describe("SearchTS", func() {
			It("Should return a completely certain approximation", func() {
				Expect(bsi.SearchTS(0, telem.Uncertain)).To(Equal(telem.ExactlyAt(0)))
				Expect(bsi.SearchTS(6, telem.Uncertain)).To(Equal(telem.ExactlyAt(8)))
			})
		})
	})
	Context("Inexact Match", func() {
		BeforeEach(func() {
			bsi = newRollingBinarySearchIndex(10)
			Expect(bsi.Write([]index.Alignment{{0, 0}, {2, 4}, {6, 8}})).To(Succeed())
		})
		Describe("SearchP", func() {
			It("Should return the correct approximation", func() {
				Expect(bsi.SearchP(1, position.Uncertain)).To(Equal(position.Between(0, 2)))
				Expect(bsi.SearchP(5, position.Uncertain)).To(Equal(position.Between(2, 6)))
			})
		})
		Describe("SearchTS", func() {
			It("Should return a completely certain approximation", func() {
				Expect(bsi.SearchTS(1, telem.Uncertain)).To(Equal(telem.Between(0, 4)))
				Expect(bsi.SearchTS(5, telem.Uncertain)).To(Equal(telem.Between(4, 8)))
			})
		})
	})
	Context("Before Start", func() {
		BeforeEach(func() {
			bsi = newRollingBinarySearchIndex(10)
			Expect(bsi.Write([]index.Alignment{{2, 4}, {6, 8}, {10, 12}})).To(Succeed())
		})
		Describe("SearchP", func() {
			It("Should return the correct approximation", func() {
				Expect(bsi.SearchP(0, position.Uncertain)).To(Equal(position.Before(2)))
			})
		})
		Describe("SearchTS", func() {
			It("Should return the correct approximation", func() {
				Expect(bsi.SearchTS(0, telem.Uncertain)).To(Equal(telem.Before(4)))
			})
		})
	})
	Context("After End", func() {
		BeforeEach(func() {
			bsi = newRollingBinarySearchIndex(10)
			Expect(bsi.Write([]index.Alignment{{2, 4}, {6, 8}, {10, 12}})).To(Succeed())
		})
		Describe("SearchP", func() {
			It("Should return the correct approximation", func() {
				Expect(bsi.SearchP(14, position.Uncertain)).To(Equal(position.After(10)))
			})
		})
		Describe("SearchTS", func() {
			It("Should return the correct approximation", func() {
				Expect(bsi.SearchTS(12, telem.Uncertain)).To(Equal(telem.After(12)))
			})
		})
	})
})
