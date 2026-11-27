// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package slave_test

import (
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/ethercat/master"
	"github.com/synnaxlabs/synnax/pkg/service/ethercat/slave"
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
		svc     *slave.Service
		rackSvc *rack.Service
		tx      gorp.Tx
		m       master.Master
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
		MustOpen(master.OpenService(ctx, master.ServiceConfig{DB: db}))
		svc = MustOpen(slave.OpenService(ctx, slave.ServiceConfig{DB: db}))
		tx = DeferClose(db.OpenTx())
		m = master.Master{
			Key:       uuid.New(),
			Rack:      rackSvc.EmbeddedKey,
			Name:      "EtherCAT eth0",
			Interface: "eth0",
		}
		Expect(gorp.NewCreate[master.Key, master.Master]().Entry(&m).Exec(ctx, tx)).
			To(Succeed())
	})

	create := func(ctx context.Context, serial uint32) slave.Slave {
		GinkgoHelper()
		s := slave.Slave{
			Key:         uuid.New(),
			Rack:        m.Rack,
			Name:        "EL3102",
			Serial:      serial,
			VendorID:    2,
			ProductCode: 0x0C1E3052,
			Master:      m.Key,
		}
		Expect(gorp.NewCreate[slave.Key, slave.Slave]().Entry(&s).Exec(ctx, tx)).
			To(Succeed())
		return s
	}

	Describe("OpenService", func() {
		It("Should reject a config missing the DB", func(ctx SpecContext) {
			Expect(slave.OpenService(ctx, slave.ServiceConfig{})).Error().
				To(MatchError(ContainSubstring("db: must be non-nil")))
		})

		It("Should let a later config override an earlier one", func() {
			cfg := slave.ServiceConfig{}.Override(slave.ServiceConfig{DB: db})
			Expect(cfg.DB).To(BeIdenticalTo(db))
			Expect(cfg.Validate()).To(Succeed())
		})
	})

	Describe("Retrieve", func() {
		It("Should match slaves by serial", func(ctx SpecContext) {
			s := create(ctx, 1001)
			create(ctx, 1002)
			var got []slave.Slave
			Expect(svc.NewRetrieve().
				Where(slave.MatchSerials(1001)).
				Entries(&got).
				Exec(ctx, tx)).To(Succeed())
			Expect(got).To(HaveLen(1))
			Expect(got[0].Key).To(Equal(s.Key))
		})

		It("Should match the slaves of a master", func(ctx SpecContext) {
			s := create(ctx, 1003)
			var got []slave.Slave
			Expect(svc.NewRetrieve().
				Where(slave.MatchMasters(m.Key)).
				Entries(&got).
				Exec(ctx, tx)).To(Succeed())
			Expect(got).To(ConsistOf(HaveField("Key", s.Key)))
		})
	})

	Describe("Rack", func() {
		It("Should refuse to delete a rack that holds a slave", func(ctx SpecContext) {
			r := rack.Rack{Name: "Bench"}
			Expect(rackSvc.NewWriter(tx).Create(ctx, &r)).To(Succeed())
			s := create(ctx, 1007)
			s.Rack = r.Key
			Expect(gorp.NewCreate[slave.Key, slave.Slave]().Entry(&s).Exec(ctx, tx)).
				To(Succeed())
			Expect(rackSvc.NewWriter(tx).Delete(ctx, nil, r.Key)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("because ethercat_slave")),
			))
		})
	})

	Describe("Resources", func() {
		It("Should name each slave by its name", func(ctx SpecContext) {
			s := create(ctx, 1005)
			Expect(svc.Resources().Retrieve(ctx, tx, []resource.ID{s.ResourceID()})).
				To(Equal([]resource.Resource{{ID: s.ResourceID(), Name: s.Name}}))
		})

		It("Should reject a key that is not a UUID", func(ctx SpecContext) {
			Expect(svc.Resources().Retrieve(ctx, tx, []resource.ID{{
				Type: resource.TypeEthercatSlave,
				Key:  "not-a-uuid",
			}})).Error().To(MatchError(validate.ErrValidation))
		})

		It("Should find a slave by its name", func(ctx SpecContext) {
			s := create(ctx, 1006)
			res := MustSucceed(svc.Resources().Search(
				ctx,
				tx,
				search.Query[resource.ID]{Term: s.Name, Limit: 10},
			))
			Expect(res.Keys).To(ContainElement(s.ResourceID()))
		})
	})
})
