// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package kv_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	xkv "github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/kv/memkv"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Iterator", func() {
	var (
		kv xkv.DB
	)
	BeforeEach(func() {
		kv = memkv.New()
	})
	AfterEach(func() {
		Expect(kv.Close()).To(Succeed())
	})
	Describe("IterPrefix", func() {
		It("Should iterate over keys with a given prefix", func() {
			Expect(kv.Set(ctx, []byte("a/foo"), []byte("bar"))).To(Succeed())
			Expect(kv.Set(ctx, []byte("a/baz"), []byte("qux"))).To(Succeed())
			Expect(kv.Set(ctx, []byte("a/qux"), []byte("quux"))).To(Succeed())
			Expect(kv.Set(ctx, []byte("b/foo"), []byte("bar"))).To(Succeed())
			Expect(kv.Set(ctx, []byte("b/baz"), []byte("qux"))).To(Succeed())

			iter := MustSucceed(kv.OpenIterator(xkv.IterPrefix([]byte("a"))))
			c := 0
			for iter.First(); iter.Valid(); iter.Next() {
				c++
			}
			Expect(c).To(Equal(3))
			Expect(iter.Close()).To(Succeed())
		})
	})
})
