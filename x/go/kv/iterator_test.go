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
	"encoding/binary"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	kvx "github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/kv/memkv"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Iterator", func() {
	var (
		kv kvx.DB
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

			iter := MustSucceed(kv.OpenIterator(kvx.IterPrefix([]byte("a"))))
			c := 0
			for iter.First(); iter.Valid(); iter.Next() {
				c++
			}
			Expect(c).To(Equal(3))
			Expect(iter.Close()).To(Succeed())
		})
	})
	Describe("Bounds Iterate", func() {
		It("Should iterate over keys in a given range", func() {
			for i := 0; i < 10; i++ {
				b := make([]byte, 4)
				binary.LittleEndian.PutUint32(b, uint32(i))
				Expect(kv.Set(ctx, b, []byte{1, 2})).To(Succeed())
			}
			lower := make([]byte, 4)
			binary.LittleEndian.PutUint32(lower, uint32(3))
			upper := make([]byte, 4)
			binary.LittleEndian.PutUint32(upper, uint32(7))
			iter := MustSucceed(kv.OpenIterator(kvx.IterRange(lower, upper)))
			c := 0
			for iter.First(); iter.Valid(); iter.Next() {
				c++
			}
			Expect(c).To(Equal(4))
			Expect(iter.Close()).To(Succeed())
		})
	})
})
