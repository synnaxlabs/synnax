// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package kv_test

import (
	"fmt"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/aspen/internal/cluster"
	"github.com/synnaxlabs/aspen/internal/kv"
	"github.com/synnaxlabs/aspen/internal/kv/kvmock"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("kvStore", func() {
	var (
		builder *kvmock.Builder
		db      *kv.DB
	)

	BeforeEach(func() {
		builder = kvmock.NewBuilder(
			kv.Config{GossipInterval: 10 * time.Millisecond},
			cluster.Config{},
		)
		db = MustSucceed(builder.New(ctx, kv.Config{}, cluster.Config{}))
	})

	AfterEach(func() {
		Expect(builder.Close()).To(Succeed())
	})

	It("should write and retrieve values", func() {
		Expect(db.Set(ctx, []byte("a"), []byte("1"))).To(Succeed())
		Expect(db.Set(ctx, []byte("b"), []byte("2"))).To(Succeed())
		v, closer := MustSucceed2(db.Get(ctx, []byte("a")))
		Expect(v).To(Equal([]byte("1")))
		Expect(closer.Close()).To(Succeed())
	})

	It("should overwrite existing entries", func() {
		Expect(db.Set(ctx, []byte("k"), []byte("first"))).To(Succeed())
		Expect(db.Set(ctx, []byte("k"), []byte("second"))).To(Succeed())
		v, closer := MustSucceed2(db.Get(ctx, []byte("k")))
		Expect(v).To(Equal([]byte("second")))
		Expect(closer.Close()).To(Succeed())
	})

	It("should support concurrent writes without data races", func() {
		errs := make([]error, 50)
		var wg sync.WaitGroup
		for i := range 50 {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				errs[i] = db.Set(ctx, fmt.Appendf(nil, "key%d", i), []byte("v"))
			}(i)
		}
		wg.Wait()
		for _, err := range errs {
			Expect(err).ToNot(HaveOccurred())
		}
	})
})
