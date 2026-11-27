// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package server_test

import (
	"encoding/json/v2"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/opcua/server"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Update", func() {
	identity := server.Update{Variant: server.IdentityUpdate{
		Identity: server.Identity{
			Rack: 1,
			Name: "Boiler PLC",
			Connection: func() server.Connection {
				c := newConnection("opc.tcp://boiler:4840")
				c.Auth = server.Auth{Variant: server.UsernameAuth{Username: "op"}}
				c.Security = signed(pinned(uuid.New()))
				return c
			}(),
		},
	}}
	configuration := server.Update{Variant: server.ConfigurationUpdate{
		Configuration: server.Configuration{
			Reads: []server.Read{
				{Node: "nsu=urn:plc;s=Temp", Field: "a", Index: 1, Channel: 2},
			},
			Writes: []server.Write{{Node: "nsu=urn:plc;s=Valve", Channel: 4}},
		},
	}}

	DescribeTable("Should keep the group through JSON",
		func(update server.Update) {
			var got server.Update
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
		var got server.Update
		Expect(json.Unmarshal([]byte(`{"group":"other"}`), &got)).
			To(MatchError(ContainSubstring("unknown group")))
	})
})
