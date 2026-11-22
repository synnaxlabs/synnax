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
	v0 "github.com/synnaxlabs/synnax/pkg/service/opcua/server/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/types"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Server", func() {
	server := func() v0.Server {
		return v0.Server{
			Key:  uuid.MustParse("22222222-2222-4222-8222-222222222222"),
			Rack: 1,
			Name: "PLC",
			Connection: v0.Connection{
				Endpoint:                "opc.tcp://plc:4840",
				SecurityMode:            v0.SecurityModeNone,
				SecurityPolicy:          v0.SecurityPolicyNone,
				ClientTimeoutMs:         5000,
				SecureChannelLifetimeMs: 600000,
				SessionTimeoutMs:        1200000,
			},
		}
	}

	It("Should have a table prefix of its own", func() {
		Expect(types.Name[v0.Server]()).To(Equal("opcua_server"))
	})

	It("Should identify itself by its key", func() {
		s := server()
		Expect(s.GorpKey()).To(Equal(s.Key))
		Expect(s.ResourceID()).To(Equal(resource.ID{
			Type: resource.TypeOpcuaServer,
			Key:  s.Key.String(),
		}))
	})

	It("Should accept a valid server", func() {
		Expect(server().Validate()).To(Succeed())
	})

	DescribeTable("Should reject an invalid server",
		func(change func(*v0.Server), message string) {
			s := server()
			change(&s)
			Expect(s.Validate()).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(message)),
			))
		},
		Entry("no name", func(s *v0.Server) { s.Name = "" }, "name"),
		Entry(
			"no endpoint",
			func(s *v0.Server) { s.Connection.Endpoint = "" },
			"endpoint",
		),
		Entry(
			"an unknown security policy",
			func(s *v0.Server) { s.Connection.SecurityPolicy = "Basic512" },
			"security_policy",
		),
		Entry(
			"a request timeout under 100 ms",
			func(s *v0.Server) { s.Connection.ClientTimeoutMs = 99 },
			"client_timeout_ms",
		),
		Entry(
			"a channel lifetime under 1 s",
			func(s *v0.Server) { s.Connection.SecureChannelLifetimeMs = 999 },
			"secure_channel_lifetime_ms",
		),
		Entry(
			"a session timeout under 1 s",
			func(s *v0.Server) { s.Connection.SessionTimeoutMs = 999 },
			"session_timeout_ms",
		),
		Entry(
			"a field with no node",
			func(s *v0.Server) { s.ReadFields = []v0.ReadField{{Field: "a", Channel: 1}} },
			"node_id",
		),
	)
})
