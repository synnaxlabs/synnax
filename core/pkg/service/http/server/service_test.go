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
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	channeltestutil "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/http/server"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	racktestutil "github.com/synnaxlabs/synnax/pkg/service/rack/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/secret"
	secrettestutil "github.com/synnaxlabs/synnax/pkg/service/secret/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/http"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/search"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Service", func() {
	var (
		stat       *status.Service
		rackSvc    *rack.Service
		secrets    *secret.Service
		channelSvc *channel.Service
		svc        *server.Service
		tx         gorp.Tx
		channels   channel.Keys
	)
	createChannels := func(ctx context.Context, count int) channel.Keys {
		GinkgoHelper()
		chs := make([]channel.Channel, count)
		for i := range chs {
			chs[i] = channel.Channel{
				Name:     channeltestutil.UniqueChannelName(),
				DataType: telem.Float64T,
				Virtual:  true,
			}
		}
		Expect(channelSvc.NewWriter(tx).CreateMany(ctx, &chs)).To(Succeed())
		return channel.KeysFromChannels(chs)
	}
	BeforeEach(func(ctx SpecContext) {
		MustOpen(group.OpenService(ctx, group.ServiceConfig{DB: db}))
		MustOpen(label.OpenService(ctx, label.ServiceConfig{DB: db}))
		stat = MustOpen(status.OpenService(ctx, status.ServiceConfig{DB: db}))
		roleSvc := MustOpen(role.OpenService(ctx, role.ServiceConfig{DB: db}))
		rackSvc = MustOpen(rack.OpenService(ctx, rack.ServiceConfig{
			DB:         db,
			HostKey:    node.Identity.HostKey,
			Status:     stat,
			DriverRole: MustSucceed(racktestutil.CreateDriverRole(ctx, db, roleSvc)),
		}))
		secrets = secrettestutil.OpenService(
			ctx,
			db,
			secret.ServiceConfig{EncryptionKey: secretsKey},
		)
		svc = MustOpen(server.OpenService(ctx, server.ServiceConfig{
			DB:     db,
			Status: stat,
			Secret: secrets,
		}))
		channelSvc = MustOpen(channel.OpenService(ctx, channel.ServiceConfig{
			Channel: node.Channel,
			TS:      node.Storage.TS,
			DB:      db,
			HostKey: node.Identity.HostKey,
			Status:  stat,
		}))
		tx = DeferClose(db.OpenTx())
		channels = createChannels(ctx, 6)
	})

	key := func(n int) channel.Key { return channels[n] }
	newServer := func() server.Server {
		var s server.Server
		s.SetDefaults()
		s.Rack = rackSvc.EmbeddedKey
		s.Name = "Weather API"
		s.Host = "weather.example.com"
		return s
	}
	read := func(path, pointer string, query ...http.QueryParam) server.Read {
		if query == nil {
			query = []http.QueryParam{}
		}
		return server.Read{
			Request: server.ReadRequest{
				Method: http.MethodGet,
				Path:   path,
				Query:  query,
			},
			Pointer: pointer,
			Index:   key(1),
			Channel: key(2),
		}
	}
	write := func(path string, ch channel.Key, query ...http.QueryParam) server.Write {
		if query == nil {
			query = []http.QueryParam{}
		}
		return server.Write{
			Request: server.Request{Method: http.MethodPost, Path: path, Query: query},
			Channel: ch,
		}
	}
	createSecret := func(ctx context.Context) secret.Key {
		GinkgoHelper()
		s := secret.Secret{Variant: secret.StoredSecret{
			Base: secret.Base{Name: "API token"},
		}}
		Expect(secrets.NewWriter(tx).Set(ctx, &s, []byte("hunter2"))).To(Succeed())
		return s.Base().Key
	}
	create := func(ctx context.Context) server.Server {
		GinkgoHelper()
		s := newServer()
		Expect(svc.NewWriter(tx).Create(ctx, &s)).To(Succeed())
		return s
	}
	retrieve := func(ctx context.Context, key server.Key) server.Server {
		GinkgoHelper()
		var s server.Server
		Expect(svc.NewRetrieve().
			Where(server.MatchKeys(key)).
			Entry(&s).
			Exec(ctx, tx)).To(Succeed())
		return s
	}
	retrieveStatus := func(ctx context.Context, key server.Key) server.Status {
		GinkgoHelper()
		var s server.Status
		Expect(stat.NewRetrieve[server.StatusDetails]().
			Where(status.MatchOwners[server.StatusDetails](server.ResourceID(key))).
			Entry(&s).
			Exec(ctx, tx)).To(Succeed())
		return s
	}
	identity := func(s server.Server) server.Identity {
		return server.Identity{
			Rack:                  s.Rack,
			Name:                  s.Name,
			Scheme:                s.Scheme,
			Host:                  s.Host,
			Port:                  s.Port,
			BasePath:              s.BasePath,
			Timeout:               s.Timeout,
			MaxConcurrentRequests: s.MaxConcurrentRequests,
			Auth:                  s.Auth,
			HealthCheck:           s.HealthCheck,
		}
	}

	Describe("OpenService", func() {
		DescribeTable("Should reject a config missing a required field",
			func(ctx SpecContext, field string, unset func(*server.ServiceConfig)) {
				cfg := server.ServiceConfig{DB: db, Status: stat, Secret: secrets}
				unset(&cfg)
				Expect(server.OpenService(ctx, cfg)).Error().
					To(MatchError(ContainSubstring(field + ": must be non-nil")))
			},
			Entry("db", "db", func(c *server.ServiceConfig) { c.DB = nil }),
			Entry("status", "status", func(c *server.ServiceConfig) { c.Status = nil }),
			Entry("secret", "secret", func(c *server.ServiceConfig) { c.Secret = nil }),
		)

		It("Should let a later config override an earlier one", func() {
			cfg := server.ServiceConfig{DB: db}.Override(server.ServiceConfig{
				Status: stat,
				Secret: secrets,
			})
			Expect(cfg.DB).To(BeIdenticalTo(db))
			Expect(cfg.Status).To(BeIdenticalTo(stat))
			Expect(cfg.Validate()).To(Succeed())
		})
	})

	Describe("Channels", func() {
		It("Should reject a binding to a missing channel", func(ctx SpecContext) {
			s := newServer()
			s.Writes = []server.Write{write("/valve", 99999)}
			Expect(svc.NewWriter(tx).Create(ctx, &s)).
				To(MatchError(ContainSubstring("channel 99999 does not exist")))
		})

		DescribeTable("Should keep a channel that a server binds from being deleted",
			func(ctx SpecContext, bound func(s *server.Server) channel.Key) {
				s := newServer()
				ch := bound(&s)
				Expect(svc.NewWriter(tx).Create(ctx, &s)).To(Succeed())
				Expect(channelSvc.NewWriter(tx).Delete(ctx, ch, false)).
					To(MatchError(validate.ErrValidation))
			},
			Entry("a read index", func(s *server.Server) channel.Key {
				s.Reads = []server.Read{read("/now", "/t")}
				return key(1)
			}),
			Entry("a read channel", func(s *server.Server) channel.Key {
				s.Reads = []server.Read{read("/now", "/t")}
				return key(2)
			}),
			Entry("a write channel", func(s *server.Server) channel.Key {
				s.Writes = []server.Write{write("/valve", key(3))}
				return key(3)
			}),
		)
	})

	Describe("Create", func() {
		It("Should mint a key, store the server, and write an unknown status", func(
			ctx SpecContext,
		) {
			s := create(ctx)
			Expect(s.Key).ToNot(Equal(uuid.Nil()))
			Expect(retrieve(ctx, s.Key)).To(Equal(s))
			st := retrieveStatus(ctx, s.Key)
			Expect(st.Variant).To(Equal(status.VariantWarning))
			Expect(st.Message).To(Equal("Status unknown"))
			Expect(st.Name).To(Equal(s.Name))
			Expect(st.Details.Server).To(Equal(s.Key))
			Expect(st.Details.Rack).To(Equal(s.Rack))
		})

		It("Should keep a caller-set key", func(ctx SpecContext) {
			s := newServer()
			s.Key = uuid.New()
			Expect(svc.NewWriter(tx).Create(ctx, &s)).To(Succeed())
			Expect(retrieve(ctx, s.Key).Key).To(Equal(s.Key))
		})

		It("Should give an existing status the name and rack of a replaced server",
			func(ctx SpecContext) {
				s := create(ctx)
				other := rack.Rack{Name: "Other"}
				Expect(rackSvc.NewWriter(tx).Create(ctx, &other)).To(Succeed())
				s.Name = "Pump API"
				s.Rack = other.Key
				Expect(svc.NewWriter(tx).Create(ctx, &s)).To(Succeed())
				st := retrieveStatus(ctx, s.Key)
				Expect(st.Name).To(Equal("Pump API"))
				Expect(st.Details.Rack).To(Equal(other.Key))
			},
		)

		DescribeTable(
			"Should reject an invalid server",
			func(ctx SpecContext, change func(*server.Server)) {
				s := newServer()
				change(&s)
				Expect(svc.NewWriter(tx).Create(ctx, &s)).
					To(MatchError(validate.ErrValidation))
			},
			Entry("no name", func(s *server.Server) { s.Name = "" }),
			Entry("no host", func(s *server.Server) { s.Host = "" }),
			Entry("a bearer token with no secret", func(s *server.Server) {
				s.Auth = server.Auth{Variant: server.BearerAuth{}}
			}),
			Entry("a missing secret", func(s *server.Server) {
				s.Auth = server.Auth{Variant: server.BearerAuth{Token: uuid.New()}}
			}),
			Entry("a missing rack", func(s *server.Server) { s.Rack = 999 }),
			Entry("a duplicate read", func(s *server.Server) {
				s.Reads = []server.Read{
					read("/now", "/t", http.QueryParam{Parameter: "a", Value: "1"}),
					read("now", "/t", http.QueryParam{Parameter: "a", Value: "1"}),
				}
			}),
			Entry("a duplicate write with a reordered query", func(s *server.Server) {
				a := http.QueryParam{Parameter: "a", Value: "1"}
				b := http.QueryParam{Parameter: "b", Value: "2"}
				s.Writes = []server.Write{
					write("/valve", key(3), a, b),
					write("/valve", key(4), b, a),
				}
			}),
			Entry("a duplicate write", func(s *server.Server) {
				s.Writes = []server.Write{
					write("/valve", key(3)),
					write("valve", key(4)),
				}
			}),
			Entry(
				"a duplicate write with a doubled leading slash",
				func(s *server.Server) {
					s.Writes = []server.Write{
						write("/valve", key(3)),
						write("//valve", key(4)),
					}
				},
			),
		)

		DescribeTable("Should reject a malformed header or query parameter",
			func(
				ctx SpecContext,
				change func(*server.Server, secret.Key),
				path string,
			) {
				s := newServer()
				change(&s, createSecret(ctx))
				Expect(svc.NewWriter(tx).Create(ctx, &s)).To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring(path)),
				))
			},
			Entry("an API key header name with a space", func(
				s *server.Server,
				key secret.Key,
			) {
				s.Auth = server.Auth{Variant: server.APIKeyAuth{
					Key:    key,
					SendAs: server.SendAsHeader,
					Name:   "X Key",
				}}
			}, "auth.name"),
			Entry("a health check header name with a colon", func(
				s *server.Server,
				_ secret.Key,
			) {
				s.HealthCheck.Headers = []http.Header{{Name: "Accept:", Value: "json"}}
			}, "health_check.headers.0.name"),
			Entry("a health check header value with a line break", func(
				s *server.Server,
				_ secret.Key,
			) {
				s.HealthCheck.Headers = []http.Header{{Name: "Accept", Value: "a\r\nb"}}
			}, "health_check.headers.0.value"),
			Entry("a health check query parameter with no name", func(
				s *server.Server,
				_ secret.Key,
			) {
				s.HealthCheck.QueryParams = []http.QueryParam{{Value: "1"}}
			}, "health_check.query_params.0.parameter"),
			Entry("a read query parameter with no name", func(
				s *server.Server,
				_ secret.Key,
			) {
				s.Reads = []server.Read{read("/now", "/t", http.QueryParam{Value: "1"})}
			}, "reads.0.request.query.0.parameter"),
			Entry("a write query parameter with no name", func(
				s *server.Server,
				_ secret.Key,
			) {
				s.Writes = []server.Write{
					write("/valve", key(3), http.QueryParam{Value: "1"}),
				}
			}, "writes.0.request.query.0.parameter"),
		)

		It("Should accept an API key sent as a query parameter of any name", func(
			ctx SpecContext,
		) {
			s := newServer()
			s.Auth = server.Auth{Variant: server.APIKeyAuth{
				Key:    createSecret(ctx),
				SendAs: server.SendAsQueryParam,
				Name:   "api[key]",
			}}
			Expect(svc.NewWriter(tx).Create(ctx, &s)).To(Succeed())
		})

		It("Should normalize the paths and queries of bindings", func(ctx SpecContext) {
			s := newServer()
			s.Reads = []server.Read{read(
				"weather/now/",
				"/temp",
				http.QueryParam{Parameter: "units", Value: "si"},
				http.QueryParam{Parameter: "city", Value: "Austin"},
			)}
			s.Writes = []server.Write{write("//", key(3)), write("/valve/", key(4))}
			Expect(svc.NewWriter(tx).Create(ctx, &s)).To(Succeed())
			got := retrieve(ctx, s.Key)
			Expect(got.Reads[0].Request.Path).To(Equal("/weather/now/"))
			Expect(got.Reads[0].Request.Query).To(Equal([]http.QueryParam{
				{Parameter: "city", Value: "Austin"},
				{Parameter: "units", Value: "si"},
			}))
			Expect(got.Writes[0].Request.Path).To(Equal("/"))
			Expect(got.Writes[1].Request.Path).To(Equal("/valve/"))
		})

		It("Should accept bindings that differ only in method, query, or index", func(
			ctx SpecContext,
		) {
			s := newServer()
			post := read("/now", "/t")
			post.Request.Method = http.MethodPost
			reindexed := read("/now", "/t")
			reindexed.Index = key(5)
			s.Reads = []server.Read{
				read("/now", "/t"),
				post,
				read("/now", "/t", http.QueryParam{Parameter: "a", Value: "1"}),
				reindexed,
			}
			put := write("/valve", key(3))
			put.Request.Method = http.MethodPut
			s.Writes = []server.Write{
				write("/valve", key(3)),
				put,
				write("/valve", key(3), http.QueryParam{Parameter: "id", Value: "1"}),
			}
			Expect(svc.NewWriter(tx).Create(ctx, &s)).To(Succeed())
		})
	})

	Describe("Apply", func() {
		It("Should write the identity group and keep the channel maps", func(
			ctx SpecContext,
		) {
			s := newServer()
			s.Reads = []server.Read{read("/weather", "/temp")}
			Expect(svc.NewWriter(tx).Create(ctx, &s)).To(Succeed())
			update := identity(s)
			update.Name = "Renamed"
			update.Host = "other.example.com"
			update.Timeout = 500 * telem.Millisecond
			Expect(svc.NewWriter(tx).Apply(ctx, s.Key, server.Update{
				Variant: server.IdentityUpdate{Identity: update},
			})).To(Succeed())
			got := retrieve(ctx, s.Key)
			Expect(got.Name).To(Equal("Renamed"))
			Expect(got.Host).To(Equal("other.example.com"))
			Expect(got.Timeout).To(Equal(500 * telem.Millisecond))
			Expect(got.Reads).To(Equal(s.Reads))
			Expect(retrieveStatus(ctx, s.Key).Name).To(Equal("Renamed"))
		})

		It("Should write the configuration group and keep the connection", func(
			ctx SpecContext,
		) {
			s := create(ctx)
			Expect(svc.NewWriter(tx).Apply(ctx, s.Key, server.Update{
				Variant: server.ConfigurationUpdate{Configuration: server.Configuration{
					Reads:  []server.Read{read("/weather", "/temp")},
					Writes: []server.Write{write("/valve", key(3))},
				}},
			})).To(Succeed())
			got := retrieve(ctx, s.Key)
			Expect(got.Name).To(Equal(s.Name))
			Expect(got.Host).To(Equal(s.Host))
			Expect(got.Auth).To(Equal(s.Auth))
			Expect(got.Reads).To(Equal([]server.Read{read("/weather", "/temp")}))
			Expect(got.Writes).To(Equal([]server.Write{write("/valve", key(3))}))
		})

		It("Should move the status with the rack", func(ctx SpecContext) {
			s := create(ctx)
			other := rack.Rack{Name: "Other"}
			Expect(rackSvc.NewWriter(tx).Create(ctx, &other)).To(Succeed())
			update := identity(s)
			update.Rack = other.Key
			Expect(svc.NewWriter(tx).Apply(ctx, s.Key, server.Update{
				Variant: server.IdentityUpdate{Identity: update},
			})).To(Succeed())
			Expect(retrieveStatus(ctx, s.Key).Details.Rack).To(Equal(other.Key))
		})

		It("Should write an unknown status when the status is missing", func(
			ctx SpecContext,
		) {
			s := create(ctx)
			Expect(stat.NewWriter(tx).Delete(ctx, retrieveStatus(ctx, s.Key).Key)).
				To(Succeed())
			update := identity(s)
			update.Name = "Renamed"
			Expect(svc.NewWriter(tx).Apply(ctx, s.Key, server.Update{
				Variant: server.IdentityUpdate{Identity: update},
			})).To(Succeed())
			st := retrieveStatus(ctx, s.Key)
			Expect(st.Message).To(Equal("Status unknown"))
			Expect(st.Name).To(Equal("Renamed"))
		})

		DescribeTable(
			"Should reject an invalid update",
			func(ctx SpecContext, update func(server.Server) server.Update) {
				s := create(ctx)
				Expect(svc.NewWriter(tx).Apply(ctx, s.Key, update(s))).
					To(MatchError(validate.ErrValidation))
				Expect(retrieve(ctx, s.Key)).To(Equal(s))
			},
			Entry("no group", func(server.Server) server.Update {
				return server.Update{}
			}),
			Entry("an identity without a name", func(s server.Server) server.Update {
				i := identity(s)
				i.Name = ""
				return server.Update{Variant: server.IdentityUpdate{Identity: i}}
			}),
			Entry("an identity with no host", func(s server.Server) server.Update {
				i := identity(s)
				i.Host = ""
				return server.Update{Variant: server.IdentityUpdate{Identity: i}}
			}),
			Entry("a configuration with a duplicate write", func(
				server.Server,
			) server.Update {
				return server.Update{Variant: server.ConfigurationUpdate{
					Configuration: server.Configuration{
						Writes: []server.Write{
							write("/valve", key(3)),
							write("valve", key(4)),
						},
					},
				}}
			}),
			Entry("a configuration with a read that has no channel", func(
				server.Server,
			) server.Update {
				r := read("/now", "/t")
				r.Channel = 0
				return server.Update{Variant: server.ConfigurationUpdate{
					Configuration: server.Configuration{Reads: []server.Read{r}},
				}}
			}),
		)

		It("Should return not found for a missing server", func(ctx SpecContext) {
			Expect(svc.NewWriter(tx).Apply(ctx, uuid.New(), server.Update{
				Variant: server.ConfigurationUpdate{},
			})).To(MatchError(query.ErrNotFound))
		})
	})

	Describe("Delete", func() {
		It("Should remove the server and its status", func(ctx SpecContext) {
			s := create(ctx)
			Expect(svc.NewWriter(tx).Delete(ctx, s.Key)).To(Succeed())
			Expect(svc.NewRetrieve().
				Where(server.MatchKeys(s.Key)).
				Exists(ctx, tx)).To(BeFalse())
			var st server.Status
			Expect(stat.NewRetrieve[server.StatusDetails]().
				Where(status.MatchOwners[server.StatusDetails](server.ResourceID(s.Key))).
				Entry(&st).
				Exec(ctx, tx)).To(MatchError(query.ErrNotFound))
		})

		It("Should be idempotent", func(ctx SpecContext) {
			Expect(svc.NewWriter(tx).Delete(ctx, uuid.New())).To(Succeed())
		})
	})

	It("Should store a server whose auth names a secret", func(ctx SpecContext) {
		s := newServer()
		s.Auth = server.Auth{Variant: server.BasicAuth{
			Username: "admin",
			Password: createSecret(ctx),
		}}
		Expect(svc.NewWriter(tx).Create(ctx, &s)).To(Succeed())
		Expect(retrieve(ctx, s.Key).Auth).To(Equal(s.Auth))
	})

	DescribeTable("Should refuse to delete a secret that a server uses",
		func(ctx SpecContext, auth func(secret.Key) server.AuthVariant) {
			key := createSecret(ctx)
			s := newServer()
			s.Auth = server.Auth{Variant: auth(key)}
			Expect(svc.NewWriter(tx).Create(ctx, &s)).To(Succeed())
			Expect(secrets.NewWriter(tx).Delete(ctx, key)).To(MatchError(
				ContainSubstring(
					"because http_server %s references it through auth",
					s.Key,
				),
			))
		},
		Entry("bearer", func(k secret.Key) server.AuthVariant {
			return server.BearerAuth{Token: k}
		}),
		Entry("basic", func(k secret.Key) server.AuthVariant {
			return server.BasicAuth{Username: "admin", Password: k}
		}),
		Entry("API key", func(k secret.Key) server.AuthVariant {
			return server.APIKeyAuth{Key: k, SendAs: server.SendAsHeader, Name: "X-Key"}
		}),
	)

	It("Should reject a secret file of another rack", func(ctx SpecContext) {
		other := rack.Rack{Name: "Other"}
		Expect(rackSvc.NewWriter(tx).Create(ctx, &other)).To(Succeed())
		file := secret.Secret{Variant: secret.FileSecret{
			Base:     secret.Base{Name: "API token"},
			FileName: "token",
			Rack:     other.Key,
		}}
		Expect(secrets.NewWriter(tx).Set(ctx, &file, nil)).To(Succeed())
		s := newServer()
		s.Auth = server.Auth{Variant: server.BearerAuth{Token: file.Base().Key}}
		Expect(svc.NewWriter(tx).Create(ctx, &s)).To(SatisfyAll(
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
		Expect(rackSvc.NewWriter(tx).Create(ctx, &other)).To(Succeed())
		s := newServer()
		s.Rack = other.Key
		Expect(svc.NewWriter(tx).Create(ctx, &s)).To(Succeed())
		Expect(rackSvc.NewWriter(tx).Delete(ctx, nil, other.Key)).To(MatchError(
			ContainSubstring("because http_server %s references it", s.Key),
		))
		Expect(retrieve(ctx, s.Key).Rack).To(Equal(other.Key))
	})

	Describe("Resources", func() {
		It("Should name a server by its resource ID", func(ctx SpecContext) {
			s := create(ctx)
			Expect(svc.Resources().Retrieve(ctx, tx, []resource.ID{s.ResourceID()})).
				To(Equal([]resource.Resource{{ID: s.ResourceID(), Name: s.Name}}))
		})

		It("Should find a server by its name", func(ctx SpecContext) {
			s := create(ctx)
			res := MustSucceed(svc.Resources().Search(
				ctx,
				tx,
				search.Query[resource.ID]{Term: "weather", Limit: 10},
			))
			Expect(res.Keys).To(ContainElement(s.ResourceID()))
		})

		It("Should reject a resource ID with a malformed key", func(ctx SpecContext) {
			Expect(svc.Resources().Retrieve(ctx, tx, []resource.ID{{
				Type: resource.TypeHTTPServer,
				Key:  "plc",
			}})).Error().To(MatchError(validate.ErrValidation))
		})
	})
})
