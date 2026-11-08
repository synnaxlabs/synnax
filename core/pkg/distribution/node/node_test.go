// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package node_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
)

var _ = Describe("Key", func() {
	DescribeTable("Free", func(key node.Key, free bool) {
		Expect(key.Free()).To(Equal(free))
	},
		Entry("KeyFree", node.KeyFree, true),
		Entry("the host key", node.KeyBootstrapper, false),
		Entry("another key", node.Key(2), false),
	)

	DescribeTable("IsBootstrapper", func(key node.Key, bootstrapper bool) {
		Expect(key.IsBootstrapper()).To(Equal(bootstrapper))
	},
		Entry("the host key", node.KeyBootstrapper, true),
		Entry("KeyFree", node.KeyFree, false),
		Entry("another key", node.Key(2), false),
	)

	It("Should format the key as a decimal number", func() {
		Expect(node.Key(42).String()).To(Equal("42"))
		Expect(node.KeyFree.String()).To(Equal("4095"))
	})
})
