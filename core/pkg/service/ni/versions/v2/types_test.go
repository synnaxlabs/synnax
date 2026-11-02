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
	v2 "github.com/synnaxlabs/synnax/pkg/service/ni/versions/v2"
)

var _ = Describe("AccelChargeSensitivityUnits", func() {
	Describe("IsValid", func() {
		It("Should accept a defined unit", func() {
			unit := v2.AccelChargeSensitivityUnitsPicoCoulombsPerG
			Expect(unit.IsValid()).To(BeTrue())
		})

		It("Should reject an undefined unit", func() {
			Expect(v2.AccelChargeSensitivityUnits("Bogus").IsValid()).To(BeFalse())
		})
	})
})
