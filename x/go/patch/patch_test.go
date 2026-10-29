// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package patch_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/patch"
	"github.com/synnaxlabs/x/validate"
)

type port struct {
	Key  string `json:"key"`
	Port uint8  `json:"port"`
}

var _ = Describe("Patch", func() {
	Describe("Set", func() {
		It("Should store values in the form JSON decodes into", func() {
			doc := map[string]any{}
			Expect(patch.Set{Path: []string{"a"}, Value: int8(3)}.Apply(doc)).
				To(Succeed())
			Expect(patch.Set{
				Path:  []string{"b"},
				Value: map[string]any{"c": uint64(1)},
			}.Apply(doc)).To(Succeed())
			Expect(doc).To(Equal(map[string]any{
				"a": float64(3),
				"b": map[string]any{"c": float64(1)},
			}))
		})

		It("Should leave the document unchanged when the action fails", func() {
			doc := map[string]any{"a": map[string]any{"b": float64(1)}}
			Expect(patch.Set{Path: []string{"a", "b", "c"}, Value: 1}.Apply(doc)).
				To(MatchError(validate.ErrValidation))
			Expect(doc).To(Equal(map[string]any{"a": map[string]any{"b": float64(1)}}))
		})

		It("Should reject a value that JSON cannot encode", func() {
			doc := map[string]any{}
			Expect(patch.Set{Path: []string{"a"}, Value: make(chan int)}.Apply(doc)).
				To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring("encoding value")),
				))
			Expect(doc).To(BeEmpty())
		})
	})

	Describe("Insert", func() {
		It("Should accept a struct entry with a key field", func() {
			doc := map[string]any{"ports": []any{}}
			Expect(patch.Insert{
				Path:  []string{"ports"},
				Value: port{Key: "a", Port: 2},
			}.Apply(doc)).To(Succeed())
			Expect(doc).To(Equal(map[string]any{
				"ports": []any{map[string]any{"key": "a", "port": float64(2)}},
			}))
		})
	})

	Describe("Remove", func() {
		It("Should remove every entry with the key", func() {
			doc := map[string]any{"l": []any{
				map[string]any{"key": "a"},
				map[string]any{"key": "b"},
				map[string]any{"key": "a"},
			}}
			Expect(patch.Remove{Path: []string{"l"}, Key: "a"}.Apply(doc)).
				To(Succeed())
			Expect(doc).To(Equal(map[string]any{
				"l": []any{map[string]any{"key": "b"}},
			}))
		})
	})

	Describe("Move", func() {
		It("Should keep the entries it does not move in order", func() {
			doc := map[string]any{"l": []any{
				map[string]any{"key": "a"},
				map[string]any{"key": "b"},
				map[string]any{"key": "c"},
				map[string]any{"key": "d"},
			}}
			after := "c"
			Expect(patch.Move{Path: []string{"l"}, Key: "b", After: &after}.Apply(doc)).
				To(Succeed())
			Expect(doc).To(Equal(map[string]any{"l": []any{
				map[string]any{"key": "a"},
				map[string]any{"key": "c"},
				map[string]any{"key": "b"},
				map[string]any{"key": "d"},
			}}))
		})
	})
})
