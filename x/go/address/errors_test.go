// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package address_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/address"
)

var _ = Describe("Errors", func() {
	Describe("NewTargetNotFoundError", func() {
		It("Should return an error with the correct message", func() {
			Expect(address.NewTargetNotFoundError("localhost:9090")).
				To(And(
					MatchError(ContainSubstring("target localhost:9090 not found")),
					MatchError(address.ErrNotFound),
				))
		})
	})
})
