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
	"github.com/synnaxlabs/synnax/pkg/service/ethercat/slave"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	racktestutil "github.com/synnaxlabs/synnax/pkg/service/rack/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Writer", func() {
	var (
		svc     *master.Service
		slaves  *slave.Service
		stat    *status.Service
		rackSvc *rack.Service
		tx      gorp.Tx
		w       master.Writer
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
		svc = MustOpen(master.OpenService(ctx, master.ServiceConfig{
			DB:     db,
			Status: stat,
		}))
		slaves = MustOpen(slave.OpenService(ctx, slave.ServiceConfig{DB: db}))
		tx = DeferClose(db.OpenTx())
		w = svc.NewWriter(tx)
	})

	newMaster := func() master.Master {
		return master.Master{
			Rack:      rackSvc.EmbeddedKey,
			Name:      "EtherCAT eth0",
			Interface: "eth0",
		}
	}
	create := func(ctx context.Context) master.Master {
		GinkgoHelper()
		m := newMaster()
		Expect(w.Create(ctx, &m)).To(Succeed())
		return m
	}
	retrieve := func(ctx context.Context, key master.Key) master.Master {
		GinkgoHelper()
		var m master.Master
		Expect(svc.NewRetrieve().
			Where(master.MatchKeys(key)).
			Entry(&m).
			Exec(ctx, tx)).To(Succeed())
		return m
	}
	statusOf := func(ctx context.Context, m master.Master) master.Status {
		GinkgoHelper()
		var s master.Status
		Expect(stat.NewRetrieve[master.StatusDetails]().
			Where(status.MatchOwners[master.StatusDetails](m.ResourceID())).
			Entry(&s).
			Exec(ctx, tx)).To(Succeed())
		return s
	}

	Describe("Create", func() {
		It("Should mint a key and write an unknown status", func(ctx SpecContext) {
			m := create(ctx)
			Expect(m.Key).ToNot(Equal(uuid.Nil()))
			Expect(retrieve(ctx, m.Key).Interface).To(Equal("eth0"))
			s := statusOf(ctx, m)
			Expect(s.Variant).To(Equal(status.VariantWarning))
			Expect(s.Message).To(Equal("Status unknown"))
			Expect(s.Details).
				To(Equal(master.StatusDetails{Rack: m.Rack, Master: m.Key}))
		})

		It("Should reject a master without an interface", func(ctx SpecContext) {
			m := newMaster()
			m.Interface = ""
			Expect(w.Create(ctx, &m)).To(MatchError(validate.ErrValidation))
		})

		It("Should give an existing status the name of a replaced master", func(
			ctx SpecContext,
		) {
			m := create(ctx)
			running := statusOf(ctx, m)
			running.Variant = status.VariantSuccess
			running.Message = "Running"
			Expect(stat.NewWriter(tx).Set(ctx, &running)).To(Succeed())
			m.Name = "Replaced"
			Expect(w.Create(ctx, &m)).To(Succeed())
			s := statusOf(ctx, m)
			Expect(s.Message).To(Equal("Running"))
			Expect(s.Name).To(Equal("Replaced"))
			Expect(s.Details.Rack).To(Equal(m.Rack))
		})

		It("Should reject a bus order that repeats a slave", func(ctx SpecContext) {
			m := newMaster()
			k := uuid.New()
			m.Slaves = []uuid.UUID{k, k}
			Expect(w.Create(ctx, &m)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("slaves: must not repeat a key")),
			))
		})

		It("Should refuse a master on a rack that does not exist", func(
			ctx SpecContext,
		) {
			m := newMaster()
			m.Rack = rack.NewKey(9, 9)
			Expect(w.Create(ctx, &m)).To(MatchError(validate.ErrValidation))
		})
	})

	Describe("Update", func() {
		It("Should rename the status with the master", func(ctx SpecContext) {
			m := create(ctx)
			Expect(w.Update(ctx, m.Key, func(m master.Master) (master.Master, error) {
				m.Name = "Renamed"
				return m, nil
			})).To(Succeed())
			Expect(statusOf(ctx, m).Name).To(Equal("Renamed"))
		})

		It("Should move the status with the rack", func(ctx SpecContext) {
			m := create(ctx)
			other := rack.Rack{Name: "Other"}
			Expect(rackSvc.NewWriter(tx).Create(ctx, &other)).To(Succeed())
			Expect(w.Update(ctx, m.Key, func(m master.Master) (master.Master, error) {
				m.Rack = other.Key
				return m, nil
			})).To(Succeed())
			Expect(statusOf(ctx, m).Details.Rack).To(Equal(other.Key))
		})

		It("Should write an unknown status when the status is missing", func(
			ctx SpecContext,
		) {
			m := create(ctx)
			Expect(stat.NewWriter(tx).DeleteOwned(ctx, m.ResourceID())).To(Succeed())
			Expect(w.Update(ctx, m.Key, func(m master.Master) (master.Master, error) {
				m.Name = "Renamed"
				return m, nil
			})).To(Succeed())
			s := statusOf(ctx, m)
			Expect(s.Message).To(Equal("Status unknown"))
			Expect(s.Name).To(Equal("Renamed"))
		})

		It("Should return the error change returns", func(ctx SpecContext) {
			m := create(ctx)
			err := errors.New("change failed")
			Expect(w.Update(ctx, m.Key, func(master.Master) (master.Master, error) {
				return master.Master{}, err
			})).To(MatchError(err))
		})

		It("Should return not found for a missing master", func(ctx SpecContext) {
			Expect(
				w.Update(ctx, uuid.New(), func(m master.Master) (master.Master, error) {
					return m, nil
				}),
			).To(MatchError(query.ErrNotFound))
		})
	})

	Describe("Apply", func() {
		It("Should write the identity and keep the configuration", func(
			ctx SpecContext,
		) {
			m := create(ctx)
			order := []uuid.UUID{uuid.New(), uuid.New()}
			Expect(w.Apply(ctx, m.Key, master.Update{Variant: master.IdentityUpdate{
				Identity: master.Identity{
					Rack:      m.Rack,
					Interface: "eth1",
					Slaves:    order,
				},
			}})).To(Succeed())
			got := retrieve(ctx, m.Key)
			Expect(got.Interface).To(Equal("eth1"))
			Expect(got.Slaves).To(Equal(order))
			Expect(got.Name).To(Equal(m.Name))
		})

		It("Should write the configuration and keep the identity", func(
			ctx SpecContext,
		) {
			m := create(ctx)
			Expect(w.Apply(ctx, m.Key, master.Update{
				Variant: master.ConfigurationUpdate{
					Configuration: master.Configuration{Name: "Bench bus"},
				},
			})).To(Succeed())
			got := retrieve(ctx, m.Key)
			Expect(got.Name).To(Equal("Bench bus"))
			Expect(got.Interface).To(Equal(m.Interface))
		})

		It("Should reject a bus order that repeats a slave", func(ctx SpecContext) {
			m := create(ctx)
			k := uuid.New()
			Expect(w.Apply(ctx, m.Key, master.Update{Variant: master.IdentityUpdate{
				Identity: master.Identity{
					Rack:      m.Rack,
					Interface: m.Interface,
					Slaves:    []uuid.UUID{k, uuid.New(), k},
				},
			}})).To(MatchError(validate.ErrValidation))
			Expect(retrieve(ctx, m.Key).Slaves).To(BeEmpty())
		})

		It("Should reject an identity without an interface", func(ctx SpecContext) {
			m := create(ctx)
			Expect(w.Apply(ctx, m.Key, master.Update{Variant: master.IdentityUpdate{
				Identity: master.Identity{Rack: m.Rack},
			}})).To(MatchError(validate.ErrValidation))
		})

		It("Should reject an update with no field group", func(ctx SpecContext) {
			m := create(ctx)
			Expect(w.Apply(ctx, m.Key, master.Update{})).
				To(MatchError(validate.ErrValidation))
		})

		It("Should return not found for a missing master", func(ctx SpecContext) {
			Expect(w.Apply(ctx, uuid.New(), master.Update{
				Variant: master.ConfigurationUpdate{
					Configuration: master.Configuration{Name: "Bus"},
				},
			})).To(MatchError(query.ErrNotFound))
		})
	})

	Describe("Delete", func() {
		It("Should remove the master with its slaves and status", func(
			ctx SpecContext,
		) {
			m := create(ctx)
			s := slave.Slave{
				Key:         uuid.New(),
				Rack:        m.Rack,
				Name:        "EL3102",
				VendorID:    2,
				ProductCode: 0x0C1E3052,
				Master:      m.Key,
			}
			Expect(gorp.NewCreate[slave.Key, slave.Slave]().Entry(&s).Exec(ctx, tx)).
				To(Succeed())
			Expect(w.Delete(ctx, m.Key)).To(Succeed())
			Expect(svc.NewRetrieve().Where(master.MatchKeys(m.Key)).Exists(ctx, tx)).
				To(BeFalse())
			Expect(slaves.NewRetrieve().Where(slave.MatchKeys(s.Key)).Exists(ctx, tx)).
				To(BeFalse())
			var statuses []master.Status
			Expect(stat.NewRetrieve[master.StatusDetails]().
				Where(status.MatchOwners[master.StatusDetails](m.ResourceID())).
				Entries(&statuses).
				Exec(ctx, tx)).To(Succeed())
			Expect(statuses).To(BeEmpty())
		})

		It("Should succeed for a missing master", func(ctx SpecContext) {
			Expect(w.Delete(ctx, uuid.New())).To(Succeed())
		})
	})
})
