// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package pb_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/opcua/server"
	"github.com/synnaxlabs/synnax/pkg/service/opcua/server/pb"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/filter"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Translator", func() {
	It("Should round trip a server with its unions, bindings, and status", func() {
		key := uuid.New()
		password := uuid.New()
		clientCert := uuid.New()
		s := server.Server{
			Key:  key,
			Rack: rack.NewKey(1, 2),
			Name: "PLC",
			Connection: server.Connection{
				Endpoint: "opc.tcp://plc:4840",
				Auth: server.Auth{Variant: server.UsernameAuth{
					Username: "operator",
					Password: &password,
				}},
				Security: server.Security{Variant: server.SignAndEncryptSecurity{
					SecuredChannel: server.SecuredChannel{
						Policy:            server.SecurityPolicyBasic256Sha256,
						ClientCertificate: &clientCert,
						ServerTrust: server.ServerTrust{
							Variant: server.PinnedServerTrust{Certificate: uuid.New()},
						},
					},
				}},
				ClientTimeout:         5 * telem.Second,
				SecureChannelLifetime: 10 * telem.Minute,
				SessionTimeout:        20 * telem.Minute,
			},
			Reads: []server.Read{
				{Node: "nsu=urn:plc;s=Temp", Field: "a.b", Index: 1, Channel: 2},
			},
			Writes: []server.Write{{Node: "i=2258", Channel: 3}},
			Status: &server.Status{
				Key:     "status",
				Owner:   new(server.ResourceID(key)),
				Variant: status.VariantSuccess,
				Message: "Server connected",
				Labels:  []label.Key{uuid.New()},
				Details: server.StatusDetails{
					Rack:      rack.NewKey(1, 2),
					Server:    key,
					Connected: true,
				},
			},
		}
		translated := MustSucceed(pb.ServerToPB(s))
		Expect(pb.ServerFromPB(translated)).To(Equal(s))
	})

	DescribeTable("Should round trip an update", func(u server.Update) {
		translated := MustSucceed(pb.UpdateToPB(u))
		Expect(pb.UpdateFromPB(translated)).To(Equal(u))
	},
		Entry("of the identity group", server.Update{Variant: server.IdentityUpdate{
			Identity: server.Identity{
				Rack: rack.NewKey(1, 2),
				Name: "PLC",
				Connection: server.Connection{
					Endpoint: "opc.tcp://plc:4840",
					Auth: server.Auth{Variant: server.CertificateAuth{
						Certificate: uuid.New(),
					}},
					Security:       server.Security{Variant: server.NoneSecurity{}},
					ClientTimeout:  telem.Second,
					SessionTimeout: telem.Minute,
				},
			},
		}}),
		Entry("of the configuration group", server.Update{
			Variant: server.ConfigurationUpdate{Configuration: server.Configuration{
				Reads:  []server.Read{{Node: "i=2258", Index: 1, Channel: 2}},
				Writes: []server.Write{},
			}},
		}),
	)

	It("Should round trip a filter tree", func() {
		key := uuid.New()
		name := "PLC"
		rackKey := rack.NewKey(1, 2)
		w := server.Where{
			And: []server.Where{{Key: &filter.Set[server.Key]{Eq: &key}}},
			Or: []server.Where{
				{Name: &filter.String{Eq: &name}},
				{Rack: &filter.Set[rack.Key]{In: []rack.Key{rackKey}}},
			},
			Not: &server.Where{Key: &filter.Set[server.Key]{In: []server.Key{key}}},
		}
		translated := MustSucceed(pb.WhereToPB(w))
		Expect(pb.WhereFromPB(translated)).To(Equal(w))
	})

	It("Should reject an unknown security policy", func() {
		Expect(pb.SecurityPolicyToPB("None")).Error().
			To(MatchError(ContainSubstring("unrecognized")))
	})

	It("Should reject a server key that is not a UUID", func() {
		translated := MustSucceed(pb.ServerToPB(server.Server{Key: uuid.New()}))
		translated.Key = "not-a-uuid"
		Expect(pb.ServerFromPB(translated)).Error().
			To(MatchError(ContainSubstring("invalid uuid")))
	})
})
