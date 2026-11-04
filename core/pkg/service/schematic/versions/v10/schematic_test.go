// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v10_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	v10 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v10"
)

var _ = Describe("Schematic", func() {
	Describe("GorpKey", func() {
		It("Should return the schematic's key", func() {
			k := uuid.New()
			Expect(v10.Schematic{Key: k}.GorpKey()).To(Equal(k))
		})
	})

	Describe("SetOptions", func() {
		It("Should return no options", func() {
			Expect(v10.Schematic{}.SetOptions()).To(BeNil())
		})
	})
	Describe("ResourceID", func() {
		It("Should return the schematic resource identifier", func() {
			k := uuid.New()
			Expect(v10.Schematic{Key: k}.ResourceID()).To(Equal(resource.ID{
				Type: resource.TypeSchematic, Key: k.String(),
			}))
		})
	})
})
