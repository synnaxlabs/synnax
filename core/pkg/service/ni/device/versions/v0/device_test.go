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
	v0 "github.com/synnaxlabs/synnax/pkg/service/ni/device/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/types"
)

var _ = Describe("Device", func() {
	It("Should have a table prefix of its own", func() {
		Expect(types.Name[v0.Device]()).To(Equal("ni_device"))
	})

	It("Should identify itself by its key", func() {
		key := uuid.New()
		d := v0.Device{Key: key}
		Expect(d.GorpKey()).To(Equal(key))
		Expect(d.SetOptions()).To(BeNil())
		Expect(d.OntologyID()).To(Equal(ontology.ID{
			Type: ontology.ResourceTypeNiDevice,
			Key:  key.String(),
		}))
	})
})
