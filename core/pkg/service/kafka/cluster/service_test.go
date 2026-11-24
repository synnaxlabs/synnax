// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cluster_test

import (
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/certificate"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/kafka/cluster"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	racktestutil "github.com/synnaxlabs/synnax/pkg/service/rack/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/secret"
	secrettestutil "github.com/synnaxlabs/synnax/pkg/service/secret/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/search"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Service", func() {
	var (
		stat    *status.Service
		rackSvc *rack.Service
		secrets *secret.Service
		certs   *certificate.Service
		svc     *cluster.Service
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
		svc = MustOpen(cluster.OpenService(ctx, cluster.ServiceConfig{
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
				Value: []string{cluster.Integration},
			}),
		)).To(Succeed())
	})

	createRack := func(ctx context.Context, integrations ...string) rack.Key {
		GinkgoHelper()
		r := rack.Rack{Name: "Other", Integrations: integrations}
		Expect(rackSvc.NewWriter(tx).Create(ctx, &r)).To(Succeed())
		return r.Key
	}
	newCluster := func() cluster.Cluster {
		var c cluster.Cluster
		c.SetDefaults()
		c.Rack = rackSvc.EmbeddedKey
		c.Name = "Plant Cluster"
		c.Brokers = []cluster.Broker{{Host: "kafka.example.com"}}
		return c
	}
	read := func(topic string, recordKey *string, pointer string, message uuid.UUID) cluster.Read {
		return cluster.Read{
			Topic:     topic,
			RecordKey: recordKey,
			Message:   message,
			Pointer:   pointer,
			Index:     1,
			Channel:   2,
		}
	}
	createSecret := func(ctx context.Context) secret.Key {
		GinkgoHelper()
		s := secret.Secret{Variant: secret.StoredSecret{
			Base: secret.Base{Name: "Cluster password"},
		}}
		Expect(secrets.NewWriter(tx).Set(ctx, &s, []byte("hunter2"))).To(Succeed())
		return s.Base().Key
	}
	createFileSecret := func(ctx context.Context, r rack.Key) secret.Key {
		GinkgoHelper()
		s := secret.Secret{Variant: secret.FileSecret{
			Base:     secret.Base{Name: "Cluster password file"},
			FileName: "password.txt",
			Rack:     r,
		}}
		Expect(secrets.NewWriter(tx).Set(ctx, &s, nil)).To(Succeed())
		return s.Base().Key
	}
	plain := func(password secret.Key) cluster.Auth {
		return cluster.Auth{Variant: cluster.PlainAuth{
			Credentials: cluster.Credentials{Username: "admin", Password: password},
		}}
	}
	createFileCertificate := func(
		ctx context.Context,
		r rack.Key,
		privateKey bool,
	) certificate.Key {
		GinkgoHelper()
		f := certificate.FileCertificate{
			Name:            "Cluster CA",
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
	tls := func(ca, client *certificate.Key) cluster.Transport {
		return cluster.Transport{Variant: cluster.TlsTransport{
			CaCertificate:     ca,
			ClientCertificate: client,
		}}
	}
	create := func(ctx context.Context) cluster.Cluster {
		GinkgoHelper()
		c := newCluster()
		Expect(svc.NewWriter(tx).Create(ctx, &c)).To(Succeed())
		return c
	}
	retrieve := func(ctx context.Context, key cluster.Key) cluster.Cluster {
		GinkgoHelper()
		var c cluster.Cluster
		Expect(svc.NewRetrieve().
			Where(cluster.MatchKeys(key)).
			Entry(&c).
			Exec(ctx, tx)).To(Succeed())
		return c
	}
	retrieveStatus := func(ctx context.Context, key cluster.Key) cluster.Status {
		GinkgoHelper()
		var s cluster.Status
		Expect(stat.NewRetrieve[cluster.StatusDetails]().
			Where(status.MatchOwners[cluster.StatusDetails](cluster.ResourceID(key))).
			Entry(&s).
			Exec(ctx, tx)).To(Succeed())
		return s
	}
	identity := func(c cluster.Cluster) cluster.Identity {
		return cluster.Identity{
			Rack:      c.Rack,
			Name:      c.Name,
			Brokers:   c.Brokers,
			Transport: c.Transport,
			Auth:      c.Auth,
		}
	}

	Describe("OpenService", func() {
		DescribeTable(
			"Should reject a config missing a required field",
			func(ctx SpecContext, field string, unset func(*cluster.ServiceConfig)) {
				cfg := cluster.ServiceConfig{
					DB:          db,
					Status:      stat,
					Secret:      secrets,
					Certificate: certs,
					Rack:        rackSvc,
				}
				unset(&cfg)
				Expect(cluster.OpenService(ctx, cfg)).Error().
					To(MatchError(ContainSubstring(field + ": must be non-nil")))
			},
			Entry("db", "db", func(c *cluster.ServiceConfig) { c.DB = nil }),
			Entry(
				"status",
				"status",
				func(c *cluster.ServiceConfig) { c.Status = nil },
			),
			Entry(
				"secret",
				"secret",
				func(c *cluster.ServiceConfig) { c.Secret = nil },
			),
			Entry("certificate", "certificate", func(c *cluster.ServiceConfig) {
				c.Certificate = nil
			}),
			Entry("rack", "rack", func(c *cluster.ServiceConfig) { c.Rack = nil }),
		)

		It("Should let a later config override an earlier one", func() {
			cfg := cluster.ServiceConfig{DB: db}.Override(cluster.ServiceConfig{
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
		It("Should mint a key, store the cluster, and write an unknown status", func(
			ctx SpecContext,
		) {
			c := create(ctx)
			Expect(c.Key).ToNot(Equal(uuid.Nil()))
			Expect(retrieve(ctx, c.Key)).To(Equal(c))
			st := retrieveStatus(ctx, c.Key)
			Expect(st.Variant).To(Equal(status.VariantWarning))
			Expect(st.Message).To(Equal("Status unknown"))
			Expect(st.Name).To(Equal(c.Name))
			Expect(st.Details.Cluster).To(Equal(c.Key))
			Expect(st.Details.Rack).To(Equal(c.Rack))
			Expect(st.Details.Connected).To(BeFalse())
		})

		It("Should keep a caller-set key", func(ctx SpecContext) {
			c := newCluster()
			c.Key = uuid.New()
			Expect(svc.NewWriter(tx).Create(ctx, &c)).To(Succeed())
			Expect(retrieve(ctx, c.Key).Key).To(Equal(c.Key))
		})

		It("Should accept brokers and reads that differ in one part of their key", func(
			ctx SpecContext,
		) {
			c := newCluster()
			c.Brokers = []cluster.Broker{
				{Host: "kafka.example.com"},
				{Host: "kafka.example.com", Port: new(uint16(9093))},
				{Host: "kafka-2.example.com"},
			}
			message := uuid.New()
			c.Reads = []cluster.Read{
				read("plant", nil, "/t", message),
				read("plant", new("pump"), "/t", message),
				read("plant", nil, "/p", message),
				read("plant", nil, "/t", uuid.New()),
				read("other", nil, "/t", message),
			}
			Expect(svc.NewWriter(tx).Create(ctx, &c)).To(Succeed())
			Expect(retrieve(ctx, c.Key)).To(Equal(c))
		})

		DescribeTable(
			"Should reject an invalid cluster",
			func(ctx SpecContext, change func(*cluster.Cluster), path string) {
				c := newCluster()
				change(&c)
				Expect(svc.NewWriter(tx).Create(ctx, &c)).To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring(path)),
				))
				Expect(svc.NewRetrieve().
					Where(cluster.MatchKeys(c.Key)).
					Exists(ctx, tx)).To(BeFalse())
			},
			Entry("no name", func(c *cluster.Cluster) { c.Name = "" }, "name"),
			Entry(
				"no brokers",
				func(c *cluster.Cluster) { c.Brokers = nil },
				"brokers",
			),
			Entry("a broker with no host", func(c *cluster.Cluster) {
				c.Brokers = []cluster.Broker{{}}
			}, "host"),
			Entry("a duplicate broker", func(c *cluster.Cluster) {
				c.Brokers = []cluster.Broker{
					{Host: "kafka.example.com"},
					{Host: "kafka.example.com"},
				}
			}, "brokers.1"),
			Entry("a broker that repeats the default port", func(c *cluster.Cluster) {
				c.Brokers = []cluster.Broker{
					{Host: "kafka.example.com"},
					{Host: "kafka.example.com", Port: new(uint16(9092))},
				}
			}, "brokers.1"),
			Entry(
				"a broker that repeats another in other case",
				func(c *cluster.Cluster) {
					c.Brokers = []cluster.Broker{
						{Host: "Kafka.Example.com"},
						{Host: "kafka.example.com"},
					}
				},
				"brokers.1",
			),
			Entry("a broker that repeats another IPv6 form", func(c *cluster.Cluster) {
				c.Brokers = []cluster.Broker{{Host: "::1"}, {Host: "0:0:0:0:0:0:0:1"}}
			}, "brokers.1"),
			Entry("a duplicate read", func(c *cluster.Cluster) {
				m := uuid.New()
				c.Reads = []cluster.Read{
					read("plant", new("pump"), "/t", m),
					read("plant", new("pump"), "/t", m),
				}
			}, "reads.1"),
			Entry("a duplicate read with no record key", func(c *cluster.Cluster) {
				m := uuid.New()
				c.Reads = []cluster.Read{
					read("plant", nil, "", m),
					read("plant", nil, "", m),
				}
			}, "reads.1"),
			Entry("an empty record key", func(c *cluster.Cluster) {
				c.Reads = []cluster.Read{read("plant", new(""), "", uuid.New())}
			}, "record_key"),
		)

		It("Should reject a rack that does not run Kafka", func(ctx SpecContext) {
			c := newCluster()
			c.Rack = createRack(ctx, "mqtt")
			Expect(svc.NewWriter(tx).Create(ctx, &c)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("does not run the kafka integration")),
			))
		})

		It("Should reject a missing rack", func(ctx SpecContext) {
			c := newCluster()
			c.Rack = 999
			Expect(svc.NewWriter(tx).Create(ctx, &c)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("rack 999 does not exist")),
				Not(MatchError(query.ErrNotFound)),
			))
		})

		It("Should reject a move to a missing rack as invalid", func(ctx SpecContext) {
			c := create(ctx)
			update := identity(c)
			update.Rack = 999
			Expect(svc.NewWriter(tx).Apply(ctx, c.Key, cluster.Update{
				Variant: cluster.IdentityUpdate{Identity: update},
			})).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				Not(MatchError(query.ErrNotFound)),
			))
		})
	})

	Describe("Certificates", func() {
		It("Should store TLS with no CA", func(ctx SpecContext) {
			c := newCluster()
			c.Transport = tls(nil, nil)
			Expect(svc.NewWriter(tx).Create(ctx, &c)).To(Succeed())
			Expect(retrieve(ctx, c.Key).Transport).To(Equal(c.Transport))
		})

		It("Should store a CA of the rack", func(ctx SpecContext) {
			ca := createFileCertificate(ctx, rackSvc.EmbeddedKey, false)
			c := newCluster()
			c.Transport = tls(&ca, nil)
			Expect(svc.NewWriter(tx).Create(ctx, &c)).To(Succeed())
			Expect(retrieve(ctx, c.Key).Transport).To(Equal(c.Transport))
		})

		It("Should store a CA and a client certificate of the rack", func(
			ctx SpecContext,
		) {
			ca := createFileCertificate(ctx, rackSvc.EmbeddedKey, false)
			client := createFileCertificate(ctx, rackSvc.EmbeddedKey, true)
			c := newCluster()
			c.Transport = tls(&ca, &client)
			Expect(svc.NewWriter(tx).Create(ctx, &c)).To(Succeed())
			Expect(retrieve(ctx, c.Key).Transport).To(Equal(c.Transport))
		})

		It("Should reject a missing client certificate", func(ctx SpecContext) {
			c := newCluster()
			c.Transport = tls(nil, new(uuid.New()))
			Expect(svc.NewWriter(tx).Create(ctx, &c)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("transport.client_certificate")),
				Not(MatchError(query.ErrNotFound)),
			))
		})

		It("Should reject a client certificate with no private key", func(
			ctx SpecContext,
		) {
			client := createFileCertificate(ctx, rackSvc.EmbeddedKey, false)
			c := newCluster()
			c.Transport = tls(nil, &client)
			Expect(svc.NewWriter(tx).Create(ctx, &c)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("holds no private key")),
			))
		})

		It("Should reject a client certificate of another rack", func(ctx SpecContext) {
			other := createRack(ctx, cluster.Integration)
			client := createFileCertificate(ctx, other, true)
			c := newCluster()
			c.Transport = tls(nil, &client)
			Expect(svc.NewWriter(tx).Create(ctx, &c)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(
					"is a file of rack %d, not of rack %d",
					other,
					c.Rack,
				)),
			))
		})

		It("Should refuse to delete a client certificate that a cluster uses", func(
			ctx SpecContext,
		) {
			client := createFileCertificate(ctx, rackSvc.EmbeddedKey, true)
			c := newCluster()
			c.Transport = tls(nil, &client)
			Expect(svc.NewWriter(tx).Create(ctx, &c)).To(Succeed())
			Expect(certs.NewWriter(tx).Delete(ctx, client)).To(MatchError(
				ContainSubstring("because kafka_cluster %s references it", c.Key),
			))
		})

		It("Should reject a file certificate of another rack", func(ctx SpecContext) {
			other := createRack(ctx, cluster.Integration)
			ca := createFileCertificate(ctx, other, false)
			c := newCluster()
			c.Transport = tls(&ca, nil)
			Expect(svc.NewWriter(tx).Create(ctx, &c)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(
					"is a file of rack %d, not of rack %d",
					other,
					c.Rack,
				)),
			))
		})

		It("Should refuse to delete a certificate that a cluster uses", func(
			ctx SpecContext,
		) {
			ca := createFileCertificate(ctx, rackSvc.EmbeddedKey, false)
			c := newCluster()
			c.Transport = tls(&ca, nil)
			Expect(svc.NewWriter(tx).Create(ctx, &c)).To(Succeed())
			Expect(certs.NewWriter(tx).Delete(ctx, ca)).To(MatchError(
				ContainSubstring("because kafka_cluster %s references it", c.Key),
			))
		})
	})

	Describe("Secrets", func() {
		DescribeTable("Should store credentials whose password names a secret",
			func(ctx SpecContext, auth func(cluster.Credentials) cluster.Auth) {
				c := newCluster()
				c.Auth = auth(cluster.Credentials{
					Username: "admin",
					Password: createSecret(ctx),
				})
				Expect(svc.NewWriter(tx).Create(ctx, &c)).To(Succeed())
				Expect(retrieve(ctx, c.Key).Auth).To(Equal(c.Auth))
			},
			Entry("PLAIN", func(cr cluster.Credentials) cluster.Auth {
				return cluster.Auth{Variant: cluster.PlainAuth{Credentials: cr}}
			}),
			Entry("SCRAM-SHA-256", func(cr cluster.Credentials) cluster.Auth {
				return cluster.Auth{Variant: cluster.ScramSha256Auth{Credentials: cr}}
			}),
			Entry("SCRAM-SHA-512", func(cr cluster.Credentials) cluster.Auth {
				return cluster.Auth{Variant: cluster.ScramSha512Auth{Credentials: cr}}
			}),
		)

		It("Should refuse to delete a password that a cluster uses", func(
			ctx SpecContext,
		) {
			key := createSecret(ctx)
			c := newCluster()
			c.Auth = cluster.Auth{Variant: cluster.ScramSha512Auth{
				Credentials: cluster.Credentials{Username: "admin", Password: key},
			}}
			Expect(svc.NewWriter(tx).Create(ctx, &c)).To(Succeed())
			Expect(secrets.NewWriter(tx).Delete(ctx, key)).To(MatchError(
				ContainSubstring(
					"because kafka_cluster %s references it through auth.password",
					c.Key,
				),
			))
		})

		It("Should store a file secret of the rack", func(ctx SpecContext) {
			c := newCluster()
			c.Auth = plain(createFileSecret(ctx, rackSvc.EmbeddedKey))
			Expect(svc.NewWriter(tx).Create(ctx, &c)).To(Succeed())
			Expect(retrieve(ctx, c.Key).Auth).To(Equal(c.Auth))
		})

		It("Should reject a file secret of another rack", func(ctx SpecContext) {
			other := createRack(ctx, cluster.Integration)
			c := newCluster()
			c.Auth = plain(createFileSecret(ctx, other))
			Expect(svc.NewWriter(tx).Create(ctx, &c)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(
					"is a file of rack %d, not of rack %d",
					other,
					c.Rack,
				)),
			))
		})

		It("Should reject a move to a rack of which the password is not a file", func(
			ctx SpecContext,
		) {
			c := newCluster()
			c.Auth = plain(createFileSecret(ctx, rackSvc.EmbeddedKey))
			Expect(svc.NewWriter(tx).Create(ctx, &c)).To(Succeed())
			update := identity(c)
			update.Rack = createRack(ctx, cluster.Integration)
			Expect(svc.NewWriter(tx).Apply(ctx, c.Key, cluster.Update{
				Variant: cluster.IdentityUpdate{Identity: update},
			})).To(MatchError(validate.ErrValidation))
			Expect(retrieve(ctx, c.Key).Rack).To(Equal(c.Rack))
		})

		It("Should reject a missing password secret", func(ctx SpecContext) {
			c := newCluster()
			c.Auth = cluster.Auth{Variant: cluster.PlainAuth{
				Credentials: cluster.Credentials{
					Username: "admin",
					Password: uuid.New(),
				},
			}}
			Expect(svc.NewWriter(tx).Create(ctx, &c)).
				To(MatchError(validate.ErrValidation))
		})
	})

	Describe("Apply", func() {
		It("Should write the identity group and keep the reads", func(
			ctx SpecContext,
		) {
			c := newCluster()
			c.Reads = []cluster.Read{read("plant", nil, "", uuid.New())}
			Expect(svc.NewWriter(tx).Create(ctx, &c)).To(Succeed())
			update := identity(c)
			update.Name = "Renamed"
			update.Brokers = []cluster.Broker{
				{Host: "10.0.0.2", Port: new(uint16(9093))},
			}
			Expect(svc.NewWriter(tx).Apply(ctx, c.Key, cluster.Update{
				Variant: cluster.IdentityUpdate{Identity: update},
			})).To(Succeed())
			got := retrieve(ctx, c.Key)
			Expect(got.Name).To(Equal("Renamed"))
			Expect(got.Brokers).To(Equal(update.Brokers))
			Expect(got.Reads).To(Equal(c.Reads))
			Expect(retrieveStatus(ctx, c.Key).Name).To(Equal("Renamed"))
		})

		It("Should write the configuration group and keep the connection", func(
			ctx SpecContext,
		) {
			c := create(ctx)
			configuration := cluster.Configuration{
				Reads: []cluster.Read{read("plant", new("pump"), "/t", uuid.New())},
			}
			Expect(svc.NewWriter(tx).Apply(ctx, c.Key, cluster.Update{
				Variant: cluster.ConfigurationUpdate{Configuration: configuration},
			})).To(Succeed())
			got := retrieve(ctx, c.Key)
			Expect(identity(got)).To(Equal(identity(c)))
			Expect(got.Reads).To(Equal(configuration.Reads))
		})

		It("Should move the status with the rack", func(ctx SpecContext) {
			c := create(ctx)
			update := identity(c)
			update.Rack = createRack(ctx, cluster.Integration)
			Expect(svc.NewWriter(tx).Apply(ctx, c.Key, cluster.Update{
				Variant: cluster.IdentityUpdate{Identity: update},
			})).To(Succeed())
			Expect(retrieveStatus(ctx, c.Key).Details.Rack).To(Equal(update.Rack))
		})

		It("Should reject a move to a rack of which a certificate is not a file", func(
			ctx SpecContext,
		) {
			ca := createFileCertificate(ctx, rackSvc.EmbeddedKey, false)
			c := newCluster()
			c.Transport = tls(&ca, nil)
			Expect(svc.NewWriter(tx).Create(ctx, &c)).To(Succeed())
			update := identity(c)
			update.Rack = createRack(ctx, cluster.Integration)
			Expect(svc.NewWriter(tx).Apply(ctx, c.Key, cluster.Update{
				Variant: cluster.IdentityUpdate{Identity: update},
			})).To(MatchError(validate.ErrValidation))
			Expect(retrieve(ctx, c.Key).Rack).To(Equal(c.Rack))
		})

		DescribeTable(
			"Should reject an invalid update",
			func(ctx SpecContext, update func(cluster.Cluster) cluster.Update) {
				c := create(ctx)
				Expect(svc.NewWriter(tx).Apply(ctx, c.Key, update(c))).
					To(MatchError(validate.ErrValidation))
				Expect(retrieve(ctx, c.Key)).To(Equal(c))
			},
			Entry("no group", func(cluster.Cluster) cluster.Update {
				return cluster.Update{}
			}),
			Entry("an identity without a name", func(c cluster.Cluster) cluster.Update {
				i := identity(c)
				i.Name = ""
				return cluster.Update{Variant: cluster.IdentityUpdate{Identity: i}}
			}),
			Entry(
				"an identity with no brokers",
				func(c cluster.Cluster) cluster.Update {
					i := identity(c)
					i.Brokers = nil
					return cluster.Update{Variant: cluster.IdentityUpdate{Identity: i}}
				},
			),
			Entry("a configuration with a duplicate read", func(
				cluster.Cluster,
			) cluster.Update {
				m := uuid.New()
				return cluster.Update{Variant: cluster.ConfigurationUpdate{
					Configuration: cluster.Configuration{Reads: []cluster.Read{
						read("plant", nil, "", m),
						read("plant", nil, "", m),
					}},
				}}
			}),
		)

		It("Should reject a move to a rack that does not run Kafka", func(
			ctx SpecContext,
		) {
			c := create(ctx)
			update := identity(c)
			update.Rack = createRack(ctx)
			Expect(svc.NewWriter(tx).Apply(ctx, c.Key, cluster.Update{
				Variant: cluster.IdentityUpdate{Identity: update},
			})).To(MatchError(validate.ErrValidation))
			Expect(retrieve(ctx, c.Key).Rack).To(Equal(c.Rack))
		})

		It("Should return not found for a missing cluster", func(ctx SpecContext) {
			Expect(svc.NewWriter(tx).Apply(ctx, uuid.New(), cluster.Update{
				Variant: cluster.ConfigurationUpdate{},
			})).To(MatchError(query.ErrNotFound))
		})
	})

	Describe("Delete", func() {
		It("Should remove the cluster and its status", func(ctx SpecContext) {
			c := create(ctx)
			Expect(svc.NewWriter(tx).Delete(ctx, c.Key)).To(Succeed())
			Expect(svc.NewRetrieve().
				Where(cluster.MatchKeys(c.Key)).
				Exists(ctx, tx)).To(BeFalse())
			var st cluster.Status
			Expect(stat.NewRetrieve[cluster.StatusDetails]().
				Where(status.MatchOwners[cluster.StatusDetails](cluster.ResourceID(c.Key))).
				Entry(&st).
				Exec(ctx, tx)).To(MatchError(query.ErrNotFound))
		})

		It("Should be idempotent", func(ctx SpecContext) {
			Expect(svc.NewWriter(tx).Delete(ctx, uuid.New())).To(Succeed())
		})
	})

	It("Should refuse to delete a rack that holds a cluster", func(ctx SpecContext) {
		c := newCluster()
		c.Rack = createRack(ctx, cluster.Integration)
		Expect(svc.NewWriter(tx).Create(ctx, &c)).To(Succeed())
		Expect(rackSvc.NewWriter(tx).Delete(ctx, nil, c.Rack)).To(MatchError(
			ContainSubstring("because kafka_cluster %s references it", c.Key),
		))
		Expect(retrieve(ctx, c.Key).Rack).To(Equal(c.Rack))
	})

	Describe("Resources", func() {
		It("Should name a cluster by its resource ID", func(ctx SpecContext) {
			c := create(ctx)
			Expect(svc.Resources().Retrieve(ctx, tx, []resource.ID{c.ResourceID()})).
				To(Equal([]resource.Resource{{ID: c.ResourceID(), Name: c.Name}}))
		})

		It("Should find a cluster by its name", func(ctx SpecContext) {
			c := create(ctx)
			res := MustSucceed(svc.Resources().Search(
				ctx,
				tx,
				search.Query[resource.ID]{Term: "plant", Limit: 10},
			))
			Expect(res.Keys).To(ContainElement(c.ResourceID()))
		})

		It("Should reject a resource ID with a malformed key", func(ctx SpecContext) {
			Expect(svc.Resources().Retrieve(ctx, tx, []resource.ID{{
				Type: resource.TypeKafkaCluster,
				Key:  "cluster",
			}})).Error().To(MatchError(validate.ErrValidation))
		})
	})
})
