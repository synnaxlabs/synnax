// Copyright 2025 Synnax Labs, Inc.
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

var _ = Describe("NewErrTargetNotFound", func() {
	It("Should return an error with the target address", func() {
		addr := address.Address("localhost:8080")
		err := address.NewErrTargetNotFound(addr)
		Expect(err).To(MatchError(address.ErrNotFound))
		Expect(err.Error()).To(ContainSubstring("target localhost:8080 not found"))
	})
})
