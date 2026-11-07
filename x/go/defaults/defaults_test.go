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

var _ = Describe("Fill", func() {
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
