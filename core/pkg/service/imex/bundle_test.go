// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package imex_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Manifest", func() {
	It("Should marshal the bundle headers", func() {
		raw := MustSucceed(json.Marshal(imex.Manifest{
			Version: 1,
			Type:    "project",
			Name:    "Test Stand 12",
		}))
		var m map[string]any
		Expect(json.Unmarshal(raw, &m)).To(Succeed())
		Expect(m).To(Equal(map[string]any{
			"version": float64(1),
			"type":    "project",
			"name":    "Test Stand 12",
		}))
		Expect(m).To(HaveKeyWithValue("version", BeNumerically("==", 1)))
		Expect(m).To(HaveKeyWithValue("type", "project"))
		Expect(m).To(HaveKeyWithValue("name", "Test Stand 12"))
	})
})

var _ = Describe("Claims", func() {
	It("Should claim distinct names", func() {
		claims := imex.NewClaims("")
		Expect(claims.Claim("Pressure", "Pressure.json")).To(Succeed())
		Expect(claims.Claim("Thrust", "Thrust.json")).To(Succeed())
	})

	It("Should reject a reserved name in its display form", func() {
		claims := imex.NewClaims("", "Manifest.json")
		Expect(claims.Claim("manifest", "manifest.json")).
			To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(
					`"manifest" takes the reserved file name "Manifest.json"`,
				)),
			))
	})

	It("Should reject names that fold together", func() {
		claims := imex.NewClaims("g/")
		Expect(claims.Claim("Pressure", "Pressure.json")).To(Succeed())
		Expect(claims.Claim("pressure", "pressure.json")).
			To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(
					`"Pressure" and "pressure" both export to "g/pressure.json"`,
				)),
			))
	})
})
