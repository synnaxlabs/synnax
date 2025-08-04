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
	Describe("Get", func() {
		It("Should return the value associated with the given key", func() {
			m := xmap.Map[string, int]{"foo": 1}
			v, ok := m.Get("foo")
			Expect(v).To(Equal(1))
			Expect(ok).To(BeTrue())
		})
		It("Should return the zero value of V if the key is not found", func() {
			m := xmap.Map[string, int]{}
			v, ok := m.Get("foo")
			Expect(v).To(Equal(0))
			Expect(ok).To(BeFalse())
		})
	})
	Describe("Set", func() {
		It("Should set the value associated with the given key", func() {
			m := xmap.Map[string, int]{}
			v, ok := m.Get("foo")
			Expect(v).To(Equal(0))
			Expect(ok).To(BeFalse())
			m.Set("foo", 1)
			v, ok = m.Get("foo")
			Expect(v).To(Equal(1))
			Expect(ok).To(BeTrue())
		})
	})
	Describe("Delete", func() {
		It("Should delete the value associated with the given key", func() {
			m := xmap.Map[string, int]{"foo": 1}
			v, ok := m.Get("foo")
			Expect(v).To(Equal(1))
			Expect(ok).To(BeTrue())
			m.Delete("foo")
			v, ok = m.Get("foo")
			Expect(v).To(Equal(0))
			Expect(ok).To(BeFalse())
		})
	})
	Describe("Keys", func() {
		It("Should return a slice of all keys in the map", func() {
			m := xmap.Map[string, int]{"foo": 1, "bar": 2}
			keys := m.Keys()
			Expect(keys).To(ConsistOf("foo", "bar"))
		})
		It("Should return an empty slice if the map is empty", func() {
			m := xmap.Map[string, int]{}
			keys := m.Keys()
			Expect(keys).To(BeEmpty())
		})
		It("shouldn't return repeated keys", func() {
			m := xmap.Map[string, int]{"foo": 1}
			m.Set("foo", 2)
			keys := m.Keys()
			Expect(keys).To(HaveLen(1))
			Expect(keys).To(ConsistOf("foo"))
		})
	})
	Describe("Values", func() {
		It("Should return a slice of all values in the map", func() {
			m := xmap.Map[string, int]{"foo": 1, "bar": 2}
			values := m.Values()
			Expect(values).To(ConsistOf(1, 2))
		})
		It("Should return an empty slice if the map is empty", func() {
			m := xmap.Map[string, int]{}
			values := m.Values()
			Expect(values).To(BeEmpty())
		})
		It("shouldn't return repeated keys", func() {
			m := xmap.Map[string, int]{"foo": 1}
			m.Set("foo", 3)
			values := m.Values()
			Expect(values).To(HaveLen(1))
			Expect(values).To(ConsistOf(3))
		})
		It("should return repeated values", func() {
			m := xmap.Map[string, int]{"foo": 1, "bar": 1}
			values := m.Values()
			Expect(values).To(HaveLen(2))
			Expect(values).To(ConsistOf(1, 1))
		})
	})
})
