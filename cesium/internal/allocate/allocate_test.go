package allocate_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/cesium/internal/allocate"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

type IntegerItem struct {
	key  int
	size telem.Size
}

func (i IntegerItem) Key() int {
	return i.key
}

func (i IntegerItem) Size() telem.Size {
	return i.size
}

var _ = Describe("Alloc", func() {
	Describe("Simple", func() {
		Context("Default Config", Ordered, func() {
			var (
				a Allocator[int, int]
			)
			BeforeAll(func() {
				a = New[int, int](NextDescriptorInt(), DefaultConfig)
			})
			It("Should allocate the first item to a brand new descriptor", func() {
				d := MustSucceed(a.Allocate(Item[int]{Key: 1, Size: 1}))
				Expect(d[0]).To(Equal(1))
			})
			Context("Allocating a second item", func() {
				Specify("Same KVKey", func() {
					d := MustSucceed(a.Allocate(Item[int]{Key: 1, Size: 1}))
					Expect(d[0]).To(Equal(1))
				})
				Specify("Different KVKey", func() {
					d := MustSucceed(a.Allocate(Item[int]{Key: 2, Size: 1}))
					Expect(d[0]).To(Equal(2))
				})
			})
		})
	})
	Describe("Exceeding max descriptors", func() {
		It("Should start allocating to the next smallest descriptor", func() {
			a := New[int, int](NextDescriptorInt(), Config{
				MaxDescriptors: 2,
			})
			d := MustSucceed(a.Allocate(Item[int]{Key: 1, Size: 1}))
			Expect(d[0]).To(Equal(1))
			d = MustSucceed(a.Allocate(Item[int]{Key: 2, Size: 1}))
			Expect(d[0]).To(Equal(2))
			d = MustSucceed(a.Allocate(Item[int]{Key: 3, Size: 1}))
			Expect(d[0]).To(BeElementOf([]int{1, 2}))
			nd := MustSucceed(a.Allocate(Item[int]{Key: 3, Size: 1}))
			Expect(nd[0]).To(Equal(d[0]))
		})
	})
	Describe("Exceeding max descriptor size", func() {
		Context("Max descriptor count not exceeded", func() {
			It("Should allocate to a new descriptor", func() {
				a := New[int, int](NextDescriptorInt(), Config{
					MaxSize: 2,
				})

				d := MustSucceed(a.Allocate(Item[int]{Key: 1, Size: 1}))
				Expect(d[0]).To(Equal(1))

				d = MustSucceed(a.Allocate(Item[int]{Key: 1, Size: 1}))
				Expect(d[0]).To(Equal(1))

				d = MustSucceed(a.Allocate(Item[int]{Key: 2, Size: 1}))
				Expect(d[0]).To(Equal(2))

				d = MustSucceed(a.Allocate(Item[int]{Key: 1, Size: 1}))

				Expect(d[0]).To(Equal(3))
			})
		})
		Context("Max descriptor count exceeded", func() {
			It("Should allocate to the next smallest descriptor", func() {

				a := New[int, int](NextDescriptorInt(), Config{
					MaxSize:        2,
					MaxDescriptors: 2,
				})

				d := MustSucceed(a.Allocate(Item[int]{Key: 1, Size: 1}))
				Expect(d[0]).To(Equal(1))

				d = MustSucceed(a.Allocate(Item[int]{Key: 1, Size: 1}))
				Expect(d[0]).To(Equal(1))

				d = MustSucceed(a.Allocate(Item[int]{Key: 1, Size: 1}))
				Expect(d[0]).To(Equal(2))

				d = MustSucceed(a.Allocate(Item[int]{Key: 1, Size: 1}))
				Expect(d[0]).To(Equal(2))
			})
		})
	})
})
