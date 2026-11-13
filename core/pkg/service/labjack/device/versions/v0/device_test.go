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
	v0 "github.com/synnaxlabs/synnax/pkg/service/labjack/device/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/types"
)

var _ = Describe("Device", func() {
	It("Should have a table prefix of its own", func() {
		Expect(types.Name[v0.Device]()).To(Equal("labjack_device"))
	})

	It("Should identify itself by its key", func() {
		key := uuid.MustParse("22222222-2222-4222-8222-222222222222")
		d := v0.Device{Key: key}
		Expect(d.GorpKey()).To(Equal(key))
		Expect(d.ResourceID()).To(Equal(resource.ID{
			Type: resource.TypeLabjackDevice,
			Key:  key.String(),
		}))
	})
})
