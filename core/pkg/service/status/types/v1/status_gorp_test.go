// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	v1 "github.com/synnaxlabs/synnax/pkg/service/status/types/v1"
)

var _ = Describe("Status", func() {
	Describe("GorpKey", func() {
		It("Should return the status's key", func() {
			Expect(v1.Status[any]{Key: "st-1"}.GorpKey()).To(Equal("st-1"))
		})
	})

	Describe("SetOptions", func() {
		It("Should return no options", func() {
			Expect(v1.Status[any]{}.SetOptions()).To(BeNil())
		})
	})
})

var _ = Describe("OntologyID", func() {
	It("Should return the status ontology identifier", func() {
		Expect(v1.Status[any]{Key: "st-1"}.OntologyID()).To(Equal(ontology.ID{
			Type: ontology.ResourceTypeStatus, Key: "st-1",
		}))
	})
})
