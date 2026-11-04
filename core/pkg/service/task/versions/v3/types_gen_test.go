// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v3_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v3 "github.com/synnaxlabs/synnax/pkg/service/task/versions/v3"
)

var _ = Describe("Validate", func() {
	It("Should accept a named task", func() {
		Expect(v3.Task{Name: "Named"}.Validate()).To(Succeed())
	})
	It("Should reject an empty name", func() {
		Expect(v3.Task{}.Validate()).To(MatchError(ContainSubstring("name: required")))
	})
})
