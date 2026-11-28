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
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/types"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Server", func() {
	server := func() v0.Server {
		var c v0.Connection
		c.SetDefaults()
		c.Endpoint = "opc.tcp://plc:4840"
		return v0.Server{
			Key:        uuid.MustParse("22222222-2222-4222-8222-222222222222"),
			Rack:       1,
			Name:       "PLC",
			Connection: c,
			Reads:      []v0.Read{{Node: "nsu=urn:plc;s=Temp", Index: 1, Channel: 2}},
			Writes:     []v0.Write{{Node: "nsu=urn:plc;s=Valve", Channel: 3}},
		}
	}
	secured := func() v0.SecuredChannel {
		var c v0.SecuredChannel
		c.SetDefaults()
		c.Policy = v0.SecurityPolicyBasic256Sha256
		return c
	}
	signed := func() v0.Security {
		return v0.Security{Variant: v0.SignSecurity{SecuredChannel: secured()}}
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

	It("Should default to an anonymous connection without security", func() {
		s := server()
		Expect(s.Connection.Auth.Type()).To(Equal(v0.AnonymousAuthType))
		Expect(s.Connection.Security.Mode()).To(Equal(v0.NoneSecurityType))
		Expect(s.Connection.ClientTimeout).To(Equal(5 * telem.Second))
	})

	It("Should default a secured connection to trust on first use", func() {
		Expect(secured().ServerTrust.Type()).To(Equal(v0.FirstUseServerTrustType))
	})

	DescribeTable("Should accept a valid server",
		func(change func(*v0.Server)) {
			s := server()
			change(&s)
			Expect(s.Validate()).To(Succeed())
		},
		Entry("as it is", func(*v0.Server) {}),
		Entry("with a path and an IPv6 host", func(s *v0.Server) {
			s.Connection.Endpoint = "opc.tcp://[::1]:4840/UA/Server"
		}),
		Entry("with a signed connection", func(s *v0.Server) {
			s.Connection.Security = signed()
		}),
		Entry("with a user name login", func(s *v0.Server) {
			s.Connection.Auth = v0.Auth{Variant: v0.UsernameAuth{Username: "op"}}
		}),
		Entry("with the highest port", func(s *v0.Server) {
			s.Connection.Endpoint = "opc.tcp://plc:65535"
		}),
	)

	DescribeTable(
		"Should reject an invalid server",
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
			"an endpoint of another scheme",
			func(s *v0.Server) { s.Connection.Endpoint = "http://plc:4840" },
			"opc.tcp://",
		),
		Entry(
			"an endpoint without a host",
			func(s *v0.Server) { s.Connection.Endpoint = "opc.tcp://:4840" },
			"opc.tcp://",
		),
		Entry(
			"a request timeout under 100 ms",
			func(s *v0.Server) { s.Connection.ClientTimeout = 99 * telem.Millisecond },
			"client_timeout",
		),
		Entry(
			"a channel lifetime under 1 s",
			func(s *v0.Server) { s.Connection.SecureChannelLifetime = 999 * telem.Millisecond },
			"secure_channel_lifetime",
		),
		Entry(
			"a session timeout under 1 s",
			func(s *v0.Server) { s.Connection.SessionTimeout = 999 * telem.Millisecond },
			"session_timeout",
		),
		Entry(
			"a user name login without a user name",
			func(s *v0.Server) { s.Connection.Auth = v0.Auth{Variant: v0.UsernameAuth{}} },
			"username",
		),
		Entry(
			"an unknown security policy",
			func(s *v0.Server) {
				c := secured()
				c.Policy = "None"
				s.Connection.Security = v0.Security{
					Variant: v0.SignSecurity{SecuredChannel: c},
				}
			},
			"policy",
		),
		Entry(
			"a name of only spaces",
			func(s *v0.Server) { s.Name = "  " },
			"Name must",
		),
		Entry("no rack", func(s *v0.Server) { s.Rack = 0 }, "rack"),
		Entry(
			"port 0",
			func(s *v0.Server) { s.Connection.Endpoint = "opc.tcp://plc:0" },
			"65535",
		),
		Entry(
			"a port above 65535",
			func(s *v0.Server) { s.Connection.Endpoint = "opc.tcp://plc:65536" },
			"65535",
		),
		Entry(
			"a request timeout as long as the session timeout",
			func(s *v0.Server) { s.Connection.ClientTimeout = s.Connection.SessionTimeout },
			"session timeout",
		),
		Entry(
			"a user name of only spaces",
			func(s *v0.Server) {
				s.Connection.Auth = v0.Auth{Variant: v0.UsernameAuth{Username: " "}}
			},
			"User name must",
		),
		Entry(
			"a certificate login without a certificate",
			func(s *v0.Server) {
				s.Connection.Auth = v0.Auth{Variant: v0.CertificateAuth{}}
			},
			"certificate",
		),
		Entry(
			"a pinned trust without a certificate",
			func(s *v0.Server) {
				c := secured()
				c.ServerTrust = v0.ServerTrust{Variant: v0.PinnedServerTrust{}}
				s.Connection.Security = v0.Security{
					Variant: v0.SignSecurity{SecuredChannel: c},
				}
			},
			"certificate",
		),
		Entry(
			"a read without a node",
			func(s *v0.Server) { s.Reads[0].Node = "" },
			"node",
		),
		Entry(
			"a read without an index",
			func(s *v0.Server) { s.Reads[0].Index = 0 },
			"index",
		),
		Entry(
			"a write without a channel",
			func(s *v0.Server) { s.Writes[0].Channel = 0 },
			"channel",
		),
	)
})
