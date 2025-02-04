// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package deque_test

import (
	"fmt"
	"unicode"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/deque"
)

const minCapacity int = 16

var _ = Describe("Deque", func() {
	It("TestEmpty", func() {
		q := deque.New[string]()
		Expect(q.Len()).To(Equal(0))
		Expect(q.Cap()).To(Equal(0))
		f := func(item string) bool {
			return true
		}
		Expect(q.Index(f)).To(Equal(-1))
		Expect(q.RIndex(f)).To(Equal(-1))
		Expect(q.Empty()).To(BeTrue())
	})
	It("TestNil", func() {
		var q *deque.Deque[int]
		Expect(q.Len()).To(Equal(0))
		Expect(q.Cap()).To(Equal(0))
		q.Rotate(5)
		f := func(item int) bool {
			return true
		}
		Expect(q.Index(f)).To(Equal(-1))
		Expect(q.RIndex(f)).To(Equal(-1))
	})
	It("TestFrontBack", func() {
		var q deque.Deque[string]
		q.PushBack("foo")
		q.PushBack("bar")
		q.PushBack("baz")
		Expect(q.Front()).To(Equal("foo"))
		Expect(q.Back()).To(Equal("baz"))
		Expect(q.PopFront()).To(Equal("foo"))
		Expect(q.Front()).To(Equal("bar"))
		Expect(q.Back()).To(Equal("baz"))
		Expect(q.PopBack()).To(Equal("baz"))
		Expect(q.Front()).To(Equal("bar"))
		Expect(q.Back()).To(Equal("bar"))
		Expect(q.Len()).To(Equal(1))
	})
	It("TestGrowShrinkBack", func() {
		var q deque.Deque[int]
		size := minCapacity * 2

		for i := 0; i < size; i++ {
			Expect(q.Len()).To(Equal(i))
			q.PushBack(i)
		}
		bufLen := q.Cap()

		for i := size; i > 0; i-- {
			Expect(q.Len()).To(Equal(i))
			x := q.PopBack()
			Expect(x).To(Equal(i - 1))
		}
		Expect(q.Len()).To(Equal(0))
		Expect(q.Cap()).ToNot(Equal(bufLen))
	})
	It("TestGrowShrinkFront", func() {
		var q deque.Deque[int]
		size := minCapacity * 2

		for i := 0; i < size; i++ {
			Expect(q.Len()).To(Equal(i))
			q.PushBack(i)
		}
		bufLen := q.Cap()

		// Remove from Front
		for i := 0; i < size; i++ {
			Expect(q.Len()).To(Equal(size - i))
			x := q.PopFront()
			Expect(x).To(Equal(i))
		}
		Expect(q.Len()).To(Equal(0))
		Expect(q.Cap()).ToNot(Equal(bufLen))
	})
	It("TestSimple", func() {
		var q deque.Deque[int]

		for i := 0; i < minCapacity; i++ {
			q.PushBack(i)
		}
		Expect(q.Front()).To(Equal(0))
		Expect(q.Back()).To(Equal(minCapacity - 1))

		for i := 0; i < minCapacity; i++ {
			Expect(q.Front()).To(Equal(i))
			x := q.PopFront()
			Expect(x).To(Equal(i))
		}

		q.Clear()
		for i := 0; i < minCapacity; i++ {
			q.PushFront(i)
		}
		for i := minCapacity - 1; i >= 0; i-- {
			x := q.PopFront()
			Expect(x).To(Equal(i))
		}
	})
	It("TestBufferWrap", func() {
		var q deque.Deque[int]

		for i := 0; i < minCapacity; i++ {
			q.PushBack(i)
		}

		for i := 0; i < 3; i++ {
			q.PopFront()
			q.PushBack(minCapacity + i)
		}

		for i := 0; i < minCapacity; i++ {
			Expect(q.Front()).To(Equal(i + 3))
			q.PopFront()
		}
	})
	It("TestBufferWrapReverse", func() {
		var q deque.Deque[int]

		for i := 0; i < minCapacity; i++ {
			q.PushFront(i)
		}
		for i := 0; i < 3; i++ {
			q.PopBack()
			q.PushFront(minCapacity + i)
		}

		for i := 0; i < minCapacity; i++ {
			Expect(q.Back()).To(Equal(i + 3))
			q.PopBack()
		}
	})
	It("TestLen", func() {
		var q deque.Deque[int]

		Expect(q.Len()).To(Equal(0))

		for i := 0; i < 1000; i++ {
			q.PushBack(i)
			Expect(q.Len()).To(Equal(i + 1))
		}
		for i := 0; i < 1000; i++ {
			q.PopFront()
			Expect(q.Len()).To(Equal(1000 - i - 1))
		}
	})
	It("TestBack", func() {
		var q deque.Deque[int]

		for i := 0; i < minCapacity+5; i++ {
			q.PushBack(i)
			Expect(q.Back()).To(Equal(i))
		}
	})
	It("TestNew", func() {
		minCap := 64
		q := deque.New[string](0, minCap)
		Expect(q.Cap()).To(Equal(0))
		q.PushBack("foo")
		q.PopFront()
		Expect(q.Len()).To(Equal(0))
		Expect(q.Cap()).To(Equal(minCap))

		curCap := 128
		q = deque.New[string](curCap, minCap)
		Expect(q.Cap()).To(Equal(curCap))
		Expect(q.Len()).To(Equal(0))
		q.PushBack("foo")
		Expect(q.Cap()).To(Equal(curCap))
	})
	It("TestRotate", func() {
		checkRotate := func(size int) {
			var q deque.Deque[int]
			for i := 0; i < size; i++ {
				q.PushBack(i)
			}

			for i := 0; i < q.Len(); i++ {
				x := i
				for n := 0; n < q.Len(); n++ {
					Expect(q.At(n)).To(Equal(x))
					x++
					if x == q.Len() {
						x = 0
					}
				}
				q.Rotate(1)
				Expect(q.Back()).To(Equal(i))
			}
			for i := q.Len() - 1; i >= 0; i-- {
				q.Rotate(-1)
				Expect(q.Front()).To(Equal(i))
			}
		}

		checkRotate(10)
		checkRotate(minCapacity)
		checkRotate(minCapacity + minCapacity/2)

		var q deque.Deque[int]
		for i := 0; i < 10; i++ {
			q.PushBack(i)
		}
		q.Rotate(11)
		Expect(q.Front()).To(Equal(1))
		q.Rotate(-21)
		Expect(q.Front()).To(Equal(0))
		q.Rotate(q.Len())
		Expect(q.Front()).To(Equal(0))
		q.Clear()
		q.PushBack(0)
		q.Rotate(13)
		Expect(q.Front()).To(Equal(0))
	})
	It("TestAt", func() {
		var q deque.Deque[int]

		for i := 0; i < 1000; i++ {
			q.PushBack(i)
		}

		// Front to back.
		for j := 0; j < q.Len(); j++ {
			Expect(q.At(j)).To(Equal(j))
		}

		// Back to front
		for j := 1; j <= q.Len(); j++ {
			Expect(q.At(q.Len() - j)).To(Equal(q.Len() - j))
		}
	})
	It("TestSet", func() {
		var q deque.Deque[int]

		for i := 0; i < 1000; i++ {
			q.PushBack(i)
			q.Set(i, i+50)
		}

		// Front to back.
		for j := 0; j < q.Len(); j++ {
			Expect(q.At(j)).To(Equal(j + 50))
		}
	})
	It("TestClear", func() {
		var q deque.Deque[int]

		for i := 0; i < 100; i++ {
			q.PushBack(i)
		}
		Expect(q.Len()).To(Equal(100))
		cap := q.Cap()
		q.Clear()
		Expect(q.Len()).To(Equal(0))
		Expect(q.Cap()).To(Equal(cap))
	})
	It("TestIndex", func() {
		var q deque.Deque[rune]
		for _, x := range "Hello, 世界" {
			q.PushBack(x)
		}
		idx := q.Index(func(item rune) bool {
			c := item
			return unicode.Is(unicode.Han, c)
		})
		Expect(idx).To(Equal(7))
		idx = q.Index(func(item rune) bool {
			c := item
			return c == 'H'
		})
		Expect(idx).To(Equal(0))
		idx = q.Index(func(item rune) bool {
			return false
		})
		Expect(idx).To(Equal(-1))
	})
	It("TestRIndex", func() {
		var q deque.Deque[rune]
		for _, x := range "Hello, 世界" {
			q.PushBack(x)
		}
		idx := q.RIndex(func(item rune) bool {
			c := item
			return unicode.Is(unicode.Han, c)
		})
		Expect(idx).To(Equal(8))
		idx = q.RIndex(func(item rune) bool {
			c := item
			return c == 'H'
		})
		Expect(idx).To(Equal(0))
		idx = q.RIndex(func(item rune) bool {
			return false
		})
		Expect(idx).To(Equal(-1))
	})
	It("TestInsert", func() {
		q := new(deque.Deque[rune])
		for _, x := range "ABCDEFG" {
			q.PushBack(x)
		}
		q.Insert(4, 'x') // ABCDxEFG
		Expect(q.At(4)).To(Equal('x'))

		q.Insert(2, 'y') // AByCDxEFG
		Expect(q.At(2)).To(Equal('y'))
		Expect(q.At(5)).To(Equal('x'))

		q.Insert(0, 'b') // bAByCDxEFG
		Expect(q.Front()).To(Equal('b'))

		q.Insert(q.Len(), 'e') // bAByCDxEFGe

		for _, x := range "bAByCDxEFGe" {
			Expect(q.PopFront()).To(Equal(x))
		}

		qs := deque.New[string](16)

		for i := 0; i < qs.Cap(); i++ {
			qs.PushBack(fmt.Sprint(i))
		}
		// deque: 0 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15
		// buffer: [0,1,2,3,4,5,6,7,8,9,10,11,12,13,14,15]
		for i := 0; i < qs.Cap()/2; i++ {
			qs.PopFront()
		}
		// deque: 8 9 10 11 12 13 14 15
		// buffer: [_,_,_,_,_,_,_,_,8,9,10,11,12,13,14,15]
		for i := 0; i < qs.Cap()/4; i++ {
			qs.PushBack(fmt.Sprint(qs.Cap() + i))
		}
		// deque: 8 9 10 11 12 13 14 15 16 17 18 19
		// buffer: [16,17,18,19,_,_,_,_,8,9,10,11,12,13,14,15]

		at := qs.Len() - 2
		qs.Insert(at, "x")
		// deque: 8 9 10 11 12 13 14 15 16 17 x 18 19
		// buffer: [16,17,x,18,19,_,_,_,8,9,10,11,12,13,14,15]
		Expect(qs.At(at)).To(Equal("x"))

		qs.Insert(2, "y")
		// deque: 8 9 y 10 11 12 13 14 15 16 17 x 18 19
		// buffer: [16,17,x,18,19,_,_,8,9,y,10,11,12,13,14,15]
		Expect(qs.At(2)).To(Equal("y"))
		Expect(qs.At(at + 1)).To(Equal("x"))

		qs.Insert(0, "b")
		// deque: b 8 9 y 10 11 12 13 14 15 16 17 x 18 19
		// buffer: [16,17,x,18,19,_,b,8,9,y,10,11,12,13,14,15]
		Expect(qs.Front()).To(Equal("b"))

		qs.Insert(qs.Len(), "e")
		Expect(qs.Cap()).To(Equal(qs.Len()))

		// deque: b 8 9 y 10 11 12 13 14 15 16 17 x 18 19 e
		// buffer: [16,17,x,18,19,e,b,8,9,y,10,11,12,13,14,15]
		for _, x := range []string{"b", "8", "9", "y", "10", "11", "12", "13", "14", "15", "16", "17", "x", "18", "19", "e"} {
			Expect(qs.Front()).To(Equal(x))
			qs.PopFront()
		}
	})
	It("TestRemove", func() {
		q := new(deque.Deque[rune])
		for _, x := range "ABCDEFG" {
			q.PushBack(x)
		}

		Expect(q.Remove(4)).To(Equal('E')) // ABCDFG

		Expect(q.Remove(2)).To(Equal('C')) // ABDFG
		Expect(q.Back()).To(Equal('G'))

		Expect(q.Remove(0)).To(Equal('A')) // BDFG
		Expect(q.Front()).To(Equal('B'))

		Expect(q.Remove(q.Len() - 1)).To(Equal('G')) // BDF
		Expect(q.Back()).To(Equal('F'))

		Expect(q.Len()).To(Equal(3))
	})
	It("TestFrontBackOutOfRangePanics", func() {
		const msg = "should panic when peeking empty queue"
		var q deque.Deque[int]
		Expect(func() { q.Front() }).To(Panic(), msg)
		Expect(func() { q.Back() }).To(Panic(), msg)

		q.PushBack(1)
		q.PopFront()

		Expect(func() { q.Front() }).To(Panic(), msg)
		Expect(func() { q.Back() }).To(Panic(), msg)
	})
	It("TestPopFrontOutOfRangePanics", func() {
		var q deque.Deque[int]

		Expect(func() { q.PopFront() }).To(Panic(), "should panic when popping empty queue")

		q.PushBack(1)
		q.PopFront()

		Expect(func() { q.PopFront() }).To(Panic(), "should panic when popping empty queue")
	})
	It("TestPopBackOutOfRangePanics", func() {
		var q deque.Deque[int]

		Expect(func() { q.PopBack() }).To(Panic(), "should panic when popping empty queue")

		q.PushBack(1)
		q.PopBack()

		Expect(func() { q.PopBack() }).To(Panic(), "should panic when popping empty queue")
	})
	It("TestAtOutOfRangePanics", func() {
		var q deque.Deque[int]

		q.PushBack(1)
		q.PushBack(2)
		q.PushBack(3)

		Expect(func() { q.At(-4) }).To(Panic(), "should panic when negative index")

		Expect(func() { q.At(4) }).To(Panic(), "should panic when index greater than length")
	})
	It("TestSetOutOfRangePanics", func() {
		var q deque.Deque[int]

		q.PushBack(1)
		q.PushBack(2)
		q.PushBack(3)

		Expect(func() { q.Set(-4, 1) }).To(Panic(), "should panic when negative index")

		Expect(func() { q.Set(4, 1) }).To(Panic(), "should panic when index greater than length")
	})
	It("TestInsertOutOfRangePanics", func() {
		q := new(deque.Deque[string])

		Expect(func() { q.Insert(-1, "X") }).To(Panic(), "should panic when inserting at negative index")

		q.PushBack("A")

		Expect(func() { q.Insert(-1, "Y") }).To(Panic())

		Expect(func() { q.Insert(2, "B") }).To(Panic(), "should panic when inserting out of range")
	})
	It("TestRemoveOutOfRangePanics", func() {
		q := new(deque.Deque[string])

		Expect(func() { q.Remove(0) }).To(Panic(), "should panic when removing from empty queue")

		q.PushBack("A")

		Expect(func() { q.Remove(-1) }).To(Panic(), "should panic when removing at negative index")

		Expect(func() { q.Remove(1) }).To(Panic(), "should panic when removing out of range")
	})
})
