// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package validate_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("NormalizeName", func() {
	It("Should compose a decomposed name", func() {
		Expect(validate.NormalizeName("e\u0301")).To(Equal("\u00E9"))
	})

	It("Should allocate nothing for an ASCII name or a name in NFC", func() {
		for _, name := range []string{"pt_unique_01", "Stand A-01", "\u5727\u529b \u00E9"} {
			Expect(testing.AllocsPerRun(10, func() {
				_ = validate.NormalizeName(name)
			})).To(BeZero())
		}
	})
})
