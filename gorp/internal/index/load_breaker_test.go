// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package index_test

import (
	"cmp"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp/internal/index"
)

var _ = Describe("Load breaker", func() {
	engines := map[string]func() engine{
		"Lookup": func() engine { return index.NewLookup[*scope, int, string]() },
		"Sorted": func() engine {
			return index.NewSorted[*scope, int, string](cmp.Compare[int])
		},
		"Multi": func() engine {
			return singleValue{index.NewMulti[*scope, int, string]()}
		},
	}
	for name, newEngine := range engines {
		Describe(name, func() {
			It("Should delete a key that was set before the load", func() {
				e := newEngine()
				e.Set(1, "a")
				insert, finish := e.Populate()
				insert(2, "b")
				finish(nil)
				Expect(e.Get(nil, "a")).To(ConsistOf(1))
				e.Delete(1)
				Expect(e.Get(nil, "a")).To(BeEmpty())
			})

			It("Should move a key that a scope committed before the load", func() {
				e := newEngine()
				s := &scope{}
				e.Stage(s, 1, "a")
				s.end(true)
				insert, finish := e.Populate()
				insert(2, "b")
				finish(nil)
				e.Set(1, "c")
				Expect(e.Get(nil, "a")).To(BeEmpty())
				Expect(e.Get(nil, "c")).To(ConsistOf(1))
			})
		})
	}
})
