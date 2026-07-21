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
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	channel "github.com/synnaxlabs/synnax/pkg/service/channel/types/v0"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	v0 "github.com/synnaxlabs/synnax/pkg/service/ranger/alias/types/v0"
)

var _ = Describe("Alias", func() {
	Describe("GorpKey", func() {
		It("Should join the range and channel keys", func() {
			rng, ch := uuid.New(), channel.Key(65538)
			Expect(v0.Alias{Range: rng, Channel: ch}.GorpKey()).
				To(Equal(rng.String() + "---" + ch.String()))
		})
	})

	Describe("SetOptions", func() {
		It("Should return no options", func() {
			Expect(v0.Alias{}.SetOptions()).To(BeNil())
		})
	})
	Describe("OntologyID", func() {
		It("Should return the alias ontology identifier", func() {
			a := v0.Alias{Range: uuid.New(), Channel: channel.Key(65538)}
			Expect(a.OntologyID()).To(Equal(ontology.ID{
				Type: ontology.ResourceTypeRangeAlias, Key: a.GorpKey(),
			}))
		})
	})
})
