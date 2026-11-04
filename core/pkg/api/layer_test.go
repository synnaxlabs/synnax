// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package api_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/api/batch"
)

var _ = Describe("Layer", func() {
	DescribeTable("Should refuse a batch request that writes channel storage",
		func(ctx SpecContext, endpoint string) {
			Expect(apiLayer.Batch.Exec(ctx, nil, batch.ExecRequest{
				Requests: []batch.Request{{Endpoint: endpoint}},
			})).Error().To(MatchError(ContainSubstring(fmt.Sprintf(
				"requests.0.endpoint: %q cannot run in a batch",
				endpoint,
			))))
		},
		Entry("channel create", "/channel/create"),
		Entry("channel delete", "/channel/delete"),
	)
})
