// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v9_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v9 "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/v9"
)

var _ = Describe("LinePlot", func() {
	Describe("Validate", func() {
		It("Should reject an empty name", func() {
			lp := v9.LinePlot{}
			lp.ApplyDefaults()
			Expect(lp.Validate()).To(MatchError(ContainSubstring("name: required")))
		})
	})
})
