// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v2_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v2 "github.com/synnaxlabs/synnax/pkg/service/view/versions/v2"
)

var _ = Describe("Validate", func() {
	It("Should accept a named view", func() {
		Expect(v2.RangeView{Name: "Named"}.Validate()).To(Succeed())
	})
	It("Should reject an empty name", func() {
		Expect(v2.StatusView{}.Validate()).
			To(MatchError(ContainSubstring("name: required")))
	})
})
