// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package validate_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Validate", func() {
	Describe("Ternay", func() {
		It("Should accumulate the error if the condition is met", func() {
			v := validate.New("demo")
			v.Ternaryf("field", true, "error")
			executed := false
			v.Funcf(func() bool {
				executed = true
				return true
			}, "error")
			Expect(v.Error()).To(HaveOccurred())
			Expect(executed).To(BeFalse())
		})
	})
})
