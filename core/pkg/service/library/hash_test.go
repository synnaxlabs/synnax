// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package library_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/library"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Hash", func() {
	It("Should return 16 lowercase hex characters", func() {
		Expect(MustSucceed(library.Hash("a"))).To(MatchRegexp("^[0-9a-f]{16}$"))
	})

	It("Should hash equal values equally", func() {
		Expect(MustSucceed(library.Hash([]int{1, 2}))).
			To(Equal(MustSucceed(library.Hash([]int{1, 2}))))
	})

	It("Should hash a map the same whatever its iteration order", func() {
		a := map[string]int{}
		b := map[string]int{}
		for i, k := range []string{"a", "b", "c", "d", "e", "f", "g", "h"} {
			a[k] = i
		}
		for i, k := range []string{"h", "g", "f", "e", "d", "c", "b", "a"} {
			b[k] = 7 - i
		}
		Expect(MustSucceed(library.Hash(a))).To(Equal(MustSucceed(library.Hash(b))))
	})

	It("Should hash different values differently", func() {
		Expect(MustSucceed(library.Hash([]int{1, 2}))).
			ToNot(Equal(MustSucceed(library.Hash([]int{2, 1}))))
	})

	It("Should return an error for a value JSON cannot encode", func() {
		Expect(library.Hash(make(chan int))).Error().
			To(MatchError(ContainSubstring("failed to hash library content")))
	})
})
