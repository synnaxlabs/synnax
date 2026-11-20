// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package defaults_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/defaults"
)

type leaf struct {
	Size  int
	Label string
}

func (l *leaf) SetDefaults() { l.Size = 36 }

type tree struct {
	Name   string
	Inner  leaf
	Leaves []leaf
	ByKey  map[string]leaf
	Maybe  *leaf
}

func (t *tree) SetDefaults() {
	t.Name = "untitled"
	t.Inner.SetDefaults()
	t.Inner.Label = "inner"
}

// bounds mirrors a generated type with a default group: the pair takes its
// defaults only when both members are zero.
type bounds struct {
	Min   float64
	Max   float64
	Units string
}

func (b *bounds) SetDefaults() {
	b.SetUngroupedDefaults()
	b.DefaultGroups()
}

func (b *bounds) SetUngroupedDefaults() { b.Units = "volts" }

func (b *bounds) DefaultGroups() {
	if b.Min == 0 && b.Max == 0 {
		b.Max = 1
	}
}

var _ = Describe("Fill", func() {
	DescribeTable("Should fill a default group only when all of its members are zero",
		func(in, expected bounds) {
			Expect(defaults.Fill(in)).To(Equal(expected))
		},
		Entry("all zero", bounds{}, bounds{Max: 1, Units: "volts"}),
		Entry("one member set", bounds{Min: -10}, bounds{Min: -10, Units: "volts"}),
		Entry(
			"both members set",
			bounds{Min: -10, Max: 10},
			bounds{Min: -10, Max: 10, Units: "volts"},
		),
	)

	It("Should fill a default group in a slice element", func() {
		Expect(defaults.Fill([]bounds{{Min: -10}, {}})).To(Equal([]bounds{
			{Min: -10, Units: "volts"},
			{Max: 1, Units: "volts"},
		}))
	})

	It("Should start an empty value from its defaults", func() {
		Expect(defaults.Fill(tree{})).To(Equal(tree{
			Name:  "untitled",
			Inner: leaf{Size: 36, Label: "inner"},
		}))
	})

	It("Should keep every non-zero field over the defaults", func() {
		Expect(defaults.Fill(tree{Name: "custom", Inner: leaf{Size: 2}})).
			To(Equal(tree{Name: "custom", Inner: leaf{Size: 2, Label: "inner"}}))
	})

	It("Should default the elements of slices, maps, and pointers", func() {
		got := defaults.Fill(tree{
			Leaves: []leaf{{Label: "a"}},
			ByKey:  map[string]leaf{"k": {}},
			Maybe:  &leaf{Label: "b"},
		})
		Expect(got.Leaves).To(Equal([]leaf{{Size: 36, Label: "a"}}))
		Expect(got.ByKey).To(Equal(map[string]leaf{"k": {Size: 36}}))
		Expect(got.Maybe).To(Equal(&leaf{Size: 36, Label: "b"}))
	})

	It("Should default a nested struct of a type without SetDefaults", func() {
		type outer struct {
			Name  string
			Inner leaf
		}
		Expect(defaults.Fill(outer{Inner: leaf{Label: "d"}})).
			To(Equal(outer{Inner: leaf{Size: 36, Label: "d"}}))
		Expect(defaults.Fill(outer{})).To(Equal(outer{Inner: leaf{Size: 36}}))
	})

	It("Should keep a value whose type has unexported fields", func() {
		at := time.Now()
		type stamped struct {
			At      time.Time
			Variant any
		}
		Expect(defaults.Fill(stamped{At: at, Variant: at})).
			To(Equal(stamped{At: at, Variant: at}))
	})

	It("Should default the value a union holds", func() {
		type union struct{ Variant any }
		got := defaults.Fill(union{Variant: leaf{Label: "c"}})
		Expect(got.Variant).To(Equal(leaf{Size: 36, Label: "c"}))
	})

	It("Should return a nil interface unchanged", func() {
		Expect(defaults.Fill[any](nil)).To(BeNil())
	})

	It("Should return a value without SetDefaults unchanged", func() {
		type point struct{ X, Y int }
		Expect(defaults.Fill(point{X: 1})).To(Equal(point{X: 1}))
	})
})

var _ = Describe("Prefill", func() {
	It("Should set the defaults outside the default groups", func() {
		var b bounds
		defaults.Prefill(&b)
		Expect(b).To(Equal(bounds{Units: "volts"}))
	})

	It("Should set every default of a type without groups", func() {
		var l leaf
		defaults.Prefill(&l)
		Expect(l).To(Equal(leaf{Size: 36}))
	})

	It("Should leave a type without defaults as it is", func() {
		v := 3
		defaults.Prefill(&v)
		Expect(v).To(Equal(3))
	})
})

var _ = Describe("FillGroups", func() {
	DescribeTable("Should set a default group only when all of its members are zero",
		func(in, expected bounds) {
			defaults.FillGroups(&in)
			Expect(in).To(Equal(expected))
		},
		Entry("all zero", bounds{}, bounds{Max: 1}),
		Entry("one member set", bounds{Min: -10}, bounds{Min: -10}),
	)

	It("Should leave a type without groups as it is", func() {
		l := leaf{Label: "x"}
		defaults.FillGroups(&l)
		Expect(l).To(Equal(leaf{Label: "x"}))
	})
})
