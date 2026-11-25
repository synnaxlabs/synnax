// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package slave_test

import (
	"encoding/json/v2"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/ethercat/slave"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
	"github.com/vmihailenco/msgpack/v5"
)

var _ = Describe("Update", func() {
	identity := slave.Update{Variant: slave.IdentityUpdate{
		Identity: slave.Identity{
			Rack:        1,
			Serial:      1001,
			VendorID:    2,
			ProductCode: 0x0C1E3052,
			Master:      uuid.New(),
			Pdos: slave.PDOs{
				Inputs:  []slave.PDOEntry{},
				Outputs: []slave.PDOEntry{},
			},
			OpModes: []slave.OpMode{},
		},
	}}
	configuration := slave.Update{Variant: slave.ConfigurationUpdate{
		Configuration: slave.Configuration{
			Name:              "EL3102",
			Identifier:        "inlet",
			StartupParameters: []slave.StartupParameter{},
			ReadBindings:      []slave.ReadBinding{},
			WriteBindings:     []slave.WriteBinding{},
		},
	}}

	DescribeTable("Should keep the group through msgpack",
		func(update slave.Update) {
			var got slave.Update
			Expect(msgpack.Unmarshal(
				MustSucceed(msgpack.Marshal(update)),
				&got,
			)).To(Succeed())
			Expect(got).To(Equal(update))
		},
		Entry("identity", identity),
		Entry("configuration", configuration),
	)

	DescribeTable("Should keep the group through JSON",
		func(update slave.Update) {
			var got slave.Update
			Expect(json.Unmarshal(
				MustSucceed(json.Marshal(update)),
				&got,
			)).To(Succeed())
			Expect(got).To(Equal(update))
		},
		Entry("identity", identity),
		Entry("configuration", configuration),
	)

	It("Should reject an unknown group in JSON", func() {
		var got slave.Update
		Expect(json.Unmarshal([]byte(`{"group":"other"}`), &got)).
			To(MatchError(ContainSubstring("unknown group")))
	})

	It("Should match the JSON shape in msgpack", func() {
		var wire map[string]any
		Expect(msgpack.Unmarshal(
			MustSucceed(msgpack.Marshal(identity)),
			&wire,
		)).To(Succeed())
		var fromJSON map[string]any
		Expect(json.Unmarshal(
			MustSucceed(json.Marshal(identity)),
			&fromJSON,
		)).To(Succeed())
		Expect(wire["group"]).To(Equal(fromJSON["group"]))
		Expect(wire).To(HaveKey("vendor_id"))
		Expect(fromJSON).To(HaveKey("vendor_id"))
	})

	It("Should encode an empty update as nil", func() {
		var got slave.Update
		Expect(msgpack.Unmarshal(
			MustSucceed(msgpack.Marshal(slave.Update{})),
			&got,
		)).To(Succeed())
		Expect(got.Variant).To(BeNil())
	})

	It("Should reject msgpack that is not an update", func() {
		var got slave.Update
		Expect(msgpack.Unmarshal(
			MustSucceed(msgpack.Marshal("not an update")),
			&got,
		)).To(MatchError(ContainSubstring("msgpack")))
	})

	It("Should reject an unknown group", func() {
		raw := MustSucceed(msgpack.Marshal(map[string]any{"group": "other"}))
		var got slave.Update
		Expect(msgpack.Unmarshal(raw, &got)).To(MatchError(validate.ErrValidation))
	})
})
