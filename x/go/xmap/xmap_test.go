// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package xmap_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/synnaxlabs/x/xmap"
)

var _ = Describe("XMap", func() {
	var m xmap.Map[string, int]
	BeforeEach(func() {
		m = xmap.Map[string, int]{}
	})
	Describe("Get", func() {
		It("Should return the value associated with the given key", func() {
			m["foo"] = 1
			v, ok := m.Get("foo")
			Expect(v).To(Equal(1))
			Expect(ok).To(BeTrue())
		})
		It("Should return the zero value of V if the key is not found", func() {
			v, ok := m.Get("foo")
			Expect(v).To(Equal(0))
			Expect(ok).To(BeFalse())
		})
	})
	Describe("GetDefault", func() {
		It("Should return the value associated with the given key", func() {
			m["foo"] = 1
			v := m.GetDefault("foo", 0)
			Expect(v).To(Equal(1))
		})
		It("Should return the fallback value if the key is not found", func() {
			v := m.GetDefault("foo", 11)
			Expect(v).To(Equal(11))
		})
	})
	Describe("Contains", func() {
		It("Should return true if the map contains the given key", func() {
			m["foo"] = 1
			Expect(m.Contains("foo")).To(BeTrue())
		})
		It("Should return false if the map does not contain the given key", func() {
			Expect(m.Contains("foo")).To(BeFalse())
		})
	})
	Describe("Set", func() {
		It("Should set the value associated with the given key", func() {
			m.Set("foo", 1)
			Expect(m["foo"]).To(Equal(1))
		})
	})
	Describe("Delete", func() {
		It("Should delete the value associated with the given key", func() {
			m["foo"] = 1
			m.Delete("foo")
			v := m["foo"]
			Expect(v).To(Equal(0))
		})
	})
	Describe("Keys", func() {
		It("Should return a slice of all keys in the map", func() {
			m["foo"] = 1
			m["bar"] = 2
			keys := m.Keys()
			Expect(keys).To(ConsistOf("foo", "bar"))
		})
		It("Should return an empty slice if the map is empty", func() {
			keys := m.Keys()
			Expect(keys).To(BeEmpty())
		})
	})
	Describe("Values", func() {
		It("Should return a slice of all values in the map", func() {
			m["foo"] = 1
			m["bar"] = 2
			values := m.Values()
			Expect(values).To(ConsistOf(1, 2))
		})
		It("Should return an empty slice if the map is empty", func() {
			values := m.Values()
			Expect(values).To(BeEmpty())
		})
		It("should return repeated values", func() {
			m["foo"] = 1
			m["bar"] = 1
			values := m.Values()
			Expect(values).To(ConsistOf(1, 1))
		})
	})
	Describe("Copy", func() {
		It("should return a copy of the map", func() {
			m["foo"] = 1
			m["bar"] = 2
			copy := m.Copy()
			Expect(copy).To(Equal(m))
		})
		It("should not modify the original map when the copy is modified", func() {
			m["foo"] = 1
			m["bar"] = 2
			copy := m.Copy()
			copy["foo"] = 3
			Expect(m["foo"]).To(Equal(1))
		})
	})
})
