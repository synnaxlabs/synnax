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
	"github.com/synnaxlabs/synnax/pkg/service/node"
	v1 "github.com/synnaxlabs/synnax/pkg/service/rack/versions/v1"
)

// key composes node 3 in the upper 16 bits and local key 7 in the lower 16.
const key = v1.Key(3<<16 | 7)

var _ = Describe("Rack", func() {
	Describe("GorpKey", func() {
		It("Should return the rack's key", func() {
			Expect(v1.Rack{Key: key}.GorpKey()).To(Equal(key))
		})
	})

	Describe("SetOptions", func() {
		It("Should lease the rack to its node", func() {
			Expect(v1.Rack{Key: key}.SetOptions()).To(Equal([]any{node.Key(3)}))
		})
	})
	Describe("OntologyID", func() {
		It("Should return the rack ontology identifier", func() {
			Expect(v1.Rack{Key: key}.OntologyID()).To(Equal(key.OntologyID()))
		})
	})
})
