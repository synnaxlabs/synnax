// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package resource_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/resource"
)

var _ = Describe("Resource", func() {
	Describe("NewClosedError", func() {
		It("Should return an error with the correct message", func() {
			Expect(resource.NewClosedError("test")).To(And(
				MatchError(resource.ErrClosed),
				MatchError(ContainSubstring("cannot complete operation on closed test")),
			))
		})
	})

})
