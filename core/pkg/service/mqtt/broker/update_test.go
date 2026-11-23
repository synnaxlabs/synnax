// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package broker_test

import (
	"encoding/json/v2"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/mqtt/broker"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
	"github.com/vmihailenco/msgpack/v5"
)

var _ = Describe("Update", func() {
	identity := broker.Update{Variant: broker.IdentityUpdate{
		Identity: broker.Identity{
			Rack: 1,
			Name: "Plant Broker",
			Host: "mqtt.example.com",
			Port: new(uint16(8883)),
			Transport: broker.Transport{Variant: broker.TlsTransport{
				CaCertificate: new(uuid.MustParse("44444444-4444-4444-8444-444444444444")),
			}},
			Auth: broker.Auth{Variant: broker.UsernameAuth{
				Username: "admin",
				Password: new(uuid.MustParse("33333333-3333-4333-8333-333333333333")),
			}},
			ClientID:  new("plant-core"),
			KeepAlive: 60 * telem.Second,
			Sparkplug: broker.Sparkplug{
				Mode: broker.SparkplugMode{
					Variant: broker.PrimarySparkplugMode{HostID: "scada"},
				},
				Groups: []string{"Plant"},
			},
		},
	}}
	configuration := broker.Update{Variant: broker.ConfigurationUpdate{
		Configuration: broker.Configuration{
			Reads: []broker.Read{{Variant: broker.SparkplugRead{
				Group:    "Plant",
				EdgeNode: "Line1",
				Tag:      "Pressure",
				Index:    1,
				Channel:  2,
			}}},
			Writes: []broker.Write{{Variant: broker.PlainWrite{
				Topic:   "plant/valve",
				Field:   uuid.MustParse("55555555-5555-4555-8555-555555555555"),
				Channel: 3,
			}}},
			EdgeCommands: []broker.EdgeCommand{{
				Group:    "Plant",
				EdgeNode: "Synnax",
				Tag:      "Setpoint",
				Channel:  4,
			}},
		},
	}}

	DescribeTable("Should keep the group through msgpack",
		func(update broker.Update) {
			var got broker.Update
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
		func(update broker.Update) {
			var got broker.Update
			Expect(json.Unmarshal(
				MustSucceed(json.Marshal(update)),
				&got,
			)).To(Succeed())
			Expect(got.Variant).To(BeAssignableToTypeOf(update.Variant))
			Expect(MustSucceed(json.Marshal(got))).
				To(MatchJSON(MustSucceed(json.Marshal(update))))
		},
		Entry("identity", identity),
		Entry("configuration", configuration),
	)

	It("Should reject an unknown group in JSON", func() {
		var got broker.Update
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
		Expect(wire["name"]).To(Equal(fromJSON["name"]))
	})

	It("Should encode an empty update as nil", func() {
		var got broker.Update
		Expect(msgpack.Unmarshal(
			MustSucceed(msgpack.Marshal(broker.Update{})),
			&got,
		)).To(Succeed())
		Expect(got.Variant).To(BeNil())
	})

	It("Should reject msgpack that is not an update", func() {
		var got broker.Update
		Expect(msgpack.Unmarshal(
			MustSucceed(msgpack.Marshal("not an update")),
			&got,
		)).To(MatchError(ContainSubstring("msgpack")))
	})

	It("Should apply the identity defaults that msgpack leaves out", func() {
		raw := MustSucceed(msgpack.Marshal(map[string]any{
			"group": "identity",
			"rack":  1,
			"name":  "Plant Broker",
			"host":  "mqtt.example.com",
		}))
		var got broker.Update
		Expect(msgpack.Unmarshal(raw, &got)).To(Succeed())
		var expected broker.Identity
		expected.SetDefaults()
		expected.Rack = 1
		expected.Name = "Plant Broker"
		expected.Host = "mqtt.example.com"
		Expect(got.Variant).To(Equal(broker.IdentityUpdate{Identity: expected}))
		Expect(expected.Validate()).To(Succeed())
	})

	It("Should reject an unknown group", func() {
		raw := MustSucceed(msgpack.Marshal(map[string]any{"group": "other"}))
		var got broker.Update
		Expect(msgpack.Unmarshal(raw, &got)).To(MatchError(validate.ErrValidation))
	})
})
