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
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Writer", func() {
	var (
		masters *master.Service
		rackSvc *rack.Service
		svc     *slave.Service
		stat    *status.Service
		tx      gorp.Tx
		w       slave.Writer
		m       master.Master
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
		masters = MustOpen(master.OpenService(ctx, master.ServiceConfig{
			DB:     db,
			Status: stat,
		}))
		svc = MustOpen(slave.OpenService(ctx, slave.ServiceConfig{
			DB:     db,
			Status: stat,
		}))
		tx = DeferClose(db.OpenTx())
		w = svc.NewWriter(tx)
		m = master.Master{
			Rack:      rackSvc.EmbeddedKey,
			Name:      "EtherCAT eth0",
			Interface: "eth0",
		}
		Expect(masters.NewWriter(tx).Create(ctx, &m)).To(Succeed())
	})

	newSlave := func() slave.Slave {
		return slave.Slave{
			Rack:        m.Rack,
			Name:        "EL3102",
			Serial:      1001,
			VendorID:    2,
			ProductCode: 0x0C1E3052,
			Master:      m.Key,
		}
	}
	create := func(ctx context.Context) slave.Slave {
		GinkgoHelper()
		s := newSlave()
		Expect(w.Create(ctx, &s)).To(Succeed())
		return s
	}
	retrieve := func(ctx context.Context, key slave.Key) slave.Slave {
		GinkgoHelper()
		var s slave.Slave
		Expect(svc.NewRetrieve().
			Where(slave.MatchKeys(key)).
			Entry(&s).
			Exec(ctx, tx)).To(Succeed())
		return s
	}
	statusOf := func(ctx context.Context, s slave.Slave) slave.Status {
		GinkgoHelper()
		var st slave.Status
		Expect(stat.NewRetrieve[slave.StatusDetails]().
			Where(status.MatchOwners[slave.StatusDetails](s.ResourceID())).
			Entry(&st).
			Exec(ctx, tx)).To(Succeed())
		return st
	}
	configure := func(c slave.Configuration) slave.Update {
		return slave.Update{Variant: slave.ConfigurationUpdate{Configuration: c}}
	}

	Describe("Create", func() {
		It("Should mint a key and write an unknown status", func(ctx SpecContext) {
			s := create(ctx)
			Expect(s.Key).ToNot(Equal(uuid.Nil()))
			Expect(retrieve(ctx, s.Key).Serial).To(Equal(uint32(1001)))
			st := statusOf(ctx, s)
			Expect(st.Message).To(Equal("Status unknown"))
			Expect(
				st.Details,
			).To(Equal(slave.StatusDetails{Rack: s.Rack, Slave: s.Key}))
		})

		It("Should give an existing status the name of a replaced slave", func(
			ctx SpecContext,
		) {
			s := create(ctx)
			running := statusOf(ctx, s)
			running.Variant = status.VariantSuccess
			running.Message = "Running"
			Expect(stat.NewWriter(tx).Set(ctx, &running)).To(Succeed())
			s.Name = "Replaced"
			Expect(w.Create(ctx, &s)).To(Succeed())
			st := statusOf(ctx, s)
			Expect(st.Message).To(Equal("Running"))
			Expect(st.Name).To(Equal("Replaced"))
			Expect(st.Details.Rack).To(Equal(s.Rack))
		})

		It("Should refuse a slave whose master does not exist", func(ctx SpecContext) {
			s := newSlave()
			s.Master = uuid.New()
			Expect(w.Create(ctx, &s)).To(MatchError(validate.ErrValidation))
		})
	})

	Describe("Apply", func() {
		It("Should write the identity and keep the configuration", func(
			ctx SpecContext,
		) {
			s := create(ctx)
			Expect(w.Apply(ctx, s.Key, configure(slave.Configuration{
				Name:       "Inlet",
				Identifier: "inlet",
			}))).To(Succeed())
			pdos := slave.PDOs{Inputs: []slave.PDOEntry{{
				Name:      "Value",
				Index:     0x6000,
				SubIndex:  0x11,
				BitLength: 16,
				DataType:  "int16",
			}}}
			Expect(w.Apply(ctx, s.Key, slave.Update{Variant: slave.IdentityUpdate{
				Identity: slave.Identity{
					Rack:        s.Rack,
					Serial:      2002,
					VendorID:    s.VendorID,
					ProductCode: s.ProductCode,
					Revision:    3,
					Master:      s.Master,
					Pdos:        pdos,
				},
			}})).To(Succeed())
			got := retrieve(ctx, s.Key)
			Expect(got.Serial).To(Equal(uint32(2002)))
			Expect(got.Revision).To(Equal(uint32(3)))
			Expect(got.Pdos).To(Equal(pdos))
			Expect(got.Name).To(Equal("Inlet"))
			Expect(got.Identifier).To(Equal("inlet"))
		})

		It("Should write the configuration and keep the identity", func(
			ctx SpecContext,
		) {
			s := create(ctx)
			bindings := []slave.ReadBinding{{Index: 0x6000, SubIndex: 0x11, Channel: 7}}
			Expect(w.Apply(ctx, s.Key, configure(slave.Configuration{
				Name:         "Inlet",
				Identifier:   "inlet",
				Disabled:     true,
				ReadIndex:    6,
				ReadBindings: bindings,
			}))).To(Succeed())
			got := retrieve(ctx, s.Key)
			Expect(got.Disabled).To(BeTrue())
			Expect(got.ReadBindings).To(Equal(bindings))
			Expect(got.Serial).To(Equal(s.Serial))
			Expect(got.Master).To(Equal(s.Master))
			Expect(statusOf(ctx, s).Name).To(Equal("Inlet"))
		})

		It("Should move the status with the rack", func(ctx SpecContext) {
			s := create(ctx)
			other := rack.Rack{Name: "Other"}
			Expect(rackSvc.NewWriter(tx).Create(ctx, &other)).To(Succeed())
			Expect(w.Apply(ctx, s.Key, slave.Update{Variant: slave.IdentityUpdate{
				Identity: slave.Identity{
					Rack:        other.Key,
					VendorID:    s.VendorID,
					ProductCode: s.ProductCode,
					Master:      s.Master,
				},
			}})).To(Succeed())
			Expect(statusOf(ctx, s).Details.Rack).To(Equal(other.Key))
		})

		It("Should write an unknown status when the status is missing", func(
			ctx SpecContext,
		) {
			s := create(ctx)
			Expect(stat.NewWriter(tx).DeleteOwned(ctx, s.ResourceID())).To(Succeed())
			Expect(w.Apply(ctx, s.Key, configure(slave.Configuration{
				Name: "Inlet",
			}))).To(Succeed())
			st := statusOf(ctx, s)
			Expect(st.Message).To(Equal("Status unknown"))
			Expect(st.Name).To(Equal("Inlet"))
		})

		It("Should reject a configuration without a name", func(ctx SpecContext) {
			s := create(ctx)
			Expect(w.Apply(ctx, s.Key, configure(slave.Configuration{}))).
				To(MatchError(validate.ErrValidation))
			Expect(retrieve(ctx, s.Key).Name).To(Equal(s.Name))
		})

		It("Should reject an identity without a master", func(ctx SpecContext) {
			s := create(ctx)
			Expect(w.Apply(ctx, s.Key, slave.Update{Variant: slave.IdentityUpdate{
				Identity: slave.Identity{Rack: s.Rack, VendorID: 2},
			}})).To(MatchError(validate.ErrValidation))
		})

		It("Should reject an update with no field group", func(ctx SpecContext) {
			s := create(ctx)
			Expect(w.Apply(ctx, s.Key, slave.Update{})).
				To(MatchError(validate.ErrValidation))
		})

		It("Should return not found for a missing slave", func(ctx SpecContext) {
			Expect(w.Apply(ctx, uuid.New(), configure(slave.Configuration{
				Name: "Inlet",
			}))).To(MatchError(query.ErrNotFound))
		})
	})

	Describe("Delete", func() {
		It("Should remove the slave and keep its key in the bus order", func(
			ctx SpecContext,
		) {
			s := create(ctx)
			Expect(masters.NewWriter(tx).Apply(ctx, m.Key, master.Update{
				Variant: master.IdentityUpdate{Identity: master.Identity{
					Rack:      m.Rack,
					Interface: m.Interface,
					Slaves:    []uuid.UUID{s.Key},
				}},
			})).To(Succeed())
			Expect(w.Delete(ctx, s.Key)).To(Succeed())
			Expect(svc.NewRetrieve().Where(slave.MatchKeys(s.Key)).Exists(ctx, tx)).
				To(BeFalse())
			var statuses []slave.Status
			Expect(stat.NewRetrieve[slave.StatusDetails]().
				Where(status.MatchOwners[slave.StatusDetails](s.ResourceID())).
				Entries(&statuses).
				Exec(ctx, tx)).To(Succeed())
			Expect(statuses).To(BeEmpty())
			var got master.Master
			Expect(masters.NewRetrieve().
				Where(master.MatchKeys(m.Key)).
				Entry(&got).
				Exec(ctx, tx)).To(Succeed())
			Expect(got.Slaves).To(Equal([]uuid.UUID{s.Key}))
		})

		It("Should succeed for a missing slave", func(ctx SpecContext) {
			Expect(w.Delete(ctx, uuid.New())).To(Succeed())
		})
	})
})
