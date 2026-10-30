// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package device_test

import (
	"encoding/json/v2"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/ni/device"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
	"github.com/vmihailenco/msgpack/v5"
)

var _ = Describe("Update", func() {
	identity := device.Update{Variant: device.IdentityUpdate{
		Identity: device.Identity{Rack: 1, Serial: "01A", Model: "PXI-6255"},
	}}
	configuration := device.Update{Variant: device.ConfigurationUpdate{
		Configuration: device.Configuration{Name: "Module", Identifier: "dev1"},
	}}

	DescribeTable("Should keep the group through msgpack",
		func(update device.Update) {
			var got device.Update
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
		func(update device.Update) {
			var got device.Update
			Expect(json.Unmarshal(
				MustSucceed(json.Marshal(update)),
				&got,
			)).To(Succeed())
			Expect(got.Variant).To(BeAssignableToTypeOf(update.Variant))
		},
		Entry("identity", identity),
		Entry("configuration", configuration),
	)

	It("Should reject an unknown group in JSON", func() {
		var got device.Update
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
		Expect(wire["serial"]).To(Equal(fromJSON["serial"]))
	})

	It("Should encode an empty update as nil", func() {
		var got device.Update
		Expect(msgpack.Unmarshal(
			MustSucceed(msgpack.Marshal(device.Update{})),
			&got,
		)).To(Succeed())
		Expect(got.Variant).To(BeNil())
	})

	It("Should reject an unknown group", func() {
		raw := MustSucceed(msgpack.Marshal(map[string]any{"group": "other"}))
		var got device.Update
		Expect(msgpack.Unmarshal(raw, &got)).To(MatchError(validate.ErrValidation))
	})
})
