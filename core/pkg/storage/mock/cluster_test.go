// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package mock_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/synnax/pkg/storage/mock"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Storage", func() {
	ShouldNotLeakGoroutinesBeforeEach()
	DescribeTable("Name", func(cfg ...storage.Config) {
		b := mock.NewCluster(cfg...)
		store := b.Provision(ctx)
		Expect(store).NotTo(BeNil())
		Expect(store.KV.Set(ctx, []byte("foo"), []byte("bar"))).To(Succeed())
		Expect(b.Close()).To(Succeed())
	},
		Entry("Memory-backed storage implementation"),
		Entry("FS-backed storage implementation", storage.Config{InMemory: new(false), Dirname: "./tmp"}),
	)
})
