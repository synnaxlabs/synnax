// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v4_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v4 "github.com/synnaxlabs/synnax/pkg/service/log/versions/v4"
)

var _ = Describe("Log", func() {
	Describe("Validate", func() {
		It("Should reject an empty name and an invalid timestamp precision", func() {
			l := v4.Log{TimestampPrecision: 4}
			l.ApplyDefaults()
			Expect(l.Validate()).To(SatisfyAll(
				MatchError(ContainSubstring("name")),
				MatchError(ContainSubstring("timestamp_precision")),
			))
		})
	})
})
