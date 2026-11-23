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
	"github.com/synnaxlabs/x/encoding/orc"
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

	DescribeTable("Should reject an invalid slave",
		func(change func(*v0.Slave), path string) {
			m := slave()
			change(&m)
			Expect(m.Validate()).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(path)),
			))
		},
		Entry("no name", func(m *v0.Slave) { m.Name = "" }, "name"),
		Entry(
			"an identifier that starts with a digit",
			func(m *v0.Slave) { m.Identifier = "1inlet" },
			"identifier",
		),
		Entry(
			"an identifier longer than 12 characters",
			func(m *v0.Slave) { m.Identifier = "inlet_pressure" },
			"identifier",
		),
		Entry(
			"a read binding without a channel",
			func(m *v0.Slave) {
				m.ReadBindings = []v0.ReadBinding{{Index: 0x6000, SubIndex: 1}}
			},
			"channel",
		),
		Entry(
			"a write binding without a command channel",
			func(m *v0.Slave) {
				m.WriteBindings = []v0.WriteBinding{{Index: 0x7000, SubIndex: 1}}
			},
			"command",
		),
	)

	It("Should accept an empty identifier", func() {
		m := slave()
		m.Identifier = ""
		Expect(m.Validate()).To(Succeed())
	})

	It("Should keep a 64-bit startup value exactly", func() {
		m := slave()
		m.StartupParameters = []v0.StartupParameter{{
			Variant: v0.Uint64StartupParameter{
				StartupParameterBase: v0.StartupParameterBase{
					Index:    0x8000,
					SubIndex: 1,
				},
				Value: 1<<63 + 1,
			},
		}}
		w := orc.NewWriter(0)
		Expect(m.EncodeOrc(w)).To(Succeed())
		r := orc.NewReader(nil)
		r.ResetBytes(w.Bytes())
		var got v0.Slave
		Expect(got.DecodeOrc(r)).To(Succeed())
		Expect(got.StartupParameters).To(Equal(m.StartupParameters))
	})
})
