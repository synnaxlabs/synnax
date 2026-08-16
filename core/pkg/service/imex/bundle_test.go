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
	It("Should keep distinct names unchanged", func() {
		claims := imex.NewClaims()
		Expect(claims.Claim("Pressure.json", ".json")).To(Equal("Pressure.json"))
		Expect(claims.Claim("Thrust.json", ".json")).To(Equal("Thrust.json"))
	})

	It("Should suffix a name already claimed", func() {
		claims := imex.NewClaims()
		Expect(claims.Claim("New panel.json", ".json")).To(Equal("New panel.json"))
		Expect(claims.Claim("New panel.json", ".json")).To(Equal("New panel (1).json"))
		Expect(claims.Claim("New panel.json", ".json")).To(Equal("New panel (2).json"))
	})

	It("Should suffix a reserved name", func() {
		claims := imex.NewClaims("Manifest.json")
		Expect(claims.Claim("manifest.json", ".json")).To(Equal("manifest (1).json"))
	})

	It("Should suffix names that fold together", func() {
		claims := imex.NewClaims()
		Expect(claims.Claim("Pressure.json", ".json")).To(Equal("Pressure.json"))
		Expect(claims.Claim("pressure.json", ".json")).To(Equal("pressure (1).json"))
	})

	It("Should skip a suffix another member holds", func() {
		claims := imex.NewClaims()
		Expect(claims.Claim("New panel (1).json", ".json")).
			To(Equal("New panel (1).json"))
		Expect(claims.Claim("New panel.json", ".json")).To(Equal("New panel.json"))
		Expect(claims.Claim("New panel.json", ".json")).To(Equal("New panel (2).json"))
	})

	It("Should suffix a directory name without an extension", func() {
		claims := imex.NewClaims()
		Expect(claims.Claim("Valves", "")).To(Equal("Valves"))
		Expect(claims.Claim("Valves", "")).To(Equal("Valves (1)"))
	})
})
