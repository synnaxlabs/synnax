// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package content_test

import (
	"math"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/content"
)

func sum(write func(h *content.Hash)) string {
	h := content.NewHash()
	write(&h)
	return h.Sum()
}

var _ = Describe("Hash", func() {
	It("Should sum to 16 lowercase hexadecimal characters", func() {
		Expect(sum(func(h *content.Hash) { h.String("a") })).
			To(MatchRegexp(`^[0-9a-f]{16}$`))
	})

	It("Should sum the same writes equally", func() {
		write := func(h *content.Hash) {
			h.String("a")
			h.Bool(true)
			h.Int(-3)
			h.Uint(4)
			h.Float(1.5)
			h.Bytes([]byte{1, 2})
		}
		Expect(sum(write)).To(Equal(sum(write)))
	})

	It("Should sum an empty hash as the hash of no input", func() {
		Expect(sum(func(*content.Hash) {})).To(Equal("ef46db3751d8e999"))
	})

	DescribeTable("Should sum different writes differently",
		func(a, b func(h *content.Hash)) {
			Expect(sum(a)).ToNot(Equal(sum(b)))
		},
		Entry("strings split at another point",
			func(h *content.Hash) { h.String("ab"); h.String("c") },
			func(h *content.Hash) { h.String("a"); h.String("bc") },
		),
		Entry("booleans",
			func(h *content.Hash) { h.Bool(true) },
			func(h *content.Hash) { h.Bool(false) },
		),
		Entry("signed zeros",
			func(h *content.Hash) { h.Float(0) },
			func(h *content.Hash) { h.Float(math.Copysign(0, -1)) },
		),
		Entry("list lengths",
			func(h *content.Hash) { h.Len(1); h.String("a") },
			func(h *content.Hash) { h.Len(2); h.String("a") },
		),
	)

	Describe("Any", func() {
		It("Should hash an object without its key members, at any depth", func() {
			Expect(sum(func(h *content.Hash) {
				h.Any(map[string]any{
					"key": "a",
					"v":   []any{map[string]any{"key": "b", "x": 1.0}},
				})
			})).To(Equal(sum(func(h *content.Hash) {
				h.Any(map[string]any{
					"key": "c",
					"v":   []any{map[string]any{"key": "d", "x": 1.0}},
				})
			})))
		})

		It("Should hash an object by its members, in any order", func() {
			a := map[string]any{"x": 1.0, "y": "a"}
			b := map[string]any{"y": "a", "x": 1.0}
			Expect(sum(func(h *content.Hash) { h.Any(a) })).
				To(Equal(sum(func(h *content.Hash) { h.Any(b) })))
		})

		DescribeTable("Should hash different values differently",
			func(a, b any) {
				Expect(sum(func(h *content.Hash) { h.Any(a) })).
					ToNot(Equal(sum(func(h *content.Hash) { h.Any(b) })))
			},
			Entry("null and false", nil, false),
			Entry("a number and its string", 1.0, "1"),
			Entry("an empty list and an empty object", []any{}, map[string]any{}),
			Entry("lists in another order", []any{"a", "b"}, []any{"b", "a"}),
			Entry("a member name and its value",
				map[string]any{"a": "b"},
				map[string]any{"b": "a"},
			),
		)

		It("Should panic for a value that JSON cannot hold", func() {
			Expect(func() {
				h := content.NewHash()
				h.Any(int64(1))
			}).To(PanicWith("content: a JSON value cannot hold int64"))
		})
	})
})
