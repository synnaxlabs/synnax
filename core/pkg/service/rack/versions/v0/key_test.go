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
	"github.com/synnaxlabs/synnax/pkg/service/node"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	v0 "github.com/synnaxlabs/synnax/pkg/service/rack/versions/v0"
)

// key composes node 3 in the upper 16 bits and local key 7 in the lower 16.
const key = v0.Key(3<<16 | 7)

var _ = Describe("Key", func() {
	Describe("Node", func() {
		It("Should return the node the rack is leased to", func() {
			Expect(key.Node()).To(Equal(node.Key(3)))
		})
	})

	Describe("LocalKey", func() {
		It("Should return the rack's key on its leaseholder node", func() {
			Expect(key.LocalKey()).To(Equal(uint16(7)))
		})
	})

	Describe("OntologyID", func() {
		It("Should return the rack ontology identifier", func() {
			Expect(key.OntologyID()).To(Equal(ontology.ID{
				Type: ontology.ResourceTypeRack,
				Key:  "196615",
			}))
		})
	})

	Describe("IsZero", func() {
		It("Should return true for the zero key", func() {
			Expect(v0.Key(0).IsZero()).To(BeTrue())
		})
		It("Should return false for a set key", func() {
			Expect(key.IsZero()).To(BeFalse())
		})
	})

	Describe("String", func() {
		It("Should format the key as its decimal value", func() {
			Expect(key.String()).To(Equal("196615"))
		})
	})
})
