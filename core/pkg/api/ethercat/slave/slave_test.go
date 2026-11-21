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

var slaveType = resource.ID{Type: resource.TypeEthercatSlave}

var _ = Describe("Service", func() {
	var (
		u      user.User
		m      master.Master
		serial uint32
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
			return gorp.NewCreate[master.Key, master.Master]().Entry(&m).Exec(ctx, tx)
		})).To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return gorp.NewDelete[master.Key, master.Master]().
					Where(gorp.MatchKeys[master.Key, master.Master](m.Key)).
					Exec(ctx, tx)
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

		It("Should filter by serial and attach the status", func(ctx SpecContext) {
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
						Serial: &filter.Ordered[uint32]{Eq: &s.Serial},
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
			serial := createSlave(ctx).Serial
			res := MustSucceed(retrieve(
				testutil.AuthedCtx(ctx, u),
				apislave.RetrieveRequest{
					Where: &slave.Where{Serial: &filter.Ordered[uint32]{Eq: &serial}},
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
})
