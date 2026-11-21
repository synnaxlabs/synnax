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
	"strconv"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	apimaster "github.com/synnaxlabs/synnax/pkg/api/ethercat/master"
	"github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/ethercat/master"
	"github.com/synnaxlabs/synnax/pkg/service/ethercat/slave"
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
	masterType = resource.ID{Type: resource.TypeEthercatMaster}
	statusType = resource.ID{Type: resource.TypeStatus}
	allWrites  = []access.Action{
		access.ActionCreate,
		access.ActionUpdate,
		access.ActionDelete,
	}
)

var _ = Describe("Service", func() {
	var (
		u user.User
		n int
	)
	BeforeEach(func(ctx SpecContext) { u = createUser(ctx) })

	newMaster := func(overrides ...func(*master.Master)) master.Master {
		n++
		m := master.Master{
			Key:       uuid.New(),
			Rack:      svc.Rack.EmbeddedKey,
			Name:      "EtherCAT bus",
			Interface: "eth" + strconv.Itoa(n),
		}
		for _, override := range overrides {
			override(&m)
		}
		return m
	}
	// grantWrites lets u write any master, its status, and the rack it sits on.
	grantWrites := func(ctx SpecContext, r rack.Key) {
		GinkgoHelper()
		grant(ctx, u, allWrites, masterType, statusType)
		grant(ctx, u, []access.Action{access.ActionUpdate}, r.ResourceID())
	}
	retrieveMaster := func(ctx SpecContext, key master.Key) master.Master {
		GinkgoHelper()
		var m master.Master
		Expect(svc.EtherCAT.Masters.NewRetrieve().
			Where(master.MatchKeys(key)).
			Entry(&m).
			Exec(ctx, db)).To(Succeed())
		return m
	}
	deleteMaster := func(ctx SpecContext, key master.Key) {
		GinkgoHelper()
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.EtherCAT.Masters.NewWriter(tx).Delete(ctx, key)
		})).To(Succeed())
	}
	grantRetrieve := func(ctx SpecContext) {
		GinkgoHelper()
		grant(ctx, u, []access.Action{access.ActionRetrieve}, masterType)
	}
	createMaster := func(ctx SpecContext, overrides ...func(*master.Master)) master.Master {
		GinkgoHelper()
		m := newMaster(overrides...)
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
		return m
	}

	Describe("NewService", func() {
		DescribeTable("Should reject a config missing a required field",
			func(field string, unset func(*apimaster.ServiceConfig)) {
				cfg := apimaster.ServiceConfig{
					Master: svc.EtherCAT.Masters,
					Status: svc.Status,
				}
				unset(&cfg)
				Expect(apimaster.NewService(cfg)).Error().
					To(MatchError(ContainSubstring(field + ": must be non-nil")))
			},
			Entry("master", "master", func(c *apimaster.ServiceConfig) {
				c.Master = nil
			}),
			Entry("status", "status", func(c *apimaster.ServiceConfig) {
				c.Status = nil
			}),
		)

		It("Should let a later config override an earlier one", func() {
			base := apimaster.ServiceConfig{Master: svc.EtherCAT.Masters}
			Expect(apimaster.NewService(base, apimaster.ServiceConfig{
				Status: svc.Status,
			})).ToNot(BeNil())
		})
	})

	Describe("Create", func() {
		It("Should mint a key when writes are granted", func(ctx SpecContext) {
			grantWrites(ctx, svc.Rack.EmbeddedKey)
			m := newMaster()
			m.Key = uuid.Nil()
			res := MustSucceed(create(
				testutil.AuthedCtx(ctx, u),
				apimaster.CreateRequest{Masters: []master.Master{m}},
			))
			Expect(res.Masters[0].Key).ToNot(Equal(uuid.Nil()))
			deleteMaster(ctx, res.Masters[0].Key)
		})

		It("Should not echo a status the client sent", func(ctx SpecContext) {
			grantWrites(ctx, svc.Rack.EmbeddedKey)
			m := newMaster(func(m *master.Master) {
				m.Status = &master.Status{Message: "Connected"}
			})
			res := MustSucceed(create(
				testutil.AuthedCtx(ctx, u),
				apimaster.CreateRequest{Masters: []master.Master{m}},
			))
			Expect(res.Masters[0].Status).To(BeNil())
			deleteMaster(ctx, m.Key)
		})

		It("Should reject the request when create is not granted", func(
			ctx SpecContext,
		) {
			Expect(create(
				testutil.AuthedCtx(ctx, u),
				apimaster.CreateRequest{Masters: []master.Master{newMaster()}},
			)).Error().To(MatchError(access.ErrDenied))
		})

		It("Should not create a master without update access on its rack", func(
			ctx SpecContext,
		) {
			grant(ctx, u, allWrites, masterType, statusType)
			Expect(create(
				testutil.AuthedCtx(ctx, u),
				apimaster.CreateRequest{Masters: []master.Master{newMaster()}},
			)).Error().To(MatchError(access.ErrDenied))
		})
	})

	Describe("Retrieve", func() {
		It("Should reject a named master when retrieve is not granted", func(
			ctx SpecContext,
		) {
			key := createMaster(ctx).Key
			Expect(retrieve(
				testutil.AuthedCtx(ctx, u),
				apimaster.RetrieveRequest{
					Where: &master.Where{Key: &filter.Set[master.Key]{Eq: &key}},
				},
			)).Error().To(MatchError(access.ErrDenied))
		})

		It("Should filter by interface and attach the status", func(ctx SpecContext) {
			m := createMaster(ctx)
			createMaster(ctx)
			st := master.Status{
				Owner:   new(m.ResourceID()),
				Name:    m.Name,
				Time:    telem.Now(),
				Variant: status.VariantSuccess,
				Message: "Running",
				Details: master.StatusDetails{Master: m.Key},
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.Status.NewWriter(tx).Set(ctx, &st)
			})).To(Succeed())
			grantRetrieve(ctx)
			res := MustSucceed(retrieve(
				testutil.AuthedCtx(ctx, u),
				apimaster.RetrieveRequest{
					Where: &master.Where{
						Interface: &filter.String{Eq: &m.Interface},
					},
					IncludeStatus: true,
				},
			))
			Expect(res.Masters).To(HaveLen(1))
			Expect(res.Masters[0].Key).To(Equal(m.Key))
			Expect(res.Masters[0].Status.Message).To(Equal("Running"))
		})

		It("Should leave out the masters the subject cannot retrieve", func(
			ctx SpecContext,
		) {
			iface := createMaster(ctx).Interface
			res := MustSucceed(retrieve(
				testutil.AuthedCtx(ctx, u),
				apimaster.RetrieveRequest{
					Where: &master.Where{Interface: &filter.String{Eq: &iface}},
				},
			))
			Expect(res.Masters).To(BeEmpty())
		})

		It("Should find a master by its name", func(ctx SpecContext) {
			m := createMaster(ctx, func(m *master.Master) { m.Name = "Pump bus" })
			grantRetrieve(ctx)
			res := MustSucceed(retrieve(
				testutil.AuthedCtx(ctx, u),
				apimaster.RetrieveRequest{SearchTerm: "pump bus"},
			))
			Expect(lo.Map(res.Masters, func(m master.Master, _ int) master.Key {
				return m.Key
			})).To(ContainElement(m.Key))
		})

		It("Should cap the results at the limit", func(ctx SpecContext) {
			a := createMaster(ctx)
			b := createMaster(ctx)
			grantRetrieve(ctx)
			res := MustSucceed(retrieve(
				testutil.AuthedCtx(ctx, u),
				apimaster.RetrieveRequest{
					Where: &master.Where{Key: &filter.Set[master.Key]{
						In: []master.Key{a.Key, b.Key},
					}},
					Limit: new(uint32(1)),
				},
			))
			Expect(res.Masters).To(HaveLen(1))
		})

		It("Should page past the after key", func(ctx SpecContext) {
			a := createMaster(ctx)
			b := createMaster(ctx)
			grantRetrieve(ctx)
			authed := testutil.AuthedCtx(ctx, u)
			req := apimaster.RetrieveRequest{
				Where: &master.Where{Key: &filter.Set[master.Key]{
					In: []master.Key{a.Key, b.Key},
				}},
				Limit: new(uint32(1)),
			}
			first := MustSucceed(retrieve(authed, req))
			Expect(first.Masters).To(HaveLen(1))
			req.After = &apimaster.Cursor{Key: first.Masters[0].Key}
			second := MustSucceed(retrieve(authed, req))
			Expect(second.Masters).To(HaveLen(1))
			Expect(second.Masters[0].Key).
				ToNot(Equal(first.Masters[0].Key))
		})

		It("Should reject a search that resumes after a key", func(ctx SpecContext) {
			d := createMaster(ctx)
			grantRetrieve(ctx)
			Expect(retrieve(
				testutil.AuthedCtx(ctx, u),
				apimaster.RetrieveRequest{
					SearchTerm: d.Name,
					After:      &apimaster.Cursor{Key: d.Key},
				},
			)).Error().To(MatchError(validate.ErrValidation))
		})

		It("Should return a path error for a malformed where", func(ctx SpecContext) {
			grantRetrieve(ctx)
			Expect(retrieve(testutil.AuthedCtx(ctx, u), apimaster.RetrieveRequest{
				Where: &master.Where{Name: &filter.String{Regex: new("(a")}},
			})).Error().To(SatisfyAll(
				BeAssignableToTypeOf(validate.PathError{}),
				MatchError(ContainSubstring("where.name.regex: error parsing regexp")),
				MatchError(validate.ErrValidation),
			))
		})

		It("Should return no masters for a zero limit", func(ctx SpecContext) {
			createMaster(ctx)
			grantRetrieve(ctx)
			res := MustSucceed(retrieve(
				testutil.AuthedCtx(ctx, u),
				apimaster.RetrieveRequest{Limit: new(uint32(0))},
			))
			Expect(res.Masters).To(BeEmpty())
		})

		It("Should return no masters for a missing key", func(ctx SpecContext) {
			grantRetrieve(ctx)
			missing := uuid.New()
			res := MustSucceed(retrieve(
				testutil.AuthedCtx(ctx, u),
				apimaster.RetrieveRequest{
					Where: &master.Where{Key: &filter.Set[master.Key]{Eq: &missing}},
				},
			))
			Expect(res.Masters).To(BeEmpty())
		})
	})

	Describe("Update", func() {
		It("Should write the configuration and keep the identity", func(
			ctx SpecContext,
		) {
			m := newMaster()
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.EtherCAT.Masters.NewWriter(tx).Create(ctx, &m)
			})).To(Succeed())
			DeferCleanup(func(ctx SpecContext) { deleteMaster(ctx, m.Key) })
			grantWrites(ctx, m.Rack)
			Expect(update(testutil.AuthedCtx(ctx, u), apimaster.UpdateRequest{
				Key: m.Key,
				Update: master.Update{Variant: master.ConfigurationUpdate{
					Configuration: master.Configuration{Name: "Renamed"},
				}},
			})).Error().ToNot(HaveOccurred())
			got := retrieveMaster(ctx, m.Key)
			Expect(got.Name).To(Equal("Renamed"))
			Expect(got.Interface).To(Equal(m.Interface))
		})

		It("Should write the identity and keep the configuration", func(
			ctx SpecContext,
		) {
			m := newMaster()
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.EtherCAT.Masters.NewWriter(tx).Create(ctx, &m)
			})).To(Succeed())
			DeferCleanup(func(ctx SpecContext) { deleteMaster(ctx, m.Key) })
			grantWrites(ctx, m.Rack)
			order := []uuid.UUID{uuid.New(), uuid.New()}
			Expect(update(testutil.AuthedCtx(ctx, u), apimaster.UpdateRequest{
				Key: m.Key,
				Update: master.Update{Variant: master.IdentityUpdate{
					Identity: master.Identity{
						Rack:      m.Rack,
						Interface: "igh:0",
						Slaves:    order,
					},
				}},
			})).Error().ToNot(HaveOccurred())
			got := retrieveMaster(ctx, m.Key)
			Expect(got.Interface).To(Equal("igh:0"))
			Expect(got.Slaves).To(Equal(order))
			Expect(got.Name).To(Equal(m.Name))
		})

		It("Should not move a master to a rack the subject cannot update", func(
			ctx SpecContext,
		) {
			m := newMaster()
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.EtherCAT.Masters.NewWriter(tx).Create(ctx, &m)
			})).To(Succeed())
			DeferCleanup(func(ctx SpecContext) { deleteMaster(ctx, m.Key) })
			other := rack.Rack{Name: "Other"}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.Rack.NewWriter(tx).Create(ctx, &other)
			})).To(Succeed())
			grantWrites(ctx, m.Rack)
			Expect(update(testutil.AuthedCtx(ctx, u), apimaster.UpdateRequest{
				Key: m.Key,
				Update: master.Update{Variant: master.IdentityUpdate{
					Identity: master.Identity{Rack: other.Key, Interface: m.Interface},
				}},
			})).Error().To(MatchError(access.ErrDenied))
			Expect(retrieveMaster(ctx, m.Key).Rack).To(Equal(m.Rack))
			grant(ctx, u, []access.Action{access.ActionUpdate}, other.Key.ResourceID())
			Expect(update(testutil.AuthedCtx(ctx, u), apimaster.UpdateRequest{
				Key: m.Key,
				Update: master.Update{Variant: master.IdentityUpdate{
					Identity: master.Identity{Rack: other.Key, Interface: m.Interface},
				}},
			})).Error().ToNot(HaveOccurred())
			Expect(retrieveMaster(ctx, m.Key).Rack).To(Equal(other.Key))
		})

		It("Should reject an update with no field group", func(ctx SpecContext) {
			m := createMaster(ctx)
			grantWrites(ctx, m.Rack)
			Expect(update(
				testutil.AuthedCtx(ctx, u),
				apimaster.UpdateRequest{Key: m.Key},
			)).Error().To(MatchError(validate.ErrValidation))
		})

		It("Should reject the request when update is not granted", func(
			ctx SpecContext,
		) {
			m := createMaster(ctx)
			Expect(update(testutil.AuthedCtx(ctx, u), apimaster.UpdateRequest{
				Key: m.Key,
				Update: master.Update{Variant: master.ConfigurationUpdate{
					Configuration: master.Configuration{Name: "Renamed"},
				}},
			})).Error().To(MatchError(access.ErrDenied))
			Expect(retrieveMaster(ctx, m.Key).Name).To(Equal(m.Name))
		})
	})

	Describe("Delete", func() {
		It("Should remove the master when delete is granted", func(ctx SpecContext) {
			m := newMaster()
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.EtherCAT.Masters.NewWriter(tx).Create(ctx, &m)
			})).To(Succeed())
			grantWrites(ctx, m.Rack)
			Expect(del(
				testutil.AuthedCtx(ctx, u),
				apimaster.DeleteRequest{Keys: []master.Key{m.Key}},
			)).Error().ToNot(HaveOccurred())
			Expect(svc.EtherCAT.Masters.NewRetrieve().
				Where(master.MatchKeys(m.Key)).
				Exists(ctx, db)).To(BeFalse())
		})

		It("Should need delete access on the slaves it takes", func(ctx SpecContext) {
			m := newMaster()
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.EtherCAT.Masters.NewWriter(tx).Create(ctx, &m)
			})).To(Succeed())
			s := slave.Slave{
				Key:         uuid.New(),
				Rack:        m.Rack,
				Name:        "EL3102",
				VendorID:    2,
				ProductCode: 0x0C1E3052,
				Master:      m.Key,
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return gorp.NewCreate[slave.Key, slave.Slave]().Entry(&s).Exec(ctx, tx)
			})).To(Succeed())
			DeferCleanup(func(ctx SpecContext) { deleteMaster(ctx, m.Key) })
			grantWrites(ctx, m.Rack)
			req := apimaster.DeleteRequest{Keys: []master.Key{m.Key}}
			Expect(del(testutil.AuthedCtx(ctx, u), req)).
				Error().To(MatchError(access.ErrDenied))
			Expect(retrieveMaster(ctx, m.Key).Key).To(Equal(m.Key))
			grant(
				ctx,
				u,
				[]access.Action{access.ActionDelete},
				resource.ID{Type: resource.TypeEthercatSlave},
			)
			Expect(del(testutil.AuthedCtx(ctx, u), req)).Error().ToNot(HaveOccurred())
			Expect(svc.EtherCAT.Slaves.NewRetrieve().
				Where(slave.MatchKeys(s.Key)).
				Exists(ctx, db)).To(BeFalse())
		})

		It("Should reject the request when delete is not granted", func(
			ctx SpecContext,
		) {
			m := createMaster(ctx)
			Expect(del(
				testutil.AuthedCtx(ctx, u),
				apimaster.DeleteRequest{Keys: []master.Key{m.Key}},
			)).Error().To(MatchError(access.ErrDenied))
			Expect(retrieveMaster(ctx, m.Key).Key).To(Equal(m.Key))
		})
	})
})
