// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// The channel package declares And/Or/Not, which collide with a gomega dot-import, so
// this whitebox test qualifies gomega.
package channel

import (
	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var _ = Describe("ShouldValidateNames", func() {
	enabled, disabled := true, false
	DescribeTable("reports whether channel-name validation is enabled",
		func(validateNames *bool, expected bool) {
			s := &Service{cfg: ServiceConfig{ValidateNames: validateNames}}
			gomega.Expect(s.ShouldValidateNames()).To(gomega.Equal(expected))
		},
		Entry("nil defaults to enabled", (*bool)(nil), true),
		Entry("explicitly enabled", &enabled, true),
		Entry("explicitly disabled", &disabled, false),
	)
})
