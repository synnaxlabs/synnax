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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v0 "github.com/synnaxlabs/synnax/pkg/service/device/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
)

var _ = Describe("Device", func() {
	Describe("GorpKey", func() {
		It("Should return the device's key", func() {
			Expect(v0.Device{Key: "dev-1"}.GorpKey()).To(Equal(v0.Key("dev-1")))
		})
	})

	Describe("SetOptions", func() {
		It("Should return no options", func() {
			Expect(v0.Device{}.SetOptions()).To(BeNil())
		})
	})

	Describe("OntologyID", func() {
		It("Should return the device ontology identifier", func() {
			Expect(v0.Device{Key: "dev-1"}.OntologyID()).To(Equal(ontology.ID{
				Type: ontology.ResourceTypeDevice, Key: "dev-1",
			}))
		})
	})
})
