// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package master_test

import (
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/ethercat/master"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	racktestutil "github.com/synnaxlabs/synnax/pkg/service/rack/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/search"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Service", func() {
	var (
		svc     *master.Service
		stat    *status.Service
		rackSvc *rack.Service
		tx      gorp.Tx
		rackKey rack.Key
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
		rackKey = rackSvc.EmbeddedKey
		svc = MustOpen(master.OpenService(ctx, master.ServiceConfig{
			DB:     db,
			Status: stat,
		}))
		tx = DeferClose(db.OpenTx())
	})

	create := func(ctx context.Context, iface string) master.Master {
		GinkgoHelper()
		m := master.Master{
			Key:       uuid.New(),
			Rack:      rackKey,
			Name:      "EtherCAT " + iface,
			Interface: iface,
		}
		Expect(gorp.NewCreate[master.Key, master.Master]().Entry(&m).Exec(ctx, tx)).
			To(Succeed())
		return m
	}

	Describe("OpenService", func() {
		It("Should reject a config missing the DB", func(ctx SpecContext) {
			Expect(master.OpenService(ctx, master.ServiceConfig{})).Error().
				To(MatchError(ContainSubstring("db: must be non-nil")))
		})

		It("Should reject a config missing the status service", func(ctx SpecContext) {
			Expect(master.OpenService(ctx, master.ServiceConfig{DB: db})).Error().
				To(MatchError(ContainSubstring("status: must be non-nil")))
		})

		It("Should let a later config override an earlier one", func() {
			cfg := master.ServiceConfig{DB: db}.
				Override(master.ServiceConfig{Status: stat})
			Expect(cfg.DB).To(BeIdenticalTo(db))
			Expect(cfg.Status).To(BeIdenticalTo(stat))
			Expect(cfg.Validate()).To(Succeed())
		})
	})

	Describe("Retrieve", func() {
		It("Should match masters by interface", func(ctx SpecContext) {
			m := create(ctx, "eth0")
			create(ctx, "eth1")
			var got []master.Master
			Expect(svc.NewRetrieve().
				Where(master.MatchInterfaces("eth0")).
				Entries(&got).
				Exec(ctx, tx)).To(Succeed())
			Expect(got).To(HaveLen(1))
			Expect(got[0].Key).To(Equal(m.Key))
		})
	})

	Describe("Rack", func() {
		It("Should refuse to delete a rack that holds a master", func(ctx SpecContext) {
			r := rack.Rack{Name: "Bench"}
			Expect(rackSvc.NewWriter(tx).Create(ctx, &r)).To(Succeed())
			rackKey = r.Key
			m := create(ctx, "eth4")
			Expect(rackSvc.NewWriter(tx).Delete(ctx, nil, r.Key)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("because ethercat_master")),
			))
			var got master.Master
			Expect(svc.NewRetrieve().
				Where(master.MatchKeys(m.Key)).
				Entry(&got).
				Exec(ctx, tx)).To(Succeed())
			Expect(got.Rack).To(Equal(r.Key))
		})
	})

	Describe("Resources", func() {
		It("Should name each master by its name", func(ctx SpecContext) {
			m := create(ctx, "eth2")
			Expect(svc.Resources().Retrieve(ctx, tx, []resource.ID{m.ResourceID()})).
				To(Equal([]resource.Resource{{ID: m.ResourceID(), Name: m.Name}}))
		})

		It("Should reject a key that is not a UUID", func(ctx SpecContext) {
			Expect(svc.Resources().Retrieve(ctx, tx, []resource.ID{{
				Type: resource.TypeEthercatMaster,
				Key:  "not-a-uuid",
			}})).Error().To(MatchError(validate.ErrValidation))
		})

		It("Should find a master by its name", func(ctx SpecContext) {
			m := create(ctx, "eth3")
			res := MustSucceed(svc.Resources().Search(
				ctx,
				tx,
				search.Query[resource.ID]{Term: m.Name, Limit: 10},
			))
			Expect(res.Keys).To(ContainElement(m.ResourceID()))
		})
	})
})
