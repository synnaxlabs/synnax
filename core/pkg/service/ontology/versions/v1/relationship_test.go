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
	v1 "github.com/synnaxlabs/synnax/pkg/service/ontology/versions/v1"
)

var _ = Describe("Relationship", func() {
	Describe("GorpKey", func() {
		It("Should return the correct gorp key", func() {
			Expect(v1.Relationship{
				From: v1.ID{Type: "channel", Key: "qux"},
				To:   v1.ID{Type: "device", Key: "baz"},
				Type: v1.RelationshipTypeParentOf,
			}.GorpKey()).To(Equal("channel:qux->parent->device:baz"))
		})
	})
	Describe("SetOptions", func() {
		It("Should return nil", func() {
			Expect(v1.Relationship{}.SetOptions()).To(BeNil())
		})
	})
})
