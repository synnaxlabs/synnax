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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	apislave "github.com/synnaxlabs/synnax/pkg/api/ethercat/slave"
	"github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/ethercat/master"
	"github.com/synnaxlabs/synnax/pkg/service/ethercat/slave"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/filter"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var (
	slaveType  = resource.ID{Type: resource.TypeEthercatSlave}
	statusType = resource.ID{Type: resource.TypeStatus}
	allWrites  = []access.Action{
		access.ActionCreate,
		access.ActionUpdate,
		access.ActionDelete,
	}
)

var _ = Describe("Service", func() {
	var (
		u      user.User
		m      master.Master
		serial uint64
	)
	BeforeEach(func(ctx SpecContext) {
		u = createUser(ctx)
		m = master.Master{
			Key:       uuid.New(),
			Rack:      svc.Rack.EmbeddedKey,
			Name:      "EtherCAT bus",
			Interface: "eth0",
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.EtherCAT.Masters.NewWriter(tx).Create(ctx, &m)
		})).To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.EtherCAT.Masters.NewWriter(tx).Delete(ctx, m.Key)
			})).To(Succeed())
		})
	})

	newSlave := func(overrides ...func(*slave.Slave)) slave.Slave {
		serial++
		s := slave.Slave{
			Key:         uuid.New(),
			Rack:        m.Rack,
			Name:        "EL3102",
			Serial:      serial,
			VendorID:    2,
			ProductCode: 0x0C1E3052,
			Master:      m.Key,
		}
		for _, override := range overrides {
			override(&s)
		}
		return s
	}
	// grantWrites lets u write any slave and its status, and update the rack and
	// master that the slave sits under.
	grantWrites := func(ctx SpecContext) {
		GinkgoHelper()
		grant(ctx, u, allWrites, slaveType, statusType)
		grant(
			ctx,
			u,
			[]access.Action{access.ActionUpdate},
			m.Rack.ResourceID(),
			m.ResourceID(),
		)
	}
	retrieveSlave := func(ctx SpecContext, key slave.Key) slave.Slave {
		GinkgoHelper()
		var s slave.Slave
		Expect(svc.EtherCAT.Slaves.NewRetrieve().
			Where(slave.MatchKeys(key)).
			Entry(&s).
			Exec(ctx, db)).To(Succeed())
		return s
	}
	writeSlave := func(ctx SpecContext) slave.Slave {
		GinkgoHelper()
		s := newSlave()
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.EtherCAT.Slaves.NewWriter(tx).Create(ctx, &s)
		})).To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.EtherCAT.Slaves.NewWriter(tx).Delete(ctx, s.Key)
			})).To(Succeed())
		})
		return s
	}
	rename := func(key slave.Key, name string) apislave.UpdateRequest {
		return apislave.UpdateRequest{
			Key: key,
			Update: slave.Update{Variant: slave.ConfigurationUpdate{
				Configuration: slave.Configuration{Name: name, Identifier: "inlet"},
			}},
		}
	}
	grantRetrieve := func(ctx SpecContext) {
		GinkgoHelper()
		grant(ctx, u, []access.Action{access.ActionRetrieve}, slaveType)
	}
	createSlave := func(ctx SpecContext, overrides ...func(*slave.Slave)) slave.Slave {
		GinkgoHelper()
		s := newSlave(overrides...)
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return gorp.NewCreate[slave.Key, slave.Slave]().Entry(&s).Exec(ctx, tx)
		})).To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return gorp.NewDelete[slave.Key, slave.Slave]().
					Where(gorp.MatchKeys[slave.Key, slave.Slave](s.Key)).
					Exec(ctx, tx)
			})).To(Succeed())
		})
		return s
	}

	Describe("NewService", func() {
		DescribeTable("Should reject a config missing a required field",
			func(field string, unset func(*apislave.ServiceConfig)) {
				cfg := apislave.ServiceConfig{
					Slave:  svc.EtherCAT.Slaves,
					Status: svc.Status,
				}
				unset(&cfg)
				Expect(apislave.NewService(cfg)).Error().
					To(MatchError(ContainSubstring(field + ": must be non-nil")))
			},
			Entry("slave", "slave", func(c *apislave.ServiceConfig) {
				c.Slave = nil
			}),
			Entry("status", "status", func(c *apislave.ServiceConfig) {
				c.Status = nil
			}),
		)

		It("Should let a later config override an earlier one", func() {
			base := apislave.ServiceConfig{Slave: svc.EtherCAT.Slaves}
			Expect(apislave.NewService(base, apislave.ServiceConfig{
				Status: svc.Status,
			})).ToNot(BeNil())
		})
	})

	Describe("Create", func() {
		It("Should mint a key when writes are granted", func(ctx SpecContext) {
			grantWrites(ctx)
			s := newSlave()
			s.Key = uuid.Nil()
			res := MustSucceed(create(
				testutil.AuthedCtx(ctx, u),
				apislave.CreateRequest{Slaves: []slave.Slave{s}},
			))
			Expect(res.Slaves[0].Key).ToNot(Equal(uuid.Nil()))
			Expect(retrieveSlave(ctx, res.Slaves[0].Key).Master).To(Equal(m.Key))
		})

		It("Should not echo a status the client sent", func(ctx SpecContext) {
			grantWrites(ctx)
			s := newSlave(func(s *slave.Slave) {
				s.Status = &slave.Status{Message: "Connected"}
			})
			res := MustSucceed(create(
				testutil.AuthedCtx(ctx, u),
				apislave.CreateRequest{Slaves: []slave.Slave{s}},
			))
			Expect(res.Slaves[0].Status).To(BeNil())
		})

		It("Should not create a slave without update access on its rack", func(
			ctx SpecContext,
		) {
			grant(ctx, u, allWrites, slaveType, statusType)
			grant(ctx, u, []access.Action{access.ActionUpdate}, m.ResourceID())
			Expect(create(
				testutil.AuthedCtx(ctx, u),
				apislave.CreateRequest{Slaves: []slave.Slave{newSlave()}},
			)).Error().To(MatchError(access.ErrDenied))
		})

		It("Should not create a slave without update access on its master", func(
			ctx SpecContext,
		) {
			grant(ctx, u, allWrites, slaveType, statusType)
			grant(ctx, u, []access.Action{access.ActionUpdate}, m.Rack.ResourceID())
			Expect(create(
				testutil.AuthedCtx(ctx, u),
				apislave.CreateRequest{Slaves: []slave.Slave{newSlave()}},
			)).Error().To(MatchError(access.ErrDenied))
		})
	})

	Describe("Retrieve", func() {
		It("Should reject a named slave when retrieve is not granted", func(
			ctx SpecContext,
		) {
			key := createSlave(ctx).Key
			Expect(retrieve(
				testutil.AuthedCtx(ctx, u),
				apislave.RetrieveRequest{
					Where: &slave.Where{Key: &filter.Set[slave.Key]{Eq: &key}},
				},
			)).Error().To(MatchError(access.ErrDenied))
		})

		It("Should filter by key and attach the status", func(ctx SpecContext) {
			s := createSlave(ctx)
			createSlave(ctx)
			st := slave.Status{
				Owner:   new(s.ResourceID()),
				Name:    s.Name,
				Time:    telem.Now(),
				Variant: status.VariantSuccess,
				Message: "Running",
				Details: slave.StatusDetails{Slave: s.Key},
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.Status.NewWriter(tx).Set(ctx, &st)
			})).To(Succeed())
			grantRetrieve(ctx)
			res := MustSucceed(retrieve(
				testutil.AuthedCtx(ctx, u),
				apislave.RetrieveRequest{
					Where: &slave.Where{
						Key: &filter.Set[slave.Key]{Eq: &s.Key},
					},
					IncludeStatus: true,
				},
			))
			Expect(res.Slaves).To(HaveLen(1))
			Expect(res.Slaves[0].Key).To(Equal(s.Key))
			Expect(res.Slaves[0].Status.Message).To(Equal("Running"))
		})

		It("Should leave out the slaves the subject cannot retrieve", func(
			ctx SpecContext,
		) {
			createSlave(ctx)
			res := MustSucceed(retrieve(
				testutil.AuthedCtx(ctx, u),
				apislave.RetrieveRequest{
					Where: &slave.Where{Master: &filter.Set[master.Key]{Eq: &m.Key}},
				},
			))
			Expect(res.Slaves).To(BeEmpty())
		})

		It("Should find a slave by its name", func(ctx SpecContext) {
			s := createSlave(ctx, func(s *slave.Slave) { s.Name = "Pump module" })
			grantRetrieve(ctx)
			res := MustSucceed(retrieve(
				testutil.AuthedCtx(ctx, u),
				apislave.RetrieveRequest{SearchTerm: "pump module"},
			))
			Expect(lo.Map(res.Slaves, func(s slave.Slave, _ int) slave.Key {
				return s.Key
			})).To(ContainElement(s.Key))
		})

		It("Should cap the results at the limit", func(ctx SpecContext) {
			a := createSlave(ctx)
			b := createSlave(ctx)
			grantRetrieve(ctx)
			res := MustSucceed(retrieve(
				testutil.AuthedCtx(ctx, u),
				apislave.RetrieveRequest{
					Where: &slave.Where{Key: &filter.Set[slave.Key]{
						In: []slave.Key{a.Key, b.Key},
					}},
					Limit: new(uint32(1)),
				},
			))
			Expect(res.Slaves).To(HaveLen(1))
		})

		It("Should page past the after key", func(ctx SpecContext) {
			a := createSlave(ctx)
			b := createSlave(ctx)
			grantRetrieve(ctx)
			authed := testutil.AuthedCtx(ctx, u)
			req := apislave.RetrieveRequest{
				Where: &slave.Where{Key: &filter.Set[slave.Key]{
					In: []slave.Key{a.Key, b.Key},
				}},
				Limit: new(uint32(1)),
			}
			first := MustSucceed(retrieve(authed, req))
			Expect(first.Slaves).To(HaveLen(1))
			req.After = &apislave.Cursor{Key: first.Slaves[0].Key}
			second := MustSucceed(retrieve(authed, req))
			Expect(second.Slaves).To(HaveLen(1))
			Expect(second.Slaves[0].Key).
				ToNot(Equal(first.Slaves[0].Key))
		})

		It("Should reject a search that resumes after a key", func(ctx SpecContext) {
			d := createSlave(ctx)
			grantRetrieve(ctx)
			Expect(retrieve(
				testutil.AuthedCtx(ctx, u),
				apislave.RetrieveRequest{
					SearchTerm: d.Name,
					After:      &apislave.Cursor{Key: d.Key},
				},
			)).Error().To(MatchError(validate.ErrValidation))
		})

		It("Should return a path error for a malformed where", func(ctx SpecContext) {
			grantRetrieve(ctx)
			Expect(retrieve(testutil.AuthedCtx(ctx, u), apislave.RetrieveRequest{
				Where: &slave.Where{Name: &filter.String{Regex: new("(a")}},
			})).Error().To(SatisfyAll(
				BeAssignableToTypeOf(validate.PathError{}),
				MatchError(ContainSubstring("where.name.regex: error parsing regexp")),
				MatchError(validate.ErrValidation),
			))
		})

		It("Should return no slaves for a zero limit", func(ctx SpecContext) {
			createSlave(ctx)
			grantRetrieve(ctx)
			res := MustSucceed(retrieve(
				testutil.AuthedCtx(ctx, u),
				apislave.RetrieveRequest{Limit: new(uint32(0))},
			))
			Expect(res.Slaves).To(BeEmpty())
		})

		It("Should return no slaves for a missing key", func(ctx SpecContext) {
			grantRetrieve(ctx)
			missing := uuid.New()
			res := MustSucceed(retrieve(
				testutil.AuthedCtx(ctx, u),
				apislave.RetrieveRequest{
					Where: &slave.Where{Key: &filter.Set[slave.Key]{Eq: &missing}},
				},
			))
			Expect(res.Slaves).To(BeEmpty())
		})
	})

	Describe("Update", func() {
		It("Should write the configuration and keep the identity", func(
			ctx SpecContext,
		) {
			s := writeSlave(ctx)
			grantWrites(ctx)
			Expect(update(testutil.AuthedCtx(ctx, u), rename(s.Key, "Inlet"))).
				Error().ToNot(HaveOccurred())
			got := retrieveSlave(ctx, s.Key)
			Expect(got.Name).To(Equal("Inlet"))
			Expect(got.Identifier).To(Equal("inlet"))
			Expect(got.Serial).To(Equal(s.Serial))
		})

		It("Should write the identity and keep the configuration", func(
			ctx SpecContext,
		) {
			s := writeSlave(ctx)
			grantWrites(ctx)
			Expect(update(testutil.AuthedCtx(ctx, u), apislave.UpdateRequest{
				Key: s.Key,
				Update: slave.Update{Variant: slave.IdentityUpdate{
					Identity: slave.Identity{
						Rack:        s.Rack,
						Serial:      9999,
						VendorID:    s.VendorID,
						ProductCode: s.ProductCode,
						Master:      s.Master,
					},
				}},
			})).Error().ToNot(HaveOccurred())
			got := retrieveSlave(ctx, s.Key)
			Expect(got.Serial).To(Equal(uint32(9999)))
			Expect(got.Name).To(Equal(s.Name))
		})

		It("Should reject the request when update is not granted", func(
			ctx SpecContext,
		) {
			s := writeSlave(ctx)
			Expect(update(testutil.AuthedCtx(ctx, u), rename(s.Key, "Inlet"))).
				Error().To(MatchError(access.ErrDenied))
			Expect(retrieveSlave(ctx, s.Key).Name).To(Equal(s.Name))
		})
	})

	Describe("Delete", func() {
		It("Should remove the slave when delete is granted", func(ctx SpecContext) {
			s := writeSlave(ctx)
			grantWrites(ctx)
			Expect(del(
				testutil.AuthedCtx(ctx, u),
				apislave.DeleteRequest{Keys: []slave.Key{s.Key}},
			)).Error().ToNot(HaveOccurred())
			Expect(svc.EtherCAT.Slaves.NewRetrieve().
				Where(slave.MatchKeys(s.Key)).
				Exists(ctx, db)).To(BeFalse())
		})

		It("Should reject the request when delete is not granted", func(
			ctx SpecContext,
		) {
			s := writeSlave(ctx)
			Expect(del(
				testutil.AuthedCtx(ctx, u),
				apislave.DeleteRequest{Keys: []slave.Key{s.Key}},
			)).Error().To(MatchError(access.ErrDenied))
			Expect(retrieveSlave(ctx, s.Key).Key).To(Equal(s.Key))
		})
	})
})
