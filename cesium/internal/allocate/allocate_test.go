package allocate_test

import (
	"github.com/arya-analytics/cesium/internal/allocate"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type IntegerItem struct {
	key  int
	size int
}

func (i IntegerItem) Key() int {
	return i.key
}

func (i IntegerItem) Size() int {
	return i.size
}

var _ = Describe("Alloc", func() {
	Describe("Simple", func() {
		Context("Default Config", Ordered, func() {
			var (
				a allocate.Allocator[int, int, IntegerItem]
			)
			BeforeAll(func() {
				a = allocate.New[int, int, IntegerItem](&allocate.NextDescriptorInt{}, allocate.DefaultConfig())
			})
			It("Should allocate the first item to a brand new descriptor", func() {
				i := IntegerItem{key: 1, size: 1}
				d := a.Allocate(i)
				Expect(d[0]).To(Equal(1))
			})
			Context("Allocating a second item", func() {
				Specify("Same KVKey", func() {
					i := IntegerItem{key: 1, size: 1}
					d := a.Allocate(i)
					Expect(d[0]).To(Equal(1))
				})
				Specify("Different KVKey", func() {
					i := IntegerItem{key: 2, size: 1}
					d := a.Allocate(i)
					Expect(d[0]).To(Equal(2))
				})
			})
		})
	})
	Describe("Exceeding max descriptors", func() {
		It("Should start allocating to the next smallest descriptor", func() {
			a := allocate.New[int, int, IntegerItem](&allocate.NextDescriptorInt{}, allocate.Config{
				MaxDescriptors: 2,
			})
			i := IntegerItem{key: 1, size: 1}
			d := a.Allocate(i)
			Expect(d[0]).To(Equal(1))
			i = IntegerItem{key: 2, size: 1}
			d = a.Allocate(i)
			Expect(d[0]).To(Equal(2))
			i = IntegerItem{key: 3, size: 1}
			d = a.Allocate(i)
			Expect(d[0]).To(BeElementOf([]int{1, 2}))
			nd := a.Allocate(i)
			Expect(nd[0]).To(Equal(d[0]))
		})
	})
	Describe("Exceeding max descriptor size", func() {
		Context("Max descriptor count not exceeded", func() {
			It("Should allocate to a new descriptor", func() {
				a := allocate.New[int, int, IntegerItem](&allocate.NextDescriptorInt{}, allocate.Config{
					MaxSize: 2,
				})

				i := IntegerItem{key: 1, size: 1}
				d := a.Allocate(i)
				Expect(d[0]).To(Equal(1))

				i = IntegerItem{key: 1, size: 1}
				d = a.Allocate(i)
				Expect(d[0]).To(Equal(1))

				i = IntegerItem{key: 2, size: 1}
				d = a.Allocate(i)
				Expect(d[0]).To(Equal(2))

				i = IntegerItem{key: 1, size: 1}
				d = a.Allocate(i)

				Expect(d[0]).To(Equal(3))
			})
		})
		Context("Max descriptor count exceeded", func() {
			It("Should allocate to the next smallest descriptor", func() {

				a := allocate.New[int, int, IntegerItem](&allocate.NextDescriptorInt{}, allocate.Config{
					MaxSize:        2,
					MaxDescriptors: 2,
				})

				i := IntegerItem{key: 1, size: 1}
				d := a.Allocate(i)
				Expect(d[0]).To(Equal(1))

				i = IntegerItem{key: 1, size: 1}
				d = a.Allocate(i)
				Expect(d[0]).To(Equal(1))

				i = IntegerItem{key: 1, size: 1}
				d = a.Allocate(i)
				Expect(d[0]).To(Equal(2))

				i = IntegerItem{key: 1, size: 1}
				d = a.Allocate(i)
				Expect(d[0]).To(Equal(2))

			})
		})
	})
})
