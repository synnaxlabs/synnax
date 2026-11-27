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
	"github.com/onsi/gomega/types"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/certificate"
	certtestutil "github.com/synnaxlabs/synnax/pkg/service/certificate/testutil"
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
			Connection: newConnection("opc.tcp://boiler:4840"),
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
	createCertificate := func(ctx context.Context, privateKey bool) certificate.Key {
		GinkgoHelper()
		certPEM, keyPEM := certtestutil.NewPair()
		var key []byte
		if privateKey {
			key = []byte(keyPEM)
		}
		c := certificate.Certificate{Variant: certificate.StoredCertificate{
			Base: certificate.Base{Name: "PLC"},
			Pem:  certPEM,
		}}
		Expect(e.certs.NewWriter(tx).Set(ctx, &c, key)).To(Succeed())
		return c.Base().Key
	}
	createFileCertificate := func(ctx context.Context, privateKey bool) certificate.Key {
		GinkgoHelper()
		var keyFile *string
		if privateKey {
			keyFile = new("plc.key")
		}
		c := certificate.Certificate{Variant: certificate.FileCertificate{
			Base:            certificate.Base{Name: "PLC file"},
			CertificatePath: "/etc/plc.pem",
			PrivateKeyFile:  keyFile,
			Rack:            e.racks.EmbeddedKey,
		}}
		Expect(e.certs.NewWriter(tx).Set(ctx, &c, nil)).To(Succeed())
		return c.Base().Key
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

		DescribeTable(
			"Should reject an invalid server",
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
				s.Connection.Auth = server.Auth{Variant: server.UsernameAuth{
					Username: "op",
					Password: new(uuid.New()),
				}}
			}),
			Entry("a missing pinned certificate", func(s *server.Server) {
				s.Connection.Security = signed(pinned(uuid.New()))
			}),
			Entry("a node named by its namespace index", func(s *server.Server) {
				s.Reads = []server.Read{{Node: "ns=2;s=Temp", Index: 1, Channel: 2}}
			}),
			Entry("a malformed node", func(s *server.Server) {
				s.Writes = []server.Write{{Node: "nsu=urn:plc;x=Valve", Channel: 2}}
			}),
			Entry(
				"a byte string node that is not canonical base64",
				func(s *server.Server) {
					s.Writes = []server.Write{{Node: "nsu=urn:plc;b=AQI", Channel: 2}}
				},
			),
			Entry("two reads of one node, field, and index", func(s *server.Server) {
				s.Reads = []server.Read{
					{Node: "nsu=urn:plc;s=Temp", Index: 1, Channel: 2},
					{Node: "NSU=urn:plc;S=Temp", Index: 1, Channel: 3},
				}
			}),
			Entry("two writes to one node", func(s *server.Server) {
				s.Writes = []server.Write{
					{Node: "i=85", Channel: 2},
					{Node: "ns=0;i=85", Channel: 3},
				}
			}),
		)

		It("Should store the node IDs in the form the Driver writes", func(
			ctx SpecContext,
		) {
			s := newServer()
			s.Reads = []server.Read{
				{Node: "NSU=urn:plc%3Bline;S=Temp", Index: 1, Channel: 2},
				{Node: "nsu=urn:plc;s=Temp", Field: "a.b", Index: 1, Channel: 3},
				{Node: "ns=0;I=0085", Index: 1, Channel: 4},
				{Node: "nsu=urn:plc;B=AQID", Index: 1, Channel: 6},
			}
			s.Writes = []server.Write{
				{
					Node:    "nsu=http://opcfoundation.org/UA/;g=0E4B8F4C-6D2E-4B5A-9A63-2F9D8C7B1A00",
					Channel: 5,
				},
			}
			Expect(e.svc.NewWriter(tx).Create(ctx, &s)).To(Succeed())
			got := retrieve(ctx, s.Key)
			Expect(got.Reads).To(Equal([]server.Read{
				{Node: "nsu=urn:plc%3Bline;s=Temp", Index: 1, Channel: 2},
				{Node: "nsu=urn:plc;s=Temp", Field: "a.b", Index: 1, Channel: 3},
				{Node: "i=85", Index: 1, Channel: 4},
				{Node: "nsu=urn:plc;b=AQID", Index: 1, Channel: 6},
			}))
			Expect(got.Writes[0].Node).
				To(Equal("g=0e4b8f4c-6d2e-4b5a-9a63-2f9d8c7b1a00"))
		})

		It("Should store a server that signs and logs in with a password", func(
			ctx SpecContext,
		) {
			s := newServer()
			s.Connection.Auth = server.Auth{Variant: server.UsernameAuth{
				Username: "op",
				Password: new(createSecret(ctx)),
			}}
			s.Connection.Security = signed(pinned(createCertificate(ctx, false)))
			Expect(e.svc.NewWriter(tx).Create(ctx, &s)).To(Succeed())
			Expect(retrieve(ctx, s.Key).Connection).To(Equal(s.Connection))
		})

		It("Should store a certificate login whose certificate has a private key", func(
			ctx SpecContext,
		) {
			s := newServer()
			s.Connection.Auth = server.Auth{Variant: server.CertificateAuth{
				Certificate: createCertificate(ctx, true),
			}}
			Expect(e.svc.NewWriter(tx).Create(ctx, &s)).To(Succeed())
		})

		DescribeTable(
			"Should check the private key of a file login certificate",
			func(ctx SpecContext, privateKey bool, matcher types.GomegaMatcher) {
				s := newServer()
				s.Connection.Auth = server.Auth{Variant: server.CertificateAuth{
					Certificate: createFileCertificate(ctx, privateKey),
				}}
				Expect(e.svc.NewWriter(tx).Create(ctx, &s)).To(matcher)
			},
			Entry("with a key file", true, Succeed()),
			Entry(
				"without a key file",
				false,
				MatchError(ContainSubstring("has no private key")),
			),
		)

		It("Should refuse a client certificate that has no private key", func(
			ctx SpecContext,
		) {
			key := createCertificate(ctx, false)
			var channel server.SecuredChannel
			channel.SetDefaults()
			channel.Policy = server.SecurityPolicyBasic256Sha256
			channel.ClientCertificate = &key
			s := newServer()
			s.Connection.Security = server.Security{
				Variant: server.SignAndEncryptSecurity{SecuredChannel: channel},
			}
			Expect(e.svc.NewWriter(tx).Create(ctx, &s)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("client_certificate")),
				MatchError(ContainSubstring("has no private key")),
			))
		})

		It(
			"Should refuse a certificate login whose certificate has no private key",
			func(
				ctx SpecContext,
			) {
				s := newServer()
				s.Connection.Auth = server.Auth{Variant: server.CertificateAuth{
					Certificate: createCertificate(ctx, false),
				}}
				Expect(e.svc.NewWriter(tx).Create(ctx, &s)).To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring("has no private key")),
				))
			},
		)
	})

	Describe("Apply", func() {
		It("Should write the identity group and keep the bindings", func(
			ctx SpecContext,
		) {
			s := newServer()
			s.Reads = []server.Read{{Node: "nsu=urn:plc;s=Temp", Index: 1, Channel: 5}}
			Expect(e.svc.NewWriter(tx).Create(ctx, &s)).To(Succeed())
			Expect(e.svc.NewWriter(tx).Apply(ctx, s.Key, server.Update{
				Variant: server.IdentityUpdate{Identity: server.Identity{
					Rack:       s.Rack,
					Name:       "Renamed",
					Connection: newConnection("opc.tcp://other:4840"),
				}},
			})).To(Succeed())
			got := retrieve(ctx, s.Key)
			Expect(got.Name).To(Equal("Renamed"))
			Expect(got.Connection.Endpoint).To(Equal("opc.tcp://other:4840"))
			Expect(got.Reads).To(Equal(s.Reads))
			Expect(retrieveStatus(ctx, s.Key).Name).To(Equal("Renamed"))
		})

		It("Should write the configuration group and keep the connection", func(
			ctx SpecContext,
		) {
			s := create(ctx)
			Expect(e.svc.NewWriter(tx).Apply(ctx, s.Key, server.Update{
				Variant: server.ConfigurationUpdate{Configuration: server.Configuration{
					Reads: []server.Read{
						{Node: "NSU=urn:plc;s=Temp", Field: "a", Index: 1, Channel: 2},
					},
					Writes: []server.Write{{Node: "nsu=urn:plc;s=Valve", Channel: 4}},
				}},
			})).To(Succeed())
			got := retrieve(ctx, s.Key)
			Expect(got.Name).To(Equal(s.Name))
			Expect(got.Connection).To(Equal(s.Connection))
			Expect(got.Reads[0].Node).To(Equal("nsu=urn:plc;s=Temp"))
			Expect(got.Writes).To(HaveLen(1))
		})

		It(
			"Should refuse an identity whose client certificate has no private key",
			func(
				ctx SpecContext,
			) {
				s := create(ctx)
				key := createCertificate(ctx, false)
				var channel server.SecuredChannel
				channel.SetDefaults()
				channel.Policy = server.SecurityPolicyBasic256Sha256
				channel.ClientCertificate = &key
				conn := s.Connection
				conn.Security = server.Security{
					Variant: server.SignSecurity{SecuredChannel: channel},
				}
				Expect(e.svc.NewWriter(tx).Apply(ctx, s.Key, server.Update{
					Variant: server.IdentityUpdate{Identity: server.Identity{
						Rack:       s.Rack,
						Name:       s.Name,
						Connection: conn,
					}},
				})).To(MatchError(ContainSubstring("has no private key")))
			},
		)

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
					Connection: newConnection("opc.tcp://plc:4840"),
				}},
			}),
			Entry("a write to a node named by its namespace index", server.Update{
				Variant: server.ConfigurationUpdate{Configuration: server.Configuration{
					Writes: []server.Write{{Node: "ns=3;i=7", Channel: 1}},
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
		password := createSecret(ctx)
		s.Connection.Auth = server.Auth{Variant: server.UsernameAuth{
			Username: "op",
			Password: &password,
		}}
		Expect(e.svc.NewWriter(tx).Create(ctx, &s)).To(Succeed())
		Expect(e.secret.NewWriter(tx).Delete(ctx, password)).
			To(MatchError(ContainSubstring(
				"because opcua_server %s references it through connection.auth.password",
				s.Key,
			)))
	})

	DescribeTable(
		"Should refuse to delete a certificate that a server uses",
		func(ctx SpecContext, privateKey bool, use func(*server.Connection, certificate.Key)) {
			c := createCertificate(ctx, privateKey)
			s := newServer()
			use(&s.Connection, c)
			Expect(e.svc.NewWriter(tx).Create(ctx, &s)).To(Succeed())
			Expect(e.certs.NewWriter(tx).Delete(ctx, c)).
				To(MatchError(ContainSubstring(
					"because opcua_server %s references it through "+
						"connection.certificates",
					s.Key,
				)))
		},
		Entry("for a login", true, func(c *server.Connection, k certificate.Key) {
			c.Auth = server.Auth{Variant: server.CertificateAuth{Certificate: k}}
		}),
		Entry(
			"as the client certificate",
			true,
			func(c *server.Connection, k certificate.Key) {
				var channel server.SecuredChannel
				channel.SetDefaults()
				channel.Policy = server.SecurityPolicyBasic256Sha256
				channel.ClientCertificate = &k
				c.Security = server.Security{
					Variant: server.SignSecurity{SecuredChannel: channel},
				}
			},
		),
		Entry(
			"as the pinned server certificate",
			false,
			func(c *server.Connection, k certificate.Key) {
				c.Security = signed(pinned(k))
			},
		),
	)

	It("Should reject a password file of another rack", func(ctx SpecContext) {
		other := rack.Rack{Name: "Other"}
		Expect(e.racks.NewWriter(tx).Create(ctx, &other)).To(Succeed())
		file := secret.Secret{Variant: secret.FileSecret{
			Base:     secret.Base{Name: "PLC password"},
			FileName: "plc",
			Rack:     other.Key,
		}}
		Expect(e.secret.NewWriter(tx).Set(ctx, &file, nil)).To(Succeed())
		s := newServer()
		s.Connection.Auth = server.Auth{Variant: server.UsernameAuth{
			Username: "op",
			Password: new(file.Base().Key),
		}}
		Expect(e.svc.NewWriter(tx).Create(ctx, &s)).To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring(
				"is a file of rack %d, not of rack %d",
				other.Key,
				s.Rack,
			)),
		))
	})

	It("Should refuse to delete a rack that holds a server", func(ctx SpecContext) {
		other := rack.Rack{Name: "Other"}
		Expect(e.racks.NewWriter(tx).Create(ctx, &other)).To(Succeed())
		s := newServer()
		s.Rack = other.Key
		Expect(e.svc.NewWriter(tx).Create(ctx, &s)).To(Succeed())
		Expect(e.racks.NewWriter(tx).Delete(ctx, nil, other.Key)).
			To(MatchError(ContainSubstring("because opcua_server %s references it", s.Key)))
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
