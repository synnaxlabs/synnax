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
	v0 "github.com/synnaxlabs/synnax/pkg/service/ethercat/master/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/types"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Master", func() {
	master := func() v0.Master {
		return v0.Master{
			Key:       uuid.MustParse("22222222-2222-4222-8222-222222222222"),
			Rack:      1,
			Name:      "EtherCAT eth0",
			Interface: "eth0",
		}
	}

	It("Should have a table prefix of its own", func() {
		Expect(types.Name[v0.Master]()).To(Equal("ethercat_master"))
	})

	It("Should identify itself by its key", func() {
		m := master()
		Expect(m.GorpKey()).To(Equal(m.Key))
		Expect(m.ResourceID()).To(Equal(resource.ID{
			Type: resource.TypeEthercatMaster,
			Key:  m.Key.String(),
		}))
	})

	It("Should accept a valid master", func() {
		Expect(master().Validate()).To(Succeed())
	})

	DescribeTable("Should reject an invalid master",
		func(change func(*v0.Master), path string) {
			m := master()
			change(&m)
			Expect(m.Validate()).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(path)),
			))
		},
		Entry("no name", func(m *v0.Master) { m.Name = "" }, "name"),
		Entry("no interface", func(m *v0.Master) { m.Interface = "" }, "interface"),
	)
})
