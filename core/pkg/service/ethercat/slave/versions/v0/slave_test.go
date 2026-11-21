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
	v0 "github.com/synnaxlabs/synnax/pkg/service/ethercat/slave/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/types"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Slave", func() {
	slave := func() v0.Slave {
		return v0.Slave{
			Key:         uuid.MustParse("22222222-2222-4222-8222-222222222222"),
			Rack:        1,
			Name:        "EL3102",
			VendorID:    2,
			ProductCode: 0x0C1E3052,
			Master:      uuid.MustParse("33333333-3333-4333-8333-333333333333"),
		}
	}

	It("Should have a table prefix of its own", func() {
		Expect(types.Name[v0.Slave]()).To(Equal("ethercat_slave"))
	})

	It("Should identify itself by its key", func() {
		m := slave()
		Expect(m.GorpKey()).To(Equal(m.Key))
		Expect(m.ResourceID()).To(Equal(resource.ID{
			Type: resource.TypeEthercatSlave,
			Key:  m.Key.String(),
		}))
	})

	It("Should accept a valid slave", func() {
		Expect(slave().Validate()).To(Succeed())
	})

	It("Should reject a slave without a name", func() {
		m := slave()
		m.Name = ""
		Expect(m.Validate()).To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("name")),
		))
	})
})
