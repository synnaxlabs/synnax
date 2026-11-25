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

var serverType = resource.ID{Type: resource.TypeOpcuaServer}

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
	createServer := func(ctx SpecContext, overrides ...func(*server.Server)) server.Server {
		GinkgoHelper()
		s := newServer(overrides...)
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return gorp.NewCreate[server.Key, server.Server]().Entry(&s).Exec(ctx, tx)
		})).To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return gorp.NewDelete[server.Key, server.Server]().
					Where(gorp.MatchKeys[server.Key, server.Server](s.Key)).
					Exec(ctx, tx)
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
})
