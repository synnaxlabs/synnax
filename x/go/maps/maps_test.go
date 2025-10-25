// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package maps_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/synnaxlabs/x/maps"
)

var _ = Describe("Ordered", func() {
	Describe("Count", func() {
		It("should return 0 for an empty map", func() {
			m := &maps.Ordered[string, int]{}
			Expect(m.Count()).To(Equal(0))
		})

		It("should return the correct count after adding elements", func() {
			m := &maps.Ordered[string, int]{}
			m.Put("first", 1)
			m.Put("second", 2)
			m.Put("third", 3)
			Expect(m.Count()).To(Equal(3))
		})
	})

	Describe("At", func() {
		It("should return the key-value pair at the given index", func() {
			m := &maps.Ordered[string, int]{
				Keys:   []string{"first", "second", "third"},
				Values: []int{1, 2, 3},
			}
			k, v := m.At(1)
			Expect(k).To(Equal("second"))
			Expect(v).To(Equal(2))
		})

		It("should return the first element at index 0", func() {
			m := &maps.Ordered[string, int]{
				Keys:   []string{"first", "second"},
				Values: []int{1, 2},
			}
			k, v := m.At(0)
			Expect(k).To(Equal("first"))
			Expect(v).To(Equal(1))
		})

		It("should return the last element at the last index", func() {
			m := &maps.Ordered[string, int]{
				Keys:   []string{"first", "second", "third"},
				Values: []int{1, 2, 3},
			}
			k, v := m.At(2)
			Expect(k).To(Equal("third"))
			Expect(v).To(Equal(3))
		})
	})

	Describe("Iter", func() {
		It("should iterate over all key-value pairs in order", func() {
			m := &maps.Ordered[string, int]{
				Keys:   []string{"first", "second", "third"},
				Values: []int{1, 2, 3},
			}

			var keys []string
			var values []int
			for k, v := range m.Iter() {
				keys = append(keys, k)
				values = append(values, v)
			}

			Expect(keys).To(Equal([]string{"first", "second", "third"}))
			Expect(values).To(Equal([]int{1, 2, 3}))
		})

		It("should handle empty map", func() {
			m := &maps.Ordered[string, int]{}

			count := 0
			for range m.Iter() {
				count++
			}
			Expect(count).To(Equal(0))
		})

		It("should support early termination", func() {
			m := &maps.Ordered[string, int]{
				Keys:   []string{"first", "second", "third", "fourth"},
				Values: []int{1, 2, 3, 4},
			}

			var keys []string
			for k, v := range m.Iter() {
				keys = append(keys, k)
				if v == 2 {
					break
				}
			}

			Expect(keys).To(Equal([]string{"first", "second"}))
		})
	})

	Describe("Get", func() {
		It("should return the value and true for an existing key", func() {
			m := &maps.Ordered[string, int]{
				Keys:   []string{"first", "second", "third"},
				Values: []int{1, 2, 3},
			}
			v, ok := m.Get("second")
			Expect(ok).To(BeTrue())
			Expect(v).To(Equal(2))
		})

		It("should return zero value and false for non-existing key", func() {
			m := &maps.Ordered[string, int]{
				Keys:   []string{"first", "second"},
				Values: []int{1, 2},
			}
			v, ok := m.Get("third")
			Expect(ok).To(BeFalse())
			Expect(v).To(Equal(0))
		})

		It("should handle empty map", func() {
			m := &maps.Ordered[string, int]{}
			v, ok := m.Get("any")
			Expect(ok).To(BeFalse())
			Expect(v).To(Equal(0))
		})

		It("should return the first occurrence for duplicate keys", func() {
			m := &maps.Ordered[string, int]{
				Keys:   []string{"key", "other", "key"},
				Values: []int{1, 2, 3},
			}
			v, ok := m.Get("key")
			Expect(ok).To(BeTrue())
			Expect(v).To(Equal(1))
		})

		It("should work with different types", func() {
			m := &maps.Ordered[int, string]{
				Keys:   []int{10, 20, 30},
				Values: []string{"ten", "twenty", "thirty"},
			}
			v, ok := m.Get(20)
			Expect(ok).To(BeTrue())
			Expect(v).To(Equal("twenty"))
		})
	})

	Describe("Put", func() {
		It("should add a new key-value pair and return true", func() {
			m := &maps.Ordered[string, int]{}
			ok := m.Put("first", 1)
			Expect(ok).To(BeTrue())
			Expect(m.Keys).To(Equal([]string{"first"}))
			Expect(m.Values).To(Equal([]int{1}))
		})

		It("should not add duplicate key and return false", func() {
			m := &maps.Ordered[string, int]{
				Keys:   []string{"first"},
				Values: []int{1},
			}
			ok := m.Put("first", 2)
			Expect(ok).To(BeFalse())
			Expect(m.Keys).To(Equal([]string{"first"}))
			Expect(m.Values).To(Equal([]int{1}))
		})

		It("should maintain insertion order", func() {
			m := &maps.Ordered[string, int]{}
			m.Put("third", 3)
			m.Put("first", 1)
			m.Put("second", 2)
			Expect(m.Keys).To(Equal([]string{"third", "first", "second"}))
			Expect(m.Values).To(Equal([]int{3, 1, 2}))
		})

		It("should work with multiple values", func() {
			m := &maps.Ordered[string, int]{}
			for i := 0; i < 5; i++ {
				key := string(rune('a' + i))
				ok := m.Put(key, i)
				Expect(ok).To(BeTrue())
			}
			Expect(m.Count()).To(Equal(5))
			Expect(m.Keys).To(Equal([]string{"a", "b", "c", "d", "e"}))
			Expect(m.Values).To(Equal([]int{0, 1, 2, 3, 4}))
		})

		It("should work with different types", func() {
			m := &maps.Ordered[int, string]{}
			ok := m.Put(42, "answer")
			Expect(ok).To(BeTrue())
			Expect(m.Keys).To(Equal([]int{42}))
			Expect(m.Values).To(Equal([]string{"answer"}))
		})
	})

	Describe("Copy", func() {
		It("Should create an independent copy of the map", func() {
			m := &maps.Ordered[string, int]{
				Keys:   []string{"first", "second", "third"},
				Values: []int{1, 2, 3},
			}
			cpy := m.Copy()

			Expect(cpy).NotTo(BeNil())
			Expect(cpy.Keys).To(Equal([]string{"first", "second", "third"}))
			Expect(cpy.Values).To(Equal([]int{1, 2, 3}))
			Expect(cpy.Count()).To(Equal(3))
		})

		It("Should return nil for nil receiver", func() {
			var m *maps.Ordered[string, int]
			cpy := m.Copy()
			Expect(cpy).To(BeNil())
		})

		It("Should create independent slices", func() {
			m := &maps.Ordered[string, int]{
				Keys:   []string{"first", "second"},
				Values: []int{1, 2},
			}
			cpy := m.Copy()

			// Modify the cpy
			cpy.Put("third", 3)

			// Original should not be affected
			Expect(m.Count()).To(Equal(2))
			Expect(m.Keys).To(Equal([]string{"first", "second"}))
			Expect(m.Values).To(Equal([]int{1, 2}))

			// Copy should have the new element
			Expect(cpy.Count()).To(Equal(3))
			Expect(cpy.Keys).To(Equal([]string{"first", "second", "third"}))
			Expect(cpy.Values).To(Equal([]int{1, 2, 3}))
		})

		It("Should copy empty map", func() {
			m := &maps.Ordered[string, int]{}
			cpy := m.Copy()

			Expect(cpy).NotTo(BeNil())
			Expect(cpy.Count()).To(Equal(0))
			Expect(cpy.Keys).To(BeEmpty())
			Expect(cpy.Values).To(BeEmpty())
		})

		It("Should work with different types", func() {
			m := &maps.Ordered[int, string]{
				Keys:   []int{1, 2, 3},
				Values: []string{"one", "two", "three"},
			}
			cpy := m.Copy()

			Expect(cpy.Keys).To(Equal([]int{1, 2, 3}))
			Expect(cpy.Values).To(Equal([]string{"one", "two", "three"}))

			// Verify independence
			cpy.Put(4, "four")
			Expect(m.Count()).To(Equal(3))
			Expect(cpy.Count()).To(Equal(4))
		})

		It("Should create shallow copy with pointer values", func() {
			type Data struct {
				Value int
			}

			m := &maps.Ordered[string, *Data]{
				Keys:   []string{"key1"},
				Values: []*Data{{Value: 42}},
			}
			cpy := m.Copy()

			// Keys and Values slices are independent
			Expect(cpy.Keys).To(Equal([]string{"key1"}))
			Expect(&cpy.Keys).NotTo(BeIdenticalTo(&m.Keys))
			Expect(&cpy.Values).NotTo(BeIdenticalTo(&m.Values))

			// But the pointer values point to the same data (shallow cpy)
			Expect(cpy.Values[0]).To(Equal(m.Values[0]))
			cpy.Values[0].Value = 99
			Expect(m.Values[0].Value).To(Equal(99))
		})

		It("Should maintain insertion order in copy", func() {
			m := &maps.Ordered[string, int]{}
			m.Put("z", 26)
			m.Put("a", 1)
			m.Put("m", 13)

			cpy := m.Copy()

			// Verify order is maintained
			k0, v0 := cpy.At(0)
			Expect(k0).To(Equal("z"))
			Expect(v0).To(Equal(26))

			k1, v1 := cpy.At(1)
			Expect(k1).To(Equal("a"))
			Expect(v1).To(Equal(1))

			k2, v2 := cpy.At(2)
			Expect(k2).To(Equal("m"))
			Expect(v2).To(Equal(13))
		})

		It("Should allow iteration over copy", func() {
			m := &maps.Ordered[string, int]{
				Keys:   []string{"a", "b", "c"},
				Values: []int{1, 2, 3},
			}
			cpy := m.Copy()

			var keys []string
			var sum int
			for k, v := range cpy.Iter() {
				keys = append(keys, k)
				sum += v
			}

			Expect(keys).To(Equal([]string{"a", "b", "c"}))
			Expect(sum).To(Equal(6))
		})
	})

	Describe("Integration tests", func() {
		It("should handle a complete workflow", func() {
			m := &maps.Ordered[string, int]{}

			// Add some elements
			Expect(m.Put("apple", 5)).To(BeTrue())
			Expect(m.Put("banana", 3)).To(BeTrue())
			Expect(m.Put("cherry", 8)).To(BeTrue())

			// Check count
			Expect(m.Count()).To(Equal(3))

			// Try to add duplicate
			Expect(m.Put("banana", 10)).To(BeFalse())
			Expect(m.Count()).To(Equal(3))

			// Get values
			v, ok := m.Get("banana")
			Expect(ok).To(BeTrue())
			Expect(v).To(Equal(3))

			v, ok = m.Get("orange")
			Expect(ok).To(BeFalse())
			Expect(v).To(Equal(0))

			// Access by index
			k, v := m.At(0)
			Expect(k).To(Equal("apple"))
			Expect(v).To(Equal(5))

			k, v = m.At(2)
			Expect(k).To(Equal("cherry"))
			Expect(v).To(Equal(8))

			// Iterate
			var keys []string
			var sum int
			for k, v := range m.Iter() {
				keys = append(keys, k)
				sum += v
			}
			Expect(keys).To(Equal([]string{"apple", "banana", "cherry"}))
			Expect(sum).To(Equal(16))
		})

		It("should handle struct values", func() {
			type Person struct {
				Name string
				Age  int
			}

			m := &maps.Ordered[string, Person]{}

			m.Put("alice", Person{Name: "Alice", Age: 30})
			m.Put("bob", Person{Name: "Bob", Age: 25})

			Expect(m.Count()).To(Equal(2))

			p, ok := m.Get("alice")
			Expect(ok).To(BeTrue())
			Expect(p.Name).To(Equal("Alice"))
			Expect(p.Age).To(Equal(30))

			k, v := m.At(1)
			Expect(k).To(Equal("bob"))
			Expect(v.Name).To(Equal("Bob"))
		})
	})
})
