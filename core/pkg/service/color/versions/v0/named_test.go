// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v0 "github.com/synnaxlabs/synnax/pkg/service/color/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	xcolor "github.com/synnaxlabs/x/color"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Named", func() {
	var (
		key   uuid.UUID
		light xcolor.Color
		named v0.Named
	)
	BeforeEach(func() {
		key = uuid.New()
		light = MustSucceed(xcolor.Parse("#ff0000"))
		dark := MustSucceed(xcolor.Parse("#00ff00"))
		named = v0.Named{Key: key, Name: "Breaker", Light: light, Dark: &dark}
	})

	Describe("GorpKey", func() {
		It("Should return the key of the named color", func() {
			Expect(named.GorpKey()).To(Equal(key))
		})
	})

	Describe("ResourceID", func() {
		It("Should return the color resource with the key", func() {
			Expect(named.ResourceID()).To(Equal(resource.ID{
				Type: resource.TypeColor,
				Key:  key.String(),
			}))
		})
	})

	Describe("Inline", func() {
		It("Should return a fixed color with the light value", func() {
			Expect(named.Inline()).To(Equal(v0.FixedRef{Value: light}))
		})
	})
})
