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
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/opcua/server"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	racktestutil "github.com/synnaxlabs/synnax/pkg/service/rack/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/search"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Service", func() {
	var (
		rackSvc *rack.Service
		svc     *server.Service
		tx      gorp.Tx
	)
	BeforeEach(func(ctx SpecContext) {
		MustOpen(group.OpenService(ctx, group.ServiceConfig{DB: db}))
		MustOpen(label.OpenService(ctx, label.ServiceConfig{DB: db}))
		stat := MustOpen(status.OpenService(ctx, status.ServiceConfig{DB: db}))
		roleSvc := MustOpen(role.OpenService(ctx, role.ServiceConfig{DB: db}))
		rackSvc = MustOpen(rack.OpenService(ctx, rack.ServiceConfig{
			DB:         db,
			HostKey:    1,
			Status:     stat,
			DriverRole: MustSucceed(racktestutil.CreateDriverRole(ctx, db, roleSvc)),
		}))
		svc = MustOpen(server.OpenService(ctx, server.ServiceConfig{DB: db}))
		tx = DeferClose(db.OpenTx())
	})

	create := func(ctx context.Context, name string) server.Server {
		GinkgoHelper()
		s := server.Server{
			Key:        uuid.New(),
			Rack:       rackSvc.EmbeddedKey,
			Name:       name,
			Connection: newConnection("opc.tcp://plc:4840"),
		}
		Expect(gorp.NewCreate[server.Key, server.Server]().
			Entry(&s).
			Exec(ctx, tx)).To(Succeed())
		return s
	}

	Describe("OpenService", func() {
		It("Should reject a config without a database", func(ctx SpecContext) {
			Expect(server.OpenService(ctx, server.ServiceConfig{})).Error().
				To(MatchError(ContainSubstring("db: must be non-nil")))
		})

		It("Should let a later config override an earlier one", func() {
			cfg := server.ServiceConfig{}.Override(server.ServiceConfig{DB: db})
			Expect(cfg.DB).To(BeIdenticalTo(db))
			Expect(cfg.Validate()).To(Succeed())
		})
	})

	Describe("NewRetrieve", func() {
		It("Should retrieve a stored server by its key", func(ctx SpecContext) {
			s := create(ctx, "Boiler PLC")
			var res server.Server
			Expect(svc.NewRetrieve().
				Where(server.MatchKeys(s.Key)).
				Entry(&res).
				Exec(ctx, tx)).To(Succeed())
			Expect(res).To(Equal(s))
		})

		It("Should return not found for a missing key", func(ctx SpecContext) {
			var res server.Server
			Expect(svc.NewRetrieve().
				Where(server.MatchKeys(uuid.New())).
				Entry(&res).
				Exec(ctx, tx)).To(MatchError(query.ErrNotFound))
		})
	})

	Describe("Resources", func() {
		It("Should name a server by its resource ID", func(ctx SpecContext) {
			s := create(ctx, "Boiler PLC")
			Expect(svc.Resources().Retrieve(ctx, tx, []resource.ID{s.ResourceID()})).
				To(Equal([]resource.Resource{{ID: s.ResourceID(), Name: s.Name}}))
		})

		It("Should find a server by its name", func(ctx SpecContext) {
			s := create(ctx, "Boiler PLC")
			res := MustSucceed(svc.Resources().Search(
				ctx,
				tx,
				search.Query[resource.ID]{Term: "boiler", Limit: 10},
			))
			Expect(res.Keys).To(ContainElement(s.ResourceID()))
		})

		It("Should reject a resource ID with a malformed key", func(ctx SpecContext) {
			Expect(svc.Resources().Retrieve(ctx, tx, []resource.ID{{
				Type: resource.TypeOpcuaServer,
				Key:  "plc",
			}})).Error().To(MatchError(validate.ErrValidation))
		})
	})
})
