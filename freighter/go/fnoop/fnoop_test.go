// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package fnoop_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter/fnoop"
)

var _ = Describe("Fnoop", func() {
	Describe("Unary", func() {
		It("should succeed on calls to UnaryClient.Send", func() {
			var client fnoop.UnaryClient[any, any]
			Expect(client.Send(context.Background(), "", nil)).To(Succeed())
		})
	})
	Describe("Stream", func() {
		It("should succeed on calls to StreamClient.Stream", func() {
			var client fnoop.StreamClient[any, any]
			Expect(client.Stream(context.Background(), "")).To(Succeed())
		})
	})
})
