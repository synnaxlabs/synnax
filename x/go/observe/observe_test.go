// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package observe_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/observe"
)

var _ = Describe("Observer", func() {
	It("Should notify all handlers", func(ctx SpecContext) {
		obs := observe.New[int]()
		var results []int
		obs.OnChange(func(ctx context.Context, v int) {
			results = append(results, v)
		})
		obs.OnChange(func(ctx context.Context, v int) {
			results = append(results, v*2)
		})
		obs.Notify(ctx, 5)
		Expect(results).To(ContainElements(5, 10))
	})

	It("Should allow disconnecting handlers", func(ctx SpecContext) {
		obs := observe.New[int]()
		called := false
		disconnect := obs.OnChange(func(ctx context.Context, v int) {
			called = true
		})
		disconnect()
		obs.Notify(ctx, 5)
		Expect(called).To(BeFalse())
	})
})

var _ = Describe("Translator", func() {
	It("Should translate and notify when Translate returns true", func(ctx SpecContext) {
		base := observe.New[int]()
		translator := observe.Translator[int, string]{
			Observable: base,
			Translate: func(ctx context.Context, v int) (string, bool) {
				return "translated", true
			},
		}
		var result string
		translator.OnChange(func(ctx context.Context, v string) {
			result = v
		})
		base.Notify(ctx, 42)
		Expect(result).To(Equal("translated"))
	})

	It("Should not notify when Translate returns false", func(ctx SpecContext) {
		base := observe.New[int]()
		translator := observe.Translator[int, string]{
			Observable: base,
			Translate: func(ctx context.Context, v int) (string, bool) {
				return "", false
			},
		}
		called := false
		translator.OnChange(func(ctx context.Context, v string) {
			called = true
		})
		base.Notify(ctx, 42)
		Expect(called).To(BeFalse())
	})

	It("Should conditionally notify based on input", func(ctx SpecContext) {
		base := observe.New[int]()
		translator := observe.Translator[int, int]{
			Observable: base,
			Translate: func(ctx context.Context, v int) (int, bool) {
				if v > 10 {
					return v * 2, true
				}
				return 0, false
			},
		}
		var results []int
		translator.OnChange(func(ctx context.Context, v int) {
			results = append(results, v)
		})
		base.Notify(ctx, 5)
		base.Notify(ctx, 15)
		base.Notify(ctx, 3)
		base.Notify(ctx, 20)
		Expect(results).To(Equal([]int{30, 40}))
	})
})

var _ = Describe("Noop", func() {
	It("Should not call handlers", func() {
		var noop observe.Noop[int]
		called := false
		noop.OnChange(func(ctx context.Context, v int) {
			called = true
		})
		Expect(called).To(BeFalse())
	})
})
