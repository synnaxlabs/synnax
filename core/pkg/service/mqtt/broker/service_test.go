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
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/certificate"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/mqtt/broker"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	racktestutil "github.com/synnaxlabs/synnax/pkg/service/rack/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/secret"
	secrettestutil "github.com/synnaxlabs/synnax/pkg/service/secret/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/search"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Service", func() {
	var (
		stat    *status.Service
		rackSvc *rack.Service
		secrets *secret.Service
		certs   *certificate.Service
		svc     *broker.Service
		tx      gorp.Tx
	)
	BeforeEach(func(ctx SpecContext) {
		MustOpen(group.OpenService(ctx, group.ServiceConfig{DB: db}))
		MustOpen(label.OpenService(ctx, label.ServiceConfig{DB: db}))
		stat = MustOpen(status.OpenService(ctx, status.ServiceConfig{DB: db}))
		roleSvc := MustOpen(role.OpenService(ctx, role.ServiceConfig{DB: db}))
		rackSvc = MustOpen(rack.OpenService(ctx, rack.ServiceConfig{
			DB:         db,
			HostKey:    1,
			Status:     stat,
			DriverRole: MustSucceed(racktestutil.CreateDriverRole(ctx, db, roleSvc)),
		}))
		secrets = secrettestutil.OpenService(
			ctx,
			db,
			secret.ServiceConfig{EncryptionKey: secretsKey},
		)
		certs = MustOpen(certificate.OpenService(ctx, certificate.ServiceConfig{
			DB:      db,
			Secret:  secrets,
			Status:  stat,
			Rack:    rackSvc.EmbeddedKey,
			HostKey: 1,
		}))
		svc = MustOpen(broker.OpenService(ctx, broker.ServiceConfig{
			DB:          db,
			Status:      stat,
			Secret:      secrets,
			Certificate: certs,
			Rack:        rackSvc,
		}))
		tx = DeferClose(db.OpenTx())
		Expect(rackSvc.NewWriter(tx).Apply(
			ctx,
			rackSvc.EmbeddedKey,
			rack.NewSetAction(rack.SetPayload{
				Path:  []string{"integrations"},
				Value: []string{broker.Integration},
			}),
		)).To(Succeed())
	})

	createRack := func(ctx context.Context, integrations ...string) rack.Key {
		GinkgoHelper()
		r := rack.Rack{Name: "Other", Integrations: integrations}
		Expect(rackSvc.NewWriter(tx).Create(ctx, &r)).To(Succeed())
		return r.Key
	}
	newBroker := func() broker.Broker {
		var b broker.Broker
		b.SetDefaults()
		b.Rack = rackSvc.EmbeddedKey
		b.Name = "Plant Broker"
		b.Host = "mqtt.example.com"
		return b
	}
	plainRead := func(topic, pointer string, message uuid.UUID) broker.Read {
		return broker.Read{Variant: broker.PlainRead{
			Topic:   topic,
			Message: message,
			Pointer: pointer,
			Index:   1,
			Channel: 2,
		}}
	}
	sparkplugRead := func(device, tag string) broker.Read {
		return broker.Read{Variant: broker.SparkplugRead{
			Group:    "Plant",
			EdgeNode: "Line1",
			Device:   device,
			Tag:      tag,
			Index:    1,
			Channel:  2,
		}}
	}
	plainWrite := func(topic string, field uuid.UUID) broker.Write {
		return broker.Write{Variant: broker.PlainWrite{
			Topic:   topic,
			Field:   field,
			Channel: 3,
		}}
	}
	sparkplugWrite := func(tag string) broker.Write {
		return broker.Write{Variant: broker.SparkplugWrite{
			Group:    "Plant",
			EdgeNode: "Line1",
			Tag:      tag,
			Channel:  3,
		}}
	}
	edgeCommand := func(tag string) broker.EdgeCommand {
		return broker.EdgeCommand{
			Group:    "Plant",
			EdgeNode: "Synnax",
			Tag:      tag,
			Channel:  4,
		}
	}
	createSecret := func(ctx context.Context) secret.Key {
		GinkgoHelper()
		s := secret.Secret{Variant: secret.StoredSecret{
			Base: secret.Base{Name: "Broker password"},
		}}
		Expect(secrets.NewWriter(tx).Set(ctx, &s, []byte("hunter2"))).To(Succeed())
		return s.Base().Key
	}
	createFileCertificate := func(
		ctx context.Context,
		r rack.Key,
		privateKey bool,
	) certificate.Key {
		GinkgoHelper()
		f := certificate.FileCertificate{
			Name:            "Broker CA",
			CertificatePath: "/certs/ca.pem",
			Rack:            r,
		}
		if privateKey {
			f.PrivateKeyFile = new("client.pem")
		}
		c := certificate.Certificate{Variant: f}
		Expect(certs.NewWriter(tx).Set(ctx, &c, nil)).To(Succeed())
		return c.Base().Key
	}
	tls := func(ca, client *certificate.Key) broker.Transport {
		return broker.Transport{Variant: broker.TlsTransport{
			CaCertificate:     ca,
			ClientCertificate: client,
		}}
	}
	create := func(ctx context.Context) broker.Broker {
		GinkgoHelper()
		b := newBroker()
		Expect(svc.NewWriter(tx).Create(ctx, &b)).To(Succeed())
		return b
	}
	retrieve := func(ctx context.Context, key broker.Key) broker.Broker {
		GinkgoHelper()
		var b broker.Broker
		Expect(svc.NewRetrieve().
			Where(broker.MatchKeys(key)).
			Entry(&b).
			Exec(ctx, tx)).To(Succeed())
		return b
	}
	retrieveStatus := func(ctx context.Context, key broker.Key) broker.Status {
		GinkgoHelper()
		var s broker.Status
		Expect(stat.NewRetrieve[broker.StatusDetails]().
			Where(status.MatchOwners[broker.StatusDetails](broker.ResourceID(key))).
			Entry(&s).
			Exec(ctx, tx)).To(Succeed())
		return s
	}
	identity := func(b broker.Broker) broker.Identity {
		return broker.Identity{
			Rack:      b.Rack,
			Name:      b.Name,
			Host:      b.Host,
			Port:      b.Port,
			Transport: b.Transport,
			Auth:      b.Auth,
			ClientID:  b.ClientID,
			KeepAlive: b.KeepAlive,
			Sparkplug: b.Sparkplug,
		}
	}

	Describe("OpenService", func() {
		DescribeTable("Should reject a config missing a required field",
			func(ctx SpecContext, field string, unset func(*broker.ServiceConfig)) {
				cfg := broker.ServiceConfig{
					DB:          db,
					Status:      stat,
					Secret:      secrets,
					Certificate: certs,
					Rack:        rackSvc,
				}
				unset(&cfg)
				Expect(broker.OpenService(ctx, cfg)).Error().
					To(MatchError(ContainSubstring(field + ": must be non-nil")))
			},
			Entry("db", "db", func(c *broker.ServiceConfig) { c.DB = nil }),
			Entry("status", "status", func(c *broker.ServiceConfig) { c.Status = nil }),
			Entry("secret", "secret", func(c *broker.ServiceConfig) { c.Secret = nil }),
			Entry("certificate", "certificate", func(c *broker.ServiceConfig) {
				c.Certificate = nil
			}),
			Entry("rack", "rack", func(c *broker.ServiceConfig) { c.Rack = nil }),
		)

		It("Should let a later config override an earlier one", func() {
			cfg := broker.ServiceConfig{DB: db}.Override(broker.ServiceConfig{
				Status:      stat,
				Secret:      secrets,
				Certificate: certs,
				Rack:        rackSvc,
			})
			Expect(cfg.DB).To(BeIdenticalTo(db))
			Expect(cfg.Rack).To(BeIdenticalTo(rackSvc))
			Expect(cfg.Validate()).To(Succeed())
		})
	})

	Describe("Create", func() {
		It("Should mint a key, store the broker, and write an unknown status", func(
			ctx SpecContext,
		) {
			b := create(ctx)
			Expect(b.Key).ToNot(Equal(uuid.Nil()))
			Expect(retrieve(ctx, b.Key)).To(Equal(b))
			st := retrieveStatus(ctx, b.Key)
			Expect(st.Variant).To(Equal(status.VariantWarning))
			Expect(st.Message).To(Equal("Status unknown"))
			Expect(st.Name).To(Equal(b.Name))
			Expect(st.Details.Broker).To(Equal(b.Key))
			Expect(st.Details.Rack).To(Equal(b.Rack))
			Expect(st.Details.Connected).To(BeFalse())
		})

		It("Should keep a caller-set key", func(ctx SpecContext) {
			b := newBroker()
			b.Key = uuid.New()
			Expect(svc.NewWriter(tx).Create(ctx, &b)).To(Succeed())
			Expect(retrieve(ctx, b.Key).Key).To(Equal(b.Key))
		})

		It("Should refuse a key that a stored broker holds", func(ctx SpecContext) {
			b := create(ctx)
			again := newBroker()
			again.Key = b.Key
			again.Name = "Other Broker"
			Expect(svc.NewWriter(tx).Create(ctx, &again)).
				To(MatchError(validate.ErrValidation))
			Expect(retrieve(ctx, b.Key).Name).To(Equal(b.Name))
		})

		It("Should store every kind of binding", func(ctx SpecContext) {
			b := newBroker()
			b.Sparkplug = broker.Sparkplug{
				Mode: broker.SparkplugMode{
					Variant: broker.PrimarySparkplugMode{HostID: "scada"},
				},
				Groups: []string{"Plant"},
			}
			b.Reads = []broker.Read{
				plainRead("plant/temp", "/t", uuid.New()),
				sparkplugRead("", "Pressure"),
			}
			b.Writes = []broker.Write{
				plainWrite("plant/valve", uuid.New()),
				sparkplugWrite("Speed"),
			}
			b.EdgeCommands = []broker.EdgeCommand{edgeCommand("Setpoint")}
			Expect(svc.NewWriter(tx).Create(ctx, &b)).To(Succeed())
			Expect(retrieve(ctx, b.Key)).To(Equal(b))
		})

		It("Should accept bindings that differ in one part of their key", func(
			ctx SpecContext,
		) {
			b := newBroker()
			message := uuid.New()
			b.Reads = []broker.Read{
				plainRead("plant/temp", "/t", message),
				plainRead("plant/temp", "/p", message),
				plainRead("plant/temp", "/t", uuid.New()),
				plainRead("plant/other", "/t", message),
				sparkplugRead("", "Pressure"),
				sparkplugRead("Pump", "Pressure"),
			}
			field := uuid.New()
			b.Writes = []broker.Write{
				plainWrite("plant/valve", field),
				plainWrite("plant/valve", uuid.New()),
				plainWrite("plant/pump", field),
				sparkplugWrite("Speed"),
				sparkplugWrite("Mode"),
			}
			b.EdgeCommands = []broker.EdgeCommand{
				edgeCommand("Setpoint"),
				edgeCommand("Mode"),
			}
			Expect(svc.NewWriter(tx).Create(ctx, &b)).To(Succeed())
		})

		DescribeTable(
			"Should reject an invalid broker",
			func(ctx SpecContext, change func(*broker.Broker), path string) {
				b := newBroker()
				change(&b)
				Expect(svc.NewWriter(tx).Create(ctx, &b)).To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring(path)),
				))
				Expect(svc.NewRetrieve().
					Where(broker.MatchKeys(b.Key)).
					Exists(ctx, tx)).To(BeFalse())
			},
			Entry("no name", func(b *broker.Broker) { b.Name = "" }, "name"),
			Entry("no host", func(b *broker.Broker) { b.Host = "" }, "host"),
			Entry("a keep alive of a fraction of a second", func(b *broker.Broker) {
				b.KeepAlive = 1500 * telem.Millisecond
			}, "keep_alive"),
			Entry("a Sparkplug group with a wildcard", func(b *broker.Broker) {
				b.Sparkplug.Groups = []string{"Plant", "+"}
			}, "groups"),
			Entry("an empty Sparkplug group", func(b *broker.Broker) {
				b.Sparkplug.Groups = []string{""}
			}, "groups"),
			Entry("a duplicate plain read", func(b *broker.Broker) {
				m := uuid.New()
				b.Reads = []broker.Read{
					plainRead("plant/temp", "/t", m),
					plainRead("plant/temp", "/t", m),
				}
			}, "reads.1"),
			Entry("a duplicate Sparkplug read", func(b *broker.Broker) {
				b.Reads = []broker.Read{
					sparkplugRead("Pump", "Pressure"),
					sparkplugRead("Pump", "Pressure"),
				}
			}, "reads.1"),
			Entry("a duplicate plain write", func(b *broker.Broker) {
				f := uuid.New()
				b.Writes = []broker.Write{
					plainWrite("plant/valve", f),
					plainWrite("plant/valve", f),
				}
			}, "writes.1"),
			Entry("a duplicate Sparkplug write", func(b *broker.Broker) {
				b.Writes = []broker.Write{
					sparkplugWrite("Speed"),
					sparkplugWrite("Speed"),
				}
			}, "writes.1"),
			Entry("a duplicate edge command", func(b *broker.Broker) {
				b.EdgeCommands = []broker.EdgeCommand{
					edgeCommand("Setpoint"),
					edgeCommand("Setpoint"),
				}
			}, "edge_commands.1"),
		)

		It("Should reject a rack that does not run MQTT", func(ctx SpecContext) {
			b := newBroker()
			b.Rack = createRack(ctx, "ni")
			Expect(svc.NewWriter(tx).Create(ctx, &b)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("does not run the mqtt integration")),
			))
		})

		It("Should reject a missing rack", func(ctx SpecContext) {
			b := newBroker()
			b.Rack = 999
			Expect(svc.NewWriter(tx).Create(ctx, &b)).
				To(MatchError(query.ErrNotFound))
		})
	})

	Describe("Certificates", func() {
		It("Should store TLS with no certificates", func(ctx SpecContext) {
			b := newBroker()
			b.Transport = tls(nil, nil)
			Expect(svc.NewWriter(tx).Create(ctx, &b)).To(Succeed())
			Expect(retrieve(ctx, b.Key).Transport).To(Equal(b.Transport))
		})

		It("Should store a CA and a client certificate of the rack", func(
			ctx SpecContext,
		) {
			ca := createFileCertificate(ctx, rackSvc.EmbeddedKey, false)
			client := createFileCertificate(ctx, rackSvc.EmbeddedKey, true)
			b := newBroker()
			b.Transport = tls(&ca, &client)
			Expect(svc.NewWriter(tx).Create(ctx, &b)).To(Succeed())
			Expect(retrieve(ctx, b.Key).Transport).To(Equal(b.Transport))
		})

		It("Should reject a file certificate of another rack", func(ctx SpecContext) {
			other := createRack(ctx, broker.Integration)
			ca := createFileCertificate(ctx, other, false)
			b := newBroker()
			b.Transport = tls(&ca, nil)
			Expect(svc.NewWriter(tx).Create(ctx, &b)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(
					"is a file of rack %d, not of rack %d",
					other,
					b.Rack,
				)),
			))
		})

		It("Should reject a client certificate with no private key", func(
			ctx SpecContext,
		) {
			client := createFileCertificate(ctx, rackSvc.EmbeddedKey, false)
			b := newBroker()
			b.Transport = tls(nil, &client)
			Expect(svc.NewWriter(tx).Create(ctx, &b)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("holds no private key")),
			))
		})

		It("Should reject a missing client certificate", func(ctx SpecContext) {
			b := newBroker()
			b.Transport = tls(nil, new(uuid.New()))
			Expect(svc.NewWriter(tx).Create(ctx, &b)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("transport.client_certificate")),
			))
		})

		It("Should refuse to delete a certificate that a broker uses", func(
			ctx SpecContext,
		) {
			ca := createFileCertificate(ctx, rackSvc.EmbeddedKey, false)
			b := newBroker()
			b.Transport = tls(&ca, nil)
			Expect(svc.NewWriter(tx).Create(ctx, &b)).To(Succeed())
			Expect(certs.NewWriter(tx).Delete(ctx, ca)).To(MatchError(
				ContainSubstring("because mqtt_broker %s references it", b.Key),
			))
		})
	})

	Describe("Secrets", func() {
		It("Should store a user name auth whose password names a secret", func(
			ctx SpecContext,
		) {
			b := newBroker()
			b.Auth = broker.Auth{Variant: broker.UsernameAuth{
				Username: "admin",
				Password: new(createSecret(ctx)),
			}}
			Expect(svc.NewWriter(tx).Create(ctx, &b)).To(Succeed())
			Expect(retrieve(ctx, b.Key).Auth).To(Equal(b.Auth))
		})

		It("Should refuse to delete a password that a broker uses", func(
			ctx SpecContext,
		) {
			key := createSecret(ctx)
			b := newBroker()
			b.Auth = broker.Auth{Variant: broker.UsernameAuth{
				Username: "admin",
				Password: &key,
			}}
			Expect(svc.NewWriter(tx).Create(ctx, &b)).To(Succeed())
			Expect(secrets.NewWriter(tx).Delete(ctx, key)).To(MatchError(
				ContainSubstring(
					"because mqtt_broker %s references it through auth.password",
					b.Key,
				),
			))
		})

		It("Should reject a missing password secret", func(ctx SpecContext) {
			b := newBroker()
			b.Auth = broker.Auth{Variant: broker.UsernameAuth{
				Username: "admin",
				Password: new(uuid.New()),
			}}
			Expect(svc.NewWriter(tx).Create(ctx, &b)).
				To(MatchError(validate.ErrValidation))
		})
	})

	Describe("Apply", func() {
		It("Should write the identity group and keep the bindings", func(
			ctx SpecContext,
		) {
			b := newBroker()
			b.Reads = []broker.Read{sparkplugRead("", "Pressure")}
			Expect(svc.NewWriter(tx).Create(ctx, &b)).To(Succeed())
			update := identity(b)
			update.Name = "Renamed"
			update.Port = new(uint16(8883))
			update.ClientID = new("plant-core")
			update.KeepAlive = 60 * telem.Second
			Expect(svc.NewWriter(tx).Apply(ctx, b.Key, broker.Update{
				Variant: broker.IdentityUpdate{Identity: update},
			})).To(Succeed())
			got := retrieve(ctx, b.Key)
			Expect(got.Name).To(Equal("Renamed"))
			Expect(*got.Port).To(Equal(uint16(8883)))
			Expect(*got.ClientID).To(Equal("plant-core"))
			Expect(got.KeepAlive).To(Equal(60 * telem.Second))
			Expect(got.Reads).To(Equal(b.Reads))
			Expect(retrieveStatus(ctx, b.Key).Name).To(Equal("Renamed"))
		})

		It("Should write the configuration group and keep the connection", func(
			ctx SpecContext,
		) {
			b := create(ctx)
			configuration := broker.Configuration{
				Reads:        []broker.Read{sparkplugRead("", "Pressure")},
				Writes:       []broker.Write{sparkplugWrite("Speed")},
				EdgeCommands: []broker.EdgeCommand{edgeCommand("Setpoint")},
			}
			Expect(svc.NewWriter(tx).Apply(ctx, b.Key, broker.Update{
				Variant: broker.ConfigurationUpdate{Configuration: configuration},
			})).To(Succeed())
			got := retrieve(ctx, b.Key)
			Expect(identity(got)).To(Equal(identity(b)))
			Expect(got.Reads).To(Equal(configuration.Reads))
			Expect(got.Writes).To(Equal(configuration.Writes))
			Expect(got.EdgeCommands).To(Equal(configuration.EdgeCommands))
		})

		It("Should move the status with the rack", func(ctx SpecContext) {
			b := create(ctx)
			update := identity(b)
			update.Rack = createRack(ctx, broker.Integration)
			Expect(svc.NewWriter(tx).Apply(ctx, b.Key, broker.Update{
				Variant: broker.IdentityUpdate{Identity: update},
			})).To(Succeed())
			Expect(retrieveStatus(ctx, b.Key).Details.Rack).To(Equal(update.Rack))
		})

		It("Should reject a move to a rack of which a certificate is not a file", func(
			ctx SpecContext,
		) {
			ca := createFileCertificate(ctx, rackSvc.EmbeddedKey, false)
			b := newBroker()
			b.Transport = tls(&ca, nil)
			Expect(svc.NewWriter(tx).Create(ctx, &b)).To(Succeed())
			update := identity(b)
			update.Rack = createRack(ctx, broker.Integration)
			Expect(svc.NewWriter(tx).Apply(ctx, b.Key, broker.Update{
				Variant: broker.IdentityUpdate{Identity: update},
			})).To(MatchError(validate.ErrValidation))
			Expect(retrieve(ctx, b.Key).Rack).To(Equal(b.Rack))
		})

		DescribeTable(
			"Should reject an invalid update",
			func(ctx SpecContext, update func(broker.Broker) broker.Update) {
				b := create(ctx)
				Expect(svc.NewWriter(tx).Apply(ctx, b.Key, update(b))).
					To(MatchError(validate.ErrValidation))
				Expect(retrieve(ctx, b.Key)).To(Equal(b))
			},
			Entry("no group", func(broker.Broker) broker.Update {
				return broker.Update{}
			}),
			Entry("an identity without a name", func(b broker.Broker) broker.Update {
				i := identity(b)
				i.Name = ""
				return broker.Update{Variant: broker.IdentityUpdate{Identity: i}}
			}),
			Entry("an identity with a fractional keep alive", func(
				b broker.Broker,
			) broker.Update {
				i := identity(b)
				i.KeepAlive = 2500 * telem.Millisecond
				return broker.Update{Variant: broker.IdentityUpdate{Identity: i}}
			}),
			Entry("a configuration with a duplicate edge command", func(
				broker.Broker,
			) broker.Update {
				return broker.Update{Variant: broker.ConfigurationUpdate{
					Configuration: broker.Configuration{
						EdgeCommands: []broker.EdgeCommand{
							edgeCommand("Setpoint"),
							edgeCommand("Setpoint"),
						},
					},
				}}
			}),
		)

		It("Should reject a move to a rack that does not run MQTT", func(
			ctx SpecContext,
		) {
			b := create(ctx)
			update := identity(b)
			update.Rack = createRack(ctx)
			Expect(svc.NewWriter(tx).Apply(ctx, b.Key, broker.Update{
				Variant: broker.IdentityUpdate{Identity: update},
			})).To(MatchError(validate.ErrValidation))
			Expect(retrieve(ctx, b.Key).Rack).To(Equal(b.Rack))
		})

		It("Should return not found for a missing broker", func(ctx SpecContext) {
			Expect(svc.NewWriter(tx).Apply(ctx, uuid.New(), broker.Update{
				Variant: broker.ConfigurationUpdate{},
			})).To(MatchError(query.ErrNotFound))
		})
	})

	Describe("Delete", func() {
		It("Should remove the broker and its status", func(ctx SpecContext) {
			b := create(ctx)
			Expect(svc.NewWriter(tx).Delete(ctx, b.Key)).To(Succeed())
			Expect(svc.NewRetrieve().
				Where(broker.MatchKeys(b.Key)).
				Exists(ctx, tx)).To(BeFalse())
			var st broker.Status
			Expect(stat.NewRetrieve[broker.StatusDetails]().
				Where(status.MatchOwners[broker.StatusDetails](broker.ResourceID(b.Key))).
				Entry(&st).
				Exec(ctx, tx)).To(MatchError(query.ErrNotFound))
		})

		It("Should be idempotent", func(ctx SpecContext) {
			Expect(svc.NewWriter(tx).Delete(ctx, uuid.New())).To(Succeed())
		})
	})

	It("Should refuse to delete a rack that holds a broker", func(ctx SpecContext) {
		b := newBroker()
		b.Rack = createRack(ctx, broker.Integration)
		Expect(svc.NewWriter(tx).Create(ctx, &b)).To(Succeed())
		Expect(rackSvc.NewWriter(tx).Delete(ctx, nil, b.Rack)).To(MatchError(
			ContainSubstring("because mqtt_broker %s references it", b.Key),
		))
		Expect(retrieve(ctx, b.Key).Rack).To(Equal(b.Rack))
	})

	Describe("Resources", func() {
		It("Should name a broker by its resource ID", func(ctx SpecContext) {
			b := create(ctx)
			Expect(svc.Resources().Retrieve(ctx, tx, []resource.ID{b.ResourceID()})).
				To(Equal([]resource.Resource{{ID: b.ResourceID(), Name: b.Name}}))
		})

		It("Should find a broker by its name", func(ctx SpecContext) {
			b := create(ctx)
			res := MustSucceed(svc.Resources().Search(
				ctx,
				tx,
				search.Query[resource.ID]{Term: "plant", Limit: 10},
			))
			Expect(res.Keys).To(ContainElement(b.ResourceID()))
		})

		It("Should reject a resource ID with a malformed key", func(ctx SpecContext) {
			Expect(svc.Resources().Retrieve(ctx, tx, []resource.ID{{
				Type: resource.TypeMqttBroker,
				Key:  "broker",
			}})).Error().To(MatchError(validate.ErrValidation))
		})
	})
})
