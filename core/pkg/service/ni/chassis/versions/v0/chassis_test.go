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
	v0 "github.com/synnaxlabs/synnax/pkg/service/ni/chassis/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/types"
)

var _ = Describe("Chassis", func() {
	It("Should have a table prefix of its own", func() {
		Expect(types.Name[v0.Chassis]()).To(Equal("ni_chassis"))
	})

	It("Should identify itself by its key", func() {
		key := uuid.New()
		c := v0.Chassis{Key: key}
		Expect(c.GorpKey()).To(Equal(key))
		Expect(c.SetOptions()).To(BeNil())
		Expect(c.OntologyID()).To(Equal(ontology.ID{
			Type: ontology.ResourceTypeNiChassis,
			Key:  key.String(),
		}))
	})
})
