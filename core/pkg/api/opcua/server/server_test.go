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
	apiserver "github.com/synnaxlabs/synnax/pkg/api/opcua/server"
	"github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	channeltestutil "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/opcua/server"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/filter"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var (
	serverType = resource.ID{Type: resource.TypeOpcuaServer}
	statusType = resource.ID{Type: resource.TypeStatus}
	// channelType lets a user bind any channel, which a write needs to use it.
	channelType = resource.ID{Type: resource.TypeChannel}
	allWrites   = []access.Action{
		access.ActionCreate,
		access.ActionUpdate,
		access.ActionDelete,
	}
)

var _ = Describe("Service", func() {
	var u user.User
	BeforeEach(func(ctx SpecContext) { u = createUser(ctx) })

	newServer := func(overrides ...func(*server.Server)) server.Server {
		s := server.Server{
			Key:        uuid.New(),
			Rack:       svc.Rack.EmbeddedKey,
			Name:       "PLC-" + uuid.New().String(),
			Connection: newConnection("opc.tcp://plc:4840"),
		}
		for _, override := range overrides {
			override(&s)
		}
		return s
	}
	grantRetrieve := func(ctx SpecContext) {
		GinkgoHelper()
		grant(ctx, u, []access.Action{access.ActionRetrieve}, serverType)
	}
	// grantWrites lets u write any OPC UA server, its status, and the rack it sits on.
	grantWrites := func(ctx SpecContext, r rack.Key) {
		GinkgoHelper()
		grant(ctx, u, allWrites, serverType, statusType)
		grant(ctx, u, []access.Action{access.ActionUpdate}, r.ResourceID())
	}
	retrieveServer := func(ctx SpecContext, key server.Key) server.Server {
		GinkgoHelper()
		var s server.Server
		Expect(svc.OPCUA.Servers.NewRetrieve().
			Where(server.MatchKeys(key)).
			Entry(&s).
			Exec(ctx, db)).To(Succeed())
		return s
	}
	createServer := func(ctx SpecContext, overrides ...func(*server.Server)) server.Server {
		GinkgoHelper()
		s := newServer(overrides...)
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.OPCUA.Servers.NewWriter(tx).Create(ctx, &s)
		})).To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.OPCUA.Servers.NewWriter(tx).Delete(ctx, s.Key)
			})).To(Succeed())
		})
		return s
	}

	Describe("NewService", func() {
		DescribeTable("Should reject a config missing a required field",
			func(field string, unset func(*apiserver.ServiceConfig)) {
				cfg := apiserver.ServiceConfig{
					Server: svc.OPCUA.Servers,
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
			base := apiserver.ServiceConfig{Server: svc.OPCUA.Servers}
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
				return svc.OPCUA.Servers.NewWriter(tx).Delete(ctx, key)
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
				return svc.OPCUA.Servers.NewWriter(tx).Delete(ctx, s.Key)
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
					newServer(func(s *server.Server) { s.Connection.Endpoint = "" }),
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
						Rack:       s.Rack,
						Name:       "Renamed",
						Connection: s.Connection,
					},
				}},
			}
		}

		It("Should write only the identity group", func(ctx SpecContext) {
			keys := MustSucceed(channeltestutil.CreateVirtual(ctx, db, svc.Channel, 2))
			s := createServer(ctx, func(s *server.Server) {
				s.Reads = []server.Read{
					{Node: "nsu=urn:plc;s=Temp", Index: keys[0], Channel: keys[1]},
				}
			})
			grantWrites(ctx, s.Rack)
			MustSucceed(update(testutil.AuthedCtx(ctx, u), rename(s)))
			got := retrieveServer(ctx, s.Key)
			Expect(got.Name).To(Equal("Renamed"))
			Expect(got.Reads).To(Equal(s.Reads))
		})

		// bindValve returns an update that binds the node Valve to the channel ch.
		bindValve := func(s server.Server, ch channel.Key) apiserver.UpdateRequest {
			return apiserver.UpdateRequest{
				Key: s.Key,
				Update: server.Update{Variant: server.ConfigurationUpdate{
					Configuration: server.Configuration{
						Writes: []server.Write{
							{Node: "nsu=urn:plc;s=Valve", Channel: ch},
						},
					},
				}},
			}
		}

		It("Should write only the configuration group", func(ctx SpecContext) {
			ch := MustSucceed(channeltestutil.CreateVirtual(ctx, db, svc.Channel, 1))[0]
			s := createServer(ctx)
			grantWrites(ctx, s.Rack)
			grant(ctx, u, []access.Action{access.ActionRetrieve}, channelType)
			MustSucceed(update(testutil.AuthedCtx(ctx, u), bindValve(s, ch)))
			got := retrieveServer(ctx, s.Key)
			Expect(
				got.Writes,
			).To(Equal([]server.Write{{Node: "nsu=urn:plc;s=Valve", Channel: ch}}))
			Expect(got.Name).To(Equal(s.Name))
		})

		It("Should reject a binding to a channel the user cannot retrieve", func(
			ctx SpecContext,
		) {
			ch := MustSucceed(channeltestutil.CreateVirtual(ctx, db, svc.Channel, 1))[0]
			s := createServer(ctx)
			grantWrites(ctx, s.Rack)
			Expect(update(testutil.AuthedCtx(ctx, u), bindValve(s, ch))).
				Error().To(MatchError(access.ErrDenied))
			Expect(retrieveServer(ctx, s.Key).Writes).To(BeEmpty())
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
			Expect(svc.OPCUA.Servers.NewRetrieve().
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
