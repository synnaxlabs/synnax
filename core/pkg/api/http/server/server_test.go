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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	apiserver "github.com/synnaxlabs/synnax/pkg/api/http/server"
	"github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/http/server"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/filter"
	"github.com/synnaxlabs/x/http"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var (
	serverType = resource.ID{Type: resource.TypeHTTPServer}
	statusType = resource.ID{Type: resource.TypeStatus}
	allWrites  = []access.Action{
		access.ActionCreate,
		access.ActionUpdate,
		access.ActionDelete,
	}
)

var _ = Describe("Service", func() {
	var u user.User
	BeforeEach(func(ctx SpecContext) { u = createUser(ctx) })

	newServer := func(overrides ...func(*server.Server)) server.Server {
		var s server.Server
		s.SetDefaults()
		s.Key = uuid.New()
		s.Rack = svc.Rack.EmbeddedKey
		s.Name = "API-" + uuid.New().String()
		s.Host = "api.example.com"
		for _, override := range overrides {
			override(&s)
		}
		return s
	}
	grantRetrieve := func(ctx SpecContext) {
		GinkgoHelper()
		grant(ctx, u, []access.Action{access.ActionRetrieve}, serverType)
	}
	// grantWrites lets u write any HTTP server, its status, and the rack it sits on.
	grantWrites := func(ctx SpecContext, r rack.Key) {
		GinkgoHelper()
		grant(ctx, u, allWrites, serverType, statusType)
		grant(ctx, u, []access.Action{access.ActionUpdate}, r.ResourceID())
	}
	retrieveServer := func(ctx SpecContext, key server.Key) server.Server {
		GinkgoHelper()
		var s server.Server
		Expect(svc.HTTP.Servers.NewRetrieve().
			Where(server.MatchKeys(key)).
			Entry(&s).
			Exec(ctx, db)).To(Succeed())
		return s
	}
	createServer := func(ctx SpecContext, overrides ...func(*server.Server)) server.Server {
		GinkgoHelper()
		s := newServer(overrides...)
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.HTTP.Servers.NewWriter(tx).Create(ctx, &s)
		})).To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.HTTP.Servers.NewWriter(tx).Delete(ctx, s.Key)
			})).To(Succeed())
		})
		return s
	}

	Describe("NewService", func() {
		DescribeTable("Should reject a config missing a required field",
			func(field string, unset func(*apiserver.ServiceConfig)) {
				cfg := apiserver.ServiceConfig{
					Server: svc.HTTP.Servers,
					Status: svc.Status,
				}
				unset(&cfg)
				Expect(apiserver.NewService(cfg)).Error().
					To(MatchError(ContainSubstring(field + ": must be non-nil")))
			},
			Entry("server", "server", func(c *apiserver.ServiceConfig) {
				c.Server = nil
			}),
			Entry("status", "status", func(c *apiserver.ServiceConfig) {
				c.Status = nil
			}),
		)

		It("Should let a later config override an earlier one", func() {
			base := apiserver.ServiceConfig{Server: svc.HTTP.Servers}
			Expect(apiserver.NewService(base, apiserver.ServiceConfig{
				Status: svc.Status,
			})).ToNot(BeNil())
		})
	})

	Describe("Retrieve", func() {
		It("Should reject a named server when retrieve is not granted", func(
			ctx SpecContext,
		) {
			key := createServer(ctx).Key
			Expect(retrieve(
				testutil.AuthedCtx(ctx, u),
				apiserver.RetrieveRequest{
					Where: &server.Where{Key: &filter.Set[server.Key]{Eq: &key}},
				},
			)).Error().To(MatchError(access.ErrDenied))
		})

		It("Should filter by rack and attach the status", func(ctx SpecContext) {
			s := createServer(ctx)
			createServer(ctx)
			st := server.Status{
				Owner:   new(s.ResourceID()),
				Name:    s.Name,
				Time:    telem.Now(),
				Variant: status.VariantSuccess,
				Message: "Server connected",
				Details: server.StatusDetails{Server: s.Key, Connected: true},
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.Status.NewWriter(tx).Set(ctx, &st)
			})).To(Succeed())
			grantRetrieve(ctx)
			res := MustSucceed(retrieve(
				testutil.AuthedCtx(ctx, u),
				apiserver.RetrieveRequest{
					Where: &server.Where{
						Name: &filter.String{Eq: &s.Name},
						Rack: &filter.Set[rack.Key]{Eq: &s.Rack},
					},
					IncludeStatus: true,
				},
			))
			Expect(res.Servers).To(HaveLen(1))
			Expect(res.Servers[0].Key).To(Equal(s.Key))
			Expect(res.Servers[0].Status.Message).To(Equal("Server connected"))
			Expect(res.Servers[0].Status.Details.Connected).To(BeTrue())
		})

		It("Should leave out the servers the subject cannot retrieve", func(
			ctx SpecContext,
		) {
			name := createServer(ctx).Name
			res := MustSucceed(retrieve(
				testutil.AuthedCtx(ctx, u),
				apiserver.RetrieveRequest{
					Where: &server.Where{Name: &filter.String{Eq: &name}},
				},
			))
			Expect(res.Servers).To(BeEmpty())
		})

		It("Should find a server by its name", func(ctx SpecContext) {
			d := createServer(ctx, func(s *server.Server) { s.Name = "Boiler PLC" })
			grantRetrieve(ctx)
			res := MustSucceed(retrieve(
				testutil.AuthedCtx(ctx, u),
				apiserver.RetrieveRequest{SearchTerm: "boiler plc"},
			))
			Expect(lo.Map(res.Servers, func(s server.Server, _ int) server.Key {
				return s.Key
			})).To(ContainElement(d.Key))
		})

		It("Should cap the results at the limit", func(ctx SpecContext) {
			a := createServer(ctx)
			b := createServer(ctx)
			grantRetrieve(ctx)
			res := MustSucceed(retrieve(
				testutil.AuthedCtx(ctx, u),
				apiserver.RetrieveRequest{
					Where: &server.Where{Key: &filter.Set[server.Key]{
						In: []server.Key{a.Key, b.Key},
					}},
					Limit: new(uint32(1)),
				},
			))
			Expect(res.Servers).To(HaveLen(1))
		})

		It("Should page past the after key", func(ctx SpecContext) {
			a := createServer(ctx)
			b := createServer(ctx)
			grantRetrieve(ctx)
			authed := testutil.AuthedCtx(ctx, u)
			req := apiserver.RetrieveRequest{
				Where: &server.Where{Key: &filter.Set[server.Key]{
					In: []server.Key{a.Key, b.Key},
				}},
				Limit: new(uint32(1)),
			}
			first := MustSucceed(retrieve(authed, req))
			Expect(first.Servers).To(HaveLen(1))
			req.After = &apiserver.Cursor{Key: first.Servers[0].Key}
			second := MustSucceed(retrieve(authed, req))
			Expect(second.Servers).To(HaveLen(1))
			Expect(second.Servers[0].Key).
				ToNot(Equal(first.Servers[0].Key))
		})

		It("Should reject a search that resumes after a key", func(ctx SpecContext) {
			d := createServer(ctx)
			grantRetrieve(ctx)
			Expect(retrieve(
				testutil.AuthedCtx(ctx, u),
				apiserver.RetrieveRequest{
					SearchTerm: d.Name,
					After:      &apiserver.Cursor{Key: d.Key},
				},
			)).Error().To(MatchError(validate.ErrValidation))
		})

		It("Should return a path error for a malformed where", func(ctx SpecContext) {
			grantRetrieve(ctx)
			Expect(retrieve(testutil.AuthedCtx(ctx, u), apiserver.RetrieveRequest{
				Where: &server.Where{Name: &filter.String{Regex: new("(a")}},
			})).Error().To(SatisfyAll(
				BeAssignableToTypeOf(validate.PathError{}),
				MatchError(ContainSubstring("where.name.regex: error parsing regexp")),
				MatchError(validate.ErrValidation),
			))
		})

		It("Should return no servers for a zero limit", func(ctx SpecContext) {
			createServer(ctx)
			grantRetrieve(ctx)
			res := MustSucceed(retrieve(
				testutil.AuthedCtx(ctx, u),
				apiserver.RetrieveRequest{Limit: new(uint32(0))},
			))
			Expect(res.Servers).To(BeEmpty())
		})

		It("Should return no servers for a missing key", func(ctx SpecContext) {
			grantRetrieve(ctx)
			missing := uuid.New()
			res := MustSucceed(retrieve(
				testutil.AuthedCtx(ctx, u),
				apiserver.RetrieveRequest{
					Where: &server.Where{Key: &filter.Set[server.Key]{Eq: &missing}},
				},
			))
			Expect(res.Servers).To(BeEmpty())
		})
	})

	Describe("Create", func() {
		It("Should mint a key when writes are granted", func(ctx SpecContext) {
			grantWrites(ctx, svc.Rack.EmbeddedKey)
			s := newServer()
			s.Key = uuid.Nil()
			res := MustSucceed(create(
				testutil.AuthedCtx(ctx, u),
				apiserver.CreateRequest{Servers: []server.Server{s}},
			))
			key := res.Servers[0].Key
			Expect(key).ToNot(Equal(uuid.Nil()))
			Expect(retrieveServer(ctx, key).Name).To(Equal(s.Name))
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.HTTP.Servers.NewWriter(tx).Delete(ctx, key)
			})).To(Succeed())
		})

		It("Should not echo a status the client sent", func(ctx SpecContext) {
			grantWrites(ctx, svc.Rack.EmbeddedKey)
			s := newServer()
			s.Status = &server.Status{Message: "Server connected"}
			res := MustSucceed(create(
				testutil.AuthedCtx(ctx, u),
				apiserver.CreateRequest{Servers: []server.Server{s}},
			))
			Expect(res.Servers[0].Status).To(BeNil())
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.HTTP.Servers.NewWriter(tx).Delete(ctx, s.Key)
			})).To(Succeed())
		})

		It("Should reject the request without update access on the rack", func(
			ctx SpecContext,
		) {
			grant(ctx, u, allWrites, serverType, statusType)
			s := newServer()
			Expect(create(
				testutil.AuthedCtx(ctx, u),
				apiserver.CreateRequest{Servers: []server.Server{s}},
			)).Error().To(MatchError(access.ErrDenied))
		})

		It("Should reject the request when create is not granted", func(
			ctx SpecContext,
		) {
			Expect(create(
				testutil.AuthedCtx(ctx, u),
				apiserver.CreateRequest{Servers: []server.Server{newServer()}},
			)).Error().To(MatchError(access.ErrDenied))
		})

		It("Should reject an invalid server", func(ctx SpecContext) {
			grantWrites(ctx, svc.Rack.EmbeddedKey)
			Expect(create(
				testutil.AuthedCtx(ctx, u),
				apiserver.CreateRequest{Servers: []server.Server{
					newServer(
						func(s *server.Server) { s.Host = "" },
					),
				}},
			)).Error().To(MatchError(validate.ErrValidation))
		})
	})

	Describe("Update", func() {
		rename := func(s server.Server) apiserver.UpdateRequest {
			return apiserver.UpdateRequest{
				Key: s.Key,
				Update: server.Update{Variant: server.IdentityUpdate{
					Identity: server.Identity{
						Rack:                  s.Rack,
						Name:                  "Renamed",
						Scheme:                s.Scheme,
						Host:                  s.Host,
						Port:                  s.Port,
						BasePath:              s.BasePath,
						Timeout:               s.Timeout,
						MaxConcurrentRequests: s.MaxConcurrentRequests,
						Auth:                  s.Auth,
						HealthCheck:           s.HealthCheck,
					},
				}},
			}
		}

		It("Should write only the identity group", func(ctx SpecContext) {
			s := createServer(ctx, func(s *server.Server) {
				s.Reads = []server.Read{
					{
						Request: server.ReadRequest{
							Method: http.MethodGet,
							Path:   "/weather",
						},
						Pointer: "/temp",
						Index:   1,
						Channel: 2,
					},
				}
			})
			grantWrites(ctx, s.Rack)
			MustSucceed(update(testutil.AuthedCtx(ctx, u), rename(s)))
			got := retrieveServer(ctx, s.Key)
			Expect(got.Name).To(Equal("Renamed"))
			Expect(got.Reads).To(Equal(s.Reads))
		})

		It("Should write only the configuration group", func(ctx SpecContext) {
			s := createServer(ctx)
			grantWrites(ctx, s.Rack)
			MustSucceed(update(
				testutil.AuthedCtx(ctx, u),
				apiserver.UpdateRequest{
					Key: s.Key,
					Update: server.Update{Variant: server.ConfigurationUpdate{
						Configuration: server.Configuration{
							Writes: []server.Write{{
								Request: server.Request{
									Method: http.MethodPost,
									Path:   "/valve",
								},
								Channel: 2,
							}},
						},
					}},
				},
			))
			got := retrieveServer(ctx, s.Key)
			Expect(got.Writes).To(HaveLen(1))
			Expect(got.Writes[0].Channel).To(Equal(channel.Key(2)))
			Expect(got.Name).To(Equal(s.Name))
		})

		It("Should move a server only with update access on the new rack", func(
			ctx SpecContext,
		) {
			other := rack.Rack{Name: "Other"}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.Rack.NewWriter(tx).Create(ctx, &other)
			})).To(Succeed())
			DeferCleanup(func(ctx SpecContext) {
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.Rack.NewWriter(tx).Delete(ctx, nil, other.Key)
				})).To(Succeed())
			})
			s := createServer(ctx)
			move := apiserver.UpdateRequest{
				Key: s.Key,
				Update: server.Update{Variant: server.IdentityUpdate{
					Identity: server.Identity{
						Rack:                  other.Key,
						Name:                  s.Name,
						Scheme:                s.Scheme,
						Host:                  s.Host,
						Port:                  s.Port,
						BasePath:              s.BasePath,
						Timeout:               s.Timeout,
						MaxConcurrentRequests: s.MaxConcurrentRequests,
						Auth:                  s.Auth,
						HealthCheck:           s.HealthCheck,
					},
				}},
			}
			grantWrites(ctx, s.Rack)
			Expect(update(testutil.AuthedCtx(ctx, u), move)).
				Error().To(MatchError(access.ErrDenied))
			grant(ctx, u, []access.Action{access.ActionUpdate}, other.ResourceID())
			MustSucceed(update(testutil.AuthedCtx(ctx, u), move))
			Expect(retrieveServer(ctx, s.Key).Rack).To(Equal(other.Key))
		})

		It("Should reject a request with no group", func(ctx SpecContext) {
			s := createServer(ctx)
			grantWrites(ctx, s.Rack)
			Expect(update(
				testutil.AuthedCtx(ctx, u),
				apiserver.UpdateRequest{Key: s.Key},
			)).Error().To(MatchError(validate.ErrValidation))
		})

		It("Should reject the request when update is not granted", func(
			ctx SpecContext,
		) {
			s := createServer(ctx)
			Expect(update(testutil.AuthedCtx(ctx, u), rename(s))).
				Error().To(MatchError(access.ErrDenied))
			Expect(retrieveServer(ctx, s.Key).Name).To(Equal(s.Name))
		})
	})

	Describe("Delete", func() {
		It("Should remove the server when delete is granted", func(ctx SpecContext) {
			s := createServer(ctx)
			grantWrites(ctx, s.Rack)
			MustSucceed(del(
				testutil.AuthedCtx(ctx, u),
				apiserver.DeleteRequest{Keys: []server.Key{s.Key}},
			))
			Expect(svc.HTTP.Servers.NewRetrieve().
				Where(server.MatchKeys(s.Key)).
				Exists(ctx, db)).To(BeFalse())
		})

		It("Should reject the request when delete is not granted", func(
			ctx SpecContext,
		) {
			s := createServer(ctx)
			Expect(del(
				testutil.AuthedCtx(ctx, u),
				apiserver.DeleteRequest{Keys: []server.Key{s.Key}},
			)).Error().To(MatchError(access.ErrDenied))
			Expect(retrieveServer(ctx, s.Key).Key).To(Equal(s.Key))
		})
	})
})
