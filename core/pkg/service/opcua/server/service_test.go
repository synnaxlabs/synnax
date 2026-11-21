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
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/certificate"
	certtestutil "github.com/synnaxlabs/synnax/pkg/service/certificate/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/opcua/server"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/secret"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/search"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Service", func() {
	var (
		e  env
		tx gorp.Tx
	)
	BeforeEach(func(ctx SpecContext) {
		e = openEnv(ctx, db, nil)
		tx = DeferClose(db.OpenTx())
	})

	newServer := func() server.Server {
		return server.Server{
			Rack:       e.racks.EmbeddedKey,
			Name:       "Boiler PLC",
			Connection: server.Connection{Endpoint: "opc.tcp://boiler:4840"},
		}
	}
	create := func(ctx context.Context) server.Server {
		GinkgoHelper()
		s := newServer()
		Expect(e.svc.NewWriter(tx).Create(ctx, &s)).To(Succeed())
		return s
	}
	retrieve := func(ctx context.Context, key server.Key) server.Server {
		GinkgoHelper()
		var s server.Server
		Expect(e.svc.NewRetrieve().
			Where(server.MatchKeys(key)).
			Entry(&s).
			Exec(ctx, tx)).To(Succeed())
		return s
	}
	retrieveStatus := func(ctx context.Context, key server.Key) server.Status {
		GinkgoHelper()
		var s server.Status
		Expect(e.status.NewRetrieve[server.StatusDetails]().
			Where(status.MatchOwners[server.StatusDetails](server.ResourceID(key))).
			Entry(&s).
			Exec(ctx, tx)).To(Succeed())
		return s
	}
	createSecret := func(ctx context.Context) secret.Key {
		GinkgoHelper()
		s := secret.Secret{Variant: secret.StoredSecret{
			Base: secret.Base{Name: "PLC password"},
		}}
		Expect(e.secret.NewWriter(tx).Set(ctx, &s, []byte("hunter2"))).To(Succeed())
		return s.Base().Key
	}

	Describe("OpenService", func() {
		DescribeTable("Should reject a config missing a required field",
			func(ctx SpecContext, field string, unset func(*server.ServiceConfig)) {
				cfg := server.ServiceConfig{
					DB:          db,
					Status:      e.status,
					Secret:      e.secret,
					Certificate: e.certs,
				}
				unset(&cfg)
				Expect(server.OpenService(ctx, cfg)).Error().
					To(MatchError(ContainSubstring(field + ": must be non-nil")))
			},
			Entry("db", "db", func(c *server.ServiceConfig) { c.DB = nil }),
			Entry("status", "status", func(c *server.ServiceConfig) { c.Status = nil }),
			Entry("secret", "secret", func(c *server.ServiceConfig) { c.Secret = nil }),
			Entry("certificate", "certificate", func(c *server.ServiceConfig) {
				c.Certificate = nil
			}),
		)

		It("Should let a later config override an earlier one", func() {
			cfg := server.ServiceConfig{DB: db}.Override(server.ServiceConfig{
				Status:      e.status,
				Secret:      e.secret,
				Certificate: e.certs,
			})
			Expect(cfg.DB).To(BeIdenticalTo(db))
			Expect(cfg.Status).To(BeIdenticalTo(e.status))
			Expect(cfg.Validate()).To(Succeed())
		})
	})

	Describe("Create", func() {
		It("Should mint a key, store the server, and write an unknown status", func(
			ctx SpecContext,
		) {
			s := create(ctx)
			Expect(s.Key).ToNot(Equal(uuid.Nil()))
			Expect(retrieve(ctx, s.Key).Name).To(Equal("Boiler PLC"))
			st := retrieveStatus(ctx, s.Key)
			Expect(st.Variant).To(Equal(status.VariantWarning))
			Expect(st.Message).To(Equal("Status unknown"))
			Expect(st.Details.Server).To(Equal(s.Key))
			Expect(st.Details.Rack).To(Equal(s.Rack))
		})

		It("Should keep a caller-set key", func(ctx SpecContext) {
			s := newServer()
			s.Key = uuid.New()
			Expect(e.svc.NewWriter(tx).Create(ctx, &s)).To(Succeed())
			Expect(retrieve(ctx, s.Key).Key).To(Equal(s.Key))
		})

		DescribeTable("Should reject an invalid server",
			func(ctx SpecContext, change func(*server.Server)) {
				s := newServer()
				change(&s)
				Expect(e.svc.NewWriter(tx).Create(ctx, &s)).
					To(MatchError(validate.ErrValidation))
			},
			Entry("no name", func(s *server.Server) { s.Name = "" }),
			Entry("no endpoint", func(s *server.Server) { s.Connection.Endpoint = "" }),
			Entry("a missing rack", func(s *server.Server) { s.Rack = 999 }),
			Entry("a missing password", func(s *server.Server) {
				s.Connection.Password = new(uuid.New())
			}),
			Entry("a missing certificate", func(s *server.Server) {
				s.Connection.ServerCertificate = new(uuid.New())
			}),
		)

		It("Should store a server that uses a password and a certificate", func(
			ctx SpecContext,
		) {
			pem, _ := certtestutil.NewPair()
			c := certificate.Certificate{Variant: certificate.StoredCertificate{
				Base: certificate.Base{Name: "PLC"},
				Pem:  pem,
			}}
			Expect(e.certs.NewWriter(tx).Set(ctx, &c, nil)).To(Succeed())
			s := newServer()
			s.Connection.Password = new(createSecret(ctx))
			s.Connection.ServerCertificate = new(c.Base().Key)
			Expect(e.svc.NewWriter(tx).Create(ctx, &s)).To(Succeed())
			Expect(retrieve(ctx, s.Key).Connection.ServerCertificate).
				To(Equal(s.Connection.ServerCertificate))
		})
	})

	Describe("Apply", func() {
		It("Should write the identity group and keep the channel maps", func(
			ctx SpecContext,
		) {
			s := newServer()
			s.ReadChannels = map[string]channel.Key{"ns=2;s=Temp": 5}
			Expect(e.svc.NewWriter(tx).Create(ctx, &s)).To(Succeed())
			Expect(e.svc.NewWriter(tx).Apply(ctx, s.Key, server.Update{
				Variant: server.IdentityUpdate{Identity: server.Identity{
					Rack:       s.Rack,
					Name:       "Renamed",
					Connection: server.Connection{Endpoint: "opc.tcp://other:4840"},
				}},
			})).To(Succeed())
			got := retrieve(ctx, s.Key)
			Expect(got.Name).To(Equal("Renamed"))
			Expect(got.Connection.Endpoint).To(Equal("opc.tcp://other:4840"))
			Expect(got.ReadChannels).To(Equal(s.ReadChannels))
			Expect(retrieveStatus(ctx, s.Key).Name).To(Equal("Renamed"))
		})

		It("Should write the configuration group and keep the connection", func(
			ctx SpecContext,
		) {
			s := create(ctx)
			Expect(e.svc.NewWriter(tx).Apply(ctx, s.Key, server.Update{
				Variant: server.ConfigurationUpdate{Configuration: server.Configuration{
					ReadIndexes:  []channel.Key{1},
					ReadChannels: map[string]channel.Key{"ns=2;s=Temp": 2},
					ReadFields: []server.ReadField{
						{NodeID: "ns=2;i=7", Field: "a", Channel: 3},
					},
					WriteChannels: map[string]channel.Key{"ns=2;s=Valve": 4},
				}},
			})).To(Succeed())
			got := retrieve(ctx, s.Key)
			Expect(got.Name).To(Equal(s.Name))
			Expect(got.Connection).To(Equal(s.Connection))
			Expect(got.ReadIndexes).To(ConsistOf(channel.Key(1)))
			Expect(
				got.WriteChannels,
			).To(HaveKeyWithValue("ns=2;s=Valve", channel.Key(4)))
		})

		It("Should move the status with the rack", func(ctx SpecContext) {
			s := create(ctx)
			other := rack.Rack{Name: "Other"}
			Expect(e.racks.NewWriter(tx).Create(ctx, &other)).To(Succeed())
			Expect(e.svc.NewWriter(tx).Apply(ctx, s.Key, server.Update{
				Variant: server.IdentityUpdate{Identity: server.Identity{
					Rack:       other.Key,
					Name:       s.Name,
					Connection: s.Connection,
				}},
			})).To(Succeed())
			Expect(retrieveStatus(ctx, s.Key).Details.Rack).To(Equal(other.Key))
		})

		DescribeTable("Should reject an invalid update",
			func(ctx SpecContext, update server.Update) {
				s := create(ctx)
				Expect(e.svc.NewWriter(tx).Apply(ctx, s.Key, update)).
					To(MatchError(validate.ErrValidation))
				Expect(retrieve(ctx, s.Key)).To(Equal(s))
			},
			Entry("no group", server.Update{}),
			Entry("an identity without a name", server.Update{
				Variant: server.IdentityUpdate{Identity: server.Identity{
					Rack:       1,
					Connection: server.Connection{Endpoint: "opc.tcp://plc:4840"},
				}},
			}),
			Entry("a field without a node", server.Update{
				Variant: server.ConfigurationUpdate{Configuration: server.Configuration{
					ReadFields: []server.ReadField{{Field: "a", Channel: 1}},
				}},
			}),
		)

		It("Should return not found for a missing server", func(ctx SpecContext) {
			Expect(e.svc.NewWriter(tx).Apply(ctx, uuid.New(), server.Update{
				Variant: server.ConfigurationUpdate{},
			})).To(MatchError(query.ErrNotFound))
		})
	})

	Describe("Delete", func() {
		It("Should remove the server and its status", func(ctx SpecContext) {
			s := create(ctx)
			Expect(e.svc.NewWriter(tx).Delete(ctx, s.Key)).To(Succeed())
			Expect(e.svc.NewRetrieve().
				Where(server.MatchKeys(s.Key)).
				Exists(ctx, tx)).To(BeFalse())
			var st server.Status
			Expect(e.status.NewRetrieve[server.StatusDetails]().
				Where(status.MatchOwners[server.StatusDetails](server.ResourceID(s.Key))).
				Entry(&st).
				Exec(ctx, tx)).To(MatchError(query.ErrNotFound))
		})

		It("Should be idempotent", func(ctx SpecContext) {
			Expect(e.svc.NewWriter(tx).Delete(ctx, uuid.New())).To(Succeed())
		})
	})

	It("Should refuse to delete a password that a server uses", func(ctx SpecContext) {
		s := newServer()
		s.Connection.Password = new(createSecret(ctx))
		Expect(e.svc.NewWriter(tx).Create(ctx, &s)).To(Succeed())
		Expect(e.secret.NewWriter(tx).Delete(ctx, *s.Connection.Password)).
			To(HaveOccurred())
	})

	It("Should refuse to delete a rack that holds a server", func(ctx SpecContext) {
		other := rack.Rack{Name: "Other"}
		Expect(e.racks.NewWriter(tx).Create(ctx, &other)).To(Succeed())
		s := newServer()
		s.Rack = other.Key
		Expect(e.svc.NewWriter(tx).Create(ctx, &s)).To(Succeed())
		Expect(e.racks.NewWriter(tx).Delete(ctx, nil, other.Key)).To(HaveOccurred())
	})

	Describe("Resources", func() {
		It("Should name a server by its resource ID", func(ctx SpecContext) {
			s := create(ctx)
			Expect(e.svc.Resources().Retrieve(ctx, tx, []resource.ID{s.ResourceID()})).
				To(Equal([]resource.Resource{{ID: s.ResourceID(), Name: s.Name}}))
		})

		It("Should find a server by its name", func(ctx SpecContext) {
			s := create(ctx)
			res := MustSucceed(e.svc.Resources().Search(
				ctx,
				tx,
				search.Query[resource.ID]{Term: "boiler", Limit: 10},
			))
			Expect(res.Keys).To(ContainElement(s.ResourceID()))
		})

		It("Should reject a resource ID with a malformed key", func(ctx SpecContext) {
			Expect(e.svc.Resources().Retrieve(ctx, tx, []resource.ID{{
				Type: resource.TypeOpcuaServer,
				Key:  "plc",
			}})).Error().To(MatchError(validate.ErrValidation))
		})
	})
})
