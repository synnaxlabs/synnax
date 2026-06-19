// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package grpc_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/transport/grpc"
)

var _ = Describe("GRPC", func() {
	Describe("Bind", func() {
		It("Should return one bindable transport per gRPC service", func() {
			transports := grpc.Bind(apiLayer, apiLayer.Channel)
			Expect(transports).To(HaveLen(13))
			for _, t := range transports {
				Expect(t).ToNot(BeNil())
			}
		})

		It("Should return the same set of transports when bound repeatedly", func() {
			Expect(grpc.Bind(apiLayer, apiLayer.Channel)).To(HaveLen(13))
			Expect(grpc.Bind(apiLayer, apiLayer.Channel)).To(HaveLen(13))
		})
	})
})
