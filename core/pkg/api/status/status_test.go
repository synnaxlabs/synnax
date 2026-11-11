// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package status_test

import (
	"encoding/json/v2"
	"math/rand/v2"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"github.com/synnaxlabs/gorp"
	apistatus "github.com/synnaxlabs/synnax/pkg/api/status"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/filter"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// statusTypeOnly is the type-level resource ID for granting access across all
// statuses. Row-level grants must match the input key exactly; type-level grants
// cover any input via the resource.ID IsType matching rule.
var statusTypeOnly = resource.ID{Type: resource.TypeStatus}

// denied matches an error that denies u the action on object.
func denied(u user.User, action access.Action, object resource.ID) types.GomegaMatcher {
	return SatisfyAll(
		MatchError(access.ErrDenied),
		MatchError(ContainSubstring(
			"%s cannot %s %s",
			u.ResourceID(),
			action,
			object,
		)),
	)
}

var _ = Describe("Service.Set", func() {
	set := func(
		ctx SpecContext,
		statuses ...status.Status[any],
	) (apistatus.SetResponse, error) {
		GinkgoHelper()
		return setStatuses(
			AuthedCtx(ctx, author),
			apistatus.SetRequest{Statuses: statuses},
		)
	}
	newStatus := func(owner *resource.ID, details any) status.Status[any] {
		return status.Status[any]{
			Key:     uuid.New().String(),
			Owner:   owner,
			Time:    telem.Now(),
			Variant: status.VariantInfo,
			Details: details,
		}
	}
	BeforeEach(func(ctx SpecContext) {
		grantOn(ctx, author.ResourceID(),
			[]access.Action{access.ActionCreate, access.ActionUpdate},
			statusTypeOnly)
		grantOn(ctx, author.ResourceID(),
			[]access.Action{access.ActionUpdate},
			resource.ID{Type: resource.TypeRack},
			resource.ID{Type: resource.TypeDevice},
			resource.ID{Type: resource.TypeTask})
	})

	DescribeTable(
		"Should reject details the owner cannot decode",
		func(ctx SpecContext, owner resource.ID, details any) {
			s := newStatus(&owner, details)
			Expect(set(ctx, s)).Error().To(MatchError(
				And(
					ContainSubstring("statuses.0.details"),
					HaveSuffix("validation error"),
				),
			))
			Expect(statusSvc.NewRetrieve[any]().
				Where(status.MatchKeys[any](s.Key)).
				Entry(&status.Status[any]{}).
				Exec(ctx, db)).To(MatchError(query.ErrNotFound))
		},
		Entry("rack string", rackOwner, "not an object"),
		Entry("rack array", rackOwner, []any{"not", "an", "object"}),
		Entry("rack field type", rackOwner, map[string]any{"running": "yes"}),
		Entry("device string", deviceOwner, "not an object"),
		Entry("device field type", deviceOwner, map[string]any{"rack": "not a key"}),
		Entry("task number", taskOwner, 12),
		Entry("task field type", taskOwner, map[string]any{"running": "yes"}),
		Entry("task data array", taskOwner, map[string]any{"data": []any{1, 2}}),
		Entry("channel string", channelOwner, "not an object"),
	)

	DescribeTable(
		"Should accept details the owner can decode",
		func(ctx SpecContext, create func(SpecContext) resource.ID, details any) {
			owner := create(ctx)
			Expect(MustSucceed(set(ctx, newStatus(&owner, details))).Statuses).
				To(HaveLen(1))
		},
		Entry("rack", createRack, map[string]any{}),
		Entry("rack with no details", createRack, nil),
		Entry("rack running", createRack, map[string]any{"running": true}),
		Entry("rack with extra fields", createRack, map[string]any{"rack": 65537}),
		Entry("device", createDevice, map[string]any{
			"rack": 65537, "data": map[string]any{"any": "thing"},
		}),
		Entry("task", createTask, map[string]any{
			"running": true,
			"data":    map[string]any{"a": 1},
		}),
	)

	DescribeTable(
		"Should store the details of a rack status as the rack decodes them",
		func(ctx SpecContext, details any, expected rack.StatusDetails) {
			owner := createRack(ctx)
			s := newStatus(&owner, details)
			Expect(MustSucceed(set(ctx, s)).Statuses[0].Details).To(Equal(expected))
			var stored status.Status[rack.StatusDetails]
			Expect(statusSvc.NewRetrieve[rack.StatusDetails]().
				Where(status.MatchOwners[rack.StatusDetails](owner)).
				Entry(&stored).
				Exec(ctx, db)).To(Succeed())
			Expect(stored.Details).To(Equal(expected))
		},
		Entry("no details", nil, rack.StatusDetails{}),
		Entry("running", map[string]any{"running": true}, rack.StatusDetails{
			Running: true,
		}),
	)

	It(
		"Should accept free-form details on a status with no owner",
		func(ctx SpecContext) {
			Expect(MustSucceed(
				set(ctx, newStatus(nil, "free form")),
			).Statuses).To(HaveLen(1))
		},
	)

	It(
		"Should check details against the stored owner on an update with no owner",
		func(ctx SpecContext) {
			owner := createTask(ctx)
			valid := map[string]any{"running": true}
			stored := MustSucceed(set(ctx, newStatus(&owner, valid))).Statuses[0]
			update := newStatus(nil, "not an object")
			update.Key = stored.Key
			Expect(set(ctx, update)).Error().To(MatchError(
				And(
					ContainSubstring("statuses.0.details"),
					HaveSuffix("validation error"),
				),
			))
			var res status.Status[any]
			Expect(statusSvc.NewRetrieve[any]().
				Where(status.MatchKeys[any](stored.Key)).
				Entry(&res).
				Exec(ctx, db)).To(Succeed())
			Expect(res.Details).To(HaveKeyWithValue("running", true))
		},
	)

	It(
		"Should refuse a write to the status an owner holds without access on its key",
		func(ctx SpecContext) {
			owner := createTask(ctx)
			held := newStatus(&owner, nil)
			held.Message = "Running"
			held = MustSucceed(set(ctx, held)).Statuses[0]
			u := freshUser(ctx)
			req := newStatus(&owner, nil)
			req.Message = "Overwritten"
			grantOn(ctx, u.ResourceID(),
				[]access.Action{access.ActionCreate},
				status.ResourceID(req.Key))
			Expect(setStatuses(
				AuthedCtx(ctx, u),
				apistatus.SetRequest{Statuses: []status.Status[any]{req}},
			)).Error().To(denied(u, access.ActionUpdate, status.ResourceID(held.Key)))
			var res status.Status[any]
			Expect(statusSvc.NewRetrieve[any]().
				Where(status.MatchKeys[any](held.Key)).
				Entry(&res).
				Exec(ctx, db)).To(Succeed())
			Expect(res.Message).To(Equal("Running"))
		},
	)

	It(
		"Should check details against an owner from an earlier status in the batch",
		func(ctx SpecContext) {
			owner := createTask(ctx)
			owned := newStatus(&owner, map[string]any{"running": true})
			update := newStatus(nil, "not an object")
			update.Key = owned.Key
			Expect(set(ctx, owned, update)).Error().To(MatchError(
				And(
					ContainSubstring("statuses.1.details"),
					HaveSuffix("validation error"),
				),
			))
			Expect(statusSvc.NewRetrieve[any]().
				Where(status.MatchKeys[any](owned.Key)).
				Entry(&status.Status[any]{}).
				Exec(ctx, db)).To(MatchError(query.ErrNotFound))
		},
	)

	It(
		"Should write nothing when one status in a batch is invalid",
		func(ctx SpecContext) {
			valid := newStatus(nil, nil)
			invalid := newStatus(&rackOwner, "not an object")
			Expect(set(ctx, valid, invalid)).Error().To(MatchError(
				ContainSubstring("statuses.1.details"),
			))
			Expect(statusSvc.NewRetrieve[any]().
				Where(status.MatchKeys[any](valid.Key)).
				Entry(&status.Status[any]{}).
				Exec(ctx, db)).To(MatchError(query.ErrNotFound))
		},
	)
})

var _ = Describe("Service.Set access", func() {
	var u user.User
	BeforeEach(func(ctx SpecContext) { u = freshUser(ctx) })
	setAs := func(
		ctx SpecContext,
		s status.Status[any],
	) (apistatus.SetResponse, error) {
		return setStatuses(
			AuthedCtx(ctx, u),
			apistatus.SetRequest{Statuses: []status.Status[any]{s}},
		)
	}
	newStatus := func(owner *resource.ID, labels ...label.Key) status.Status[any] {
		return status.Status[any]{
			Key:     uuid.New().String(),
			Owner:   owner,
			Labels:  labels,
			Time:    telem.Now(),
			Variant: status.VariantInfo,
		}
	}
	retrieve := func(ctx SpecContext, key status.Key) status.Status[any] {
		GinkgoHelper()
		var s status.Status[any]
		Expect(statusSvc.NewRetrieve[any]().
			Where(status.MatchKeys[any](key)).
			Entry(&s).
			Exec(ctx, db)).To(Succeed())
		return s
	}
	exists := func(ctx SpecContext, key status.Key) bool {
		GinkgoHelper()
		var found []status.Status[any]
		Expect(statusSvc.NewRetrieve[any]().
			Where(status.MatchKeys[any](key)).
			SkipMissingKeys().
			Entries(&found).
			Exec(ctx, db)).To(Succeed())
		return len(found) > 0
	}
	createLabel := func(ctx SpecContext) label.Label {
		GinkgoHelper()
		l := label.Label{Name: "label"}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return labelSvc.NewWriter(tx).Create(ctx, &l)
		})).To(Succeed())
		return l
	}

	It("Should refuse a new status without create on it", func(ctx SpecContext) {
		s := newStatus(nil)
		Expect(setAs(ctx, s)).
			Error().To(denied(u, access.ActionCreate, status.ResourceID(s.Key)))
		Expect(exists(ctx, s.Key)).To(BeFalse())
	})

	It("Should create a new status with create on it", func(ctx SpecContext) {
		s := newStatus(nil)
		grantOn(ctx, u.ResourceID(),
			[]access.Action{access.ActionCreate},
			status.ResourceID(s.Key))
		MustSucceed(setAs(ctx, s))
		Expect(exists(ctx, s.Key)).To(BeTrue())
	})

	It(
		"Should refuse an update of a status without update on it",
		func(ctx SpecContext) {
			s := newStatus(nil)
			s.Message = "Stored"
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return statusSvc.NewWriter(tx).Set(ctx, &s)
			})).To(Succeed())
			grantOn(ctx, u.ResourceID(),
				[]access.Action{access.ActionCreate},
				status.ResourceID(s.Key))
			s.Message = "Updated"
			Expect(setAs(ctx, s)).
				Error().To(denied(u, access.ActionUpdate, status.ResourceID(s.Key)))
			Expect(retrieve(ctx, s.Key).Message).To(Equal("Stored"))
		},
	)

	It("Should update a status with update on it", func(ctx SpecContext) {
		s := newStatus(nil)
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return statusSvc.NewWriter(tx).Set(ctx, &s)
		})).To(Succeed())
		grantOn(ctx, u.ResourceID(),
			[]access.Action{access.ActionUpdate},
			status.ResourceID(s.Key))
		s.Message = "Updated"
		MustSucceed(setAs(ctx, s))
		Expect(retrieve(ctx, s.Key).Message).To(Equal("Updated"))
	})

	It(
		"Should refuse a new status without retrieve on its labels",
		func(ctx SpecContext) {
			l := createLabel(ctx)
			s := newStatus(nil, l.Key)
			grantOn(ctx, u.ResourceID(),
				[]access.Action{access.ActionCreate},
				status.ResourceID(s.Key))
			Expect(setAs(ctx, s)).
				Error().To(denied(u, access.ActionRetrieve, l.ResourceID()))
			Expect(exists(ctx, s.Key)).To(BeFalse())
		},
	)

	It(
		"Should create a new status with retrieve on its labels",
		func(ctx SpecContext) {
			l := createLabel(ctx)
			s := newStatus(nil, l.Key)
			grantOn(ctx, u.ResourceID(),
				[]access.Action{access.ActionCreate},
				status.ResourceID(s.Key))
			grantOn(ctx, u.ResourceID(),
				[]access.Action{access.ActionRetrieve},
				l.ResourceID())
			MustSucceed(setAs(ctx, s))
			Expect(retrieve(ctx, s.Key).Labels).To(Equal([]label.Key{l.Key}))
		},
	)

	It(
		"Should refuse a new owned status without update on its owner",
		func(ctx SpecContext) {
			owner := createTask(ctx)
			s := newStatus(&owner)
			grantOn(ctx, u.ResourceID(),
				[]access.Action{access.ActionCreate},
				status.ResourceID(s.Key))
			Expect(setAs(ctx, s)).Error().To(denied(u, access.ActionUpdate, owner))
			Expect(exists(ctx, s.Key)).To(BeFalse())
		},
	)

	It(
		"Should create a new owned status with update on its owner",
		func(ctx SpecContext) {
			owner := createTask(ctx)
			s := newStatus(&owner)
			grantOn(ctx, u.ResourceID(),
				[]access.Action{access.ActionCreate},
				status.ResourceID(s.Key))
			grantOn(ctx, u.ResourceID(), []access.Action{access.ActionUpdate}, owner)
			MustSucceed(setAs(ctx, s))
			Expect(retrieve(ctx, s.Key).Owner).To(Equal(&owner))
		},
	)
})

func createRack(ctx SpecContext) resource.ID {
	GinkgoHelper()
	r := rack.Rack{Key: rack.Key(65536 + rand.Uint32N(1<<16)), Name: "rack"}
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return rackSvc.NewWriter(tx).Create(ctx, &r)
	})).To(Succeed())
	return r.ResourceID()
}

func createDevice(ctx SpecContext) resource.ID {
	GinkgoHelper()
	d := device.Device{Key: uuid.New().String(), Name: "device"}
	Expect(devices.NewCreate().Entry(&d).Exec(ctx, db)).To(Succeed())
	return d.ResourceID()
}

func createTask(ctx SpecContext) resource.ID {
	GinkgoHelper()
	t := task.Task{Key: uuid.New(), Name: "task"}
	Expect(tasks.NewCreate().Entry(&t).Exec(ctx, db)).To(Succeed())
	return t.ResourceID()
}

var (
	rackOwner    = resource.ID{Type: resource.TypeRack, Key: "65537"}
	deviceOwner  = resource.ID{Type: resource.TypeDevice, Key: "dev"}
	taskOwner    = resource.ID{Type: resource.TypeTask, Key: uuid.New().String()}
	channelOwner = resource.ID{Type: resource.TypeChannel, Key: "12"}
)

var _ = Describe("Service.SetByKeyOrName", func() {
	Describe("authorized requests", func() {
		It(
			"Should create a UUID-keyed row named after the input when nothing matches",
			func(ctx SpecContext) {
				name := "api_set_fresh_" + uuid.New().String()
				grantOn(ctx, author.ResourceID(),
					[]access.Action{access.ActionCreate},
					statusTypeOnly)

				res := MustSucceed(
					setByKeyOrName(
						AuthedCtx(ctx, author),
						apistatus.SetByKeyOrNameRequest{
							KeyOrName: name,
							Message:   "hello",
							Variant:   status.VariantInfo,
						},
					),
				)
				MustSucceed(uuid.Parse(res.Key))
				Expect(res.Key).ToNot(Equal(name))
				Expect(res.MultipleMatches).To(BeFalse())

				var s status.Status[any]
				Expect(
					statusSvc.NewRetrieve[any]().
						Where(status.MatchKeys[any](res.Key)).
						Entry(&s).
						Exec(ctx, db),
				).To(Succeed())
				Expect(s.Key).To(Equal(res.Key))
				Expect(s.Name).To(Equal(name))
				Expect(s.Message).To(Equal("hello"))
			},
		)

		It(
			"Should update an existing row when the input matches its Key",
			func(ctx SpecContext) {
				key := uuid.New().String()
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return statusSvc.NewWriter(tx).Set(ctx, &status.Status[any]{
						Variant: status.VariantInfo, Message: "orig", Time: telem.Now(),
						Key: key, Name: "api_uuid",
					})
				})).To(Succeed())
				grantOn(ctx, author.ResourceID(),
					[]access.Action{access.ActionUpdate},
					status.ResourceID(key))

				res := MustSucceed(
					setByKeyOrName(
						AuthedCtx(ctx, author),
						apistatus.SetByKeyOrNameRequest{
							KeyOrName: key,
							Message:   "updated",
							Variant:   status.VariantWarning,
						},
					),
				)
				Expect(res.Key).To(Equal(key))
				Expect(res.MultipleMatches).To(BeFalse())
			},
		)

		It(
			"Should report multipleMatches when the name resolves to multiple rows",
			func(ctx SpecContext) {
				name := "api_multi_" + uuid.New().String()
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return statusSvc.NewWriter(tx).Set(ctx, &status.Status[any]{
						Variant: status.VariantInfo, Message: "a", Time: telem.Now(),
						Key: uuid.New().String(), Name: name,
					})
				})).To(Succeed())
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return statusSvc.NewWriter(tx).Set(ctx, &status.Status[any]{
						Variant: status.VariantInfo, Message: "b", Time: telem.Now(),
						Key: uuid.New().String(), Name: name,
					})
				})).To(Succeed())
				grantOn(ctx, author.ResourceID(),
					[]access.Action{access.ActionUpdate},
					statusTypeOnly)

				res := MustSucceed(
					setByKeyOrName(
						AuthedCtx(ctx, author),
						apistatus.SetByKeyOrNameRequest{
							KeyOrName: name,
							Message:   "updated",
							Variant:   status.VariantWarning,
						},
					),
				)
				Expect(res.MultipleMatches).To(BeTrue())
			},
		)
	})

	Describe("failure paths", func() {
		It(
			"Should propagate a validation error for an unknown variant",
			func(ctx SpecContext) {
				name := "api_iv_" + uuid.New().String()
				grantOn(ctx, author.ResourceID(),
					[]access.Action{access.ActionCreate},
					statusTypeOnly)

				Expect(
					setByKeyOrName(
						AuthedCtx(ctx, author),
						apistatus.SetByKeyOrNameRequest{
							KeyOrName: name,
							Message:   "x",
							Variant:   "bogus",
						},
					),
				).Error().
					To(SatisfyAll(MatchError(validate.ErrValidation), MatchError(ContainSubstring("invalid status variant"))))
			},
		)

		It(
			"Should propagate a validation error for empty input",
			func(ctx SpecContext) {
				grantOn(ctx, author.ResourceID(),
					[]access.Action{access.ActionCreate},
					statusTypeOnly)

				Expect(
					setByKeyOrName(
						AuthedCtx(ctx, author),
						apistatus.SetByKeyOrNameRequest{
							KeyOrName: "",
							Message:   "x",
							Variant:   status.VariantInfo,
						},
					),
				).Error().
					To(SatisfyAll(MatchError(validate.ErrValidation), MatchError(ContainSubstring("key_or_name is required"))))
			},
		)

		It(
			"Should refuse unauthorized requests without touching the store",
			func(ctx SpecContext) {
				name := "api_unauth_" + uuid.New().String()
				anon := freshUser(ctx)

				Expect(
					setByKeyOrName(
						AuthedCtx(ctx, anon),
						apistatus.SetByKeyOrNameRequest{
							KeyOrName: name,
							Message:   "noop",
							Variant:   status.VariantInfo,
						},
					),
				).Error().To(SatisfyAll(
					MatchError(access.ErrDenied),
					MatchError(ContainSubstring(
						"%s cannot create status:",
						anon.ResourceID(),
					)),
				))

				Expect(statusSvc.NewRetrieve[any]().Where(status.MatchKeys[any](name)).
					Entry(&status.Status[any]{}).
					Exec(ctx, db)).To(MatchError(query.ErrNotFound))
			},
		)

		It(
			"Should deny when a row-level grant does not match the input key",
			func(ctx SpecContext) {
				name := "api_rowlevel_" + uuid.New().String()
				anon := freshUser(ctx)
				grantOn(ctx, anon.ResourceID(),
					[]access.Action{access.ActionCreate},
					status.ResourceID(uuid.New().String()))

				Expect(
					setByKeyOrName(
						AuthedCtx(ctx, anon),
						apistatus.SetByKeyOrNameRequest{
							KeyOrName: name,
							Message:   "x",
							Variant:   status.VariantInfo,
						},
					),
				).Error().
					To(MatchError(access.ErrDenied))
			},
		)
	})
})

// createStatus persists a fresh status and returns it. Writes commit immediately (nil
// tx) so access-control reads can observe the new status.
func createStatus(ctx SpecContext, name string) status.Status[any] {
	GinkgoHelper()
	s := status.Status[any]{
		Key:     uuid.New().String(),
		Name:    name,
		Message: "test",
		Variant: status.VariantInfo,
		Time:    telem.Now(),
	}
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return statusSvc.NewWriter(tx).Set(ctx, &s)
	})).To(Succeed())
	return s
}

var _ = Describe("Service.Retrieve", func() {
	BeforeEach(func(ctx SpecContext) {
		grantOn(ctx, author.ResourceID(),
			[]access.Action{access.ActionRetrieve},
			statusTypeOnly)
	})

	It("Should omit a missing key instead of failing", func(ctx SpecContext) {
		s := createStatus(ctx, "api_partial_"+uuid.New().String())
		res := MustSucceed(retrieve(
			AuthedCtx(ctx, author),
			apistatus.RetrieveRequest{Where: &status.Where{
				Key: &filter.Set[status.Key]{
					In: []status.Key{s.Key, uuid.New().String()},
				},
			}},
		))
		Expect(res.Statuses).To(HaveExactElements(HaveField("Key", s.Key)))
	})

	It(
		"Should return an empty result when every requested key is missing",
		func(ctx SpecContext) {
			res := MustSucceed(retrieve(
				AuthedCtx(ctx, author),
				apistatus.RetrieveRequest{Where: &status.Where{
					Key: &filter.Set[status.Key]{Eq: new(uuid.New().String())},
				}},
			))
			Expect(res.Statuses).To(BeEmpty())
		},
	)

	It(
		"Should return the statuses whose name matches the search term",
		func(ctx SpecContext) {
			s := createStatus(ctx, "cryostat pressure")
			res := MustSucceed(retrieve(
				AuthedCtx(ctx, author),
				apistatus.RetrieveRequest{SearchTerm: "cryostat"},
			))
			Expect(res.Statuses).To(HaveExactElements(HaveField("Key", s.Key)))
		},
	)

	It("Should return a path error under where for an invalid where", func(
		ctx SpecContext,
	) {
		Expect(retrieve(AuthedCtx(ctx, author), apistatus.RetrieveRequest{
			Where: &status.Where{
				Variant: &filter.Set[status.Variant]{Eq: new(status.Variant("bogus"))},
			},
		})).Error().To(SatisfyAll(
			BeAssignableToTypeOf(validate.PathError{}),
			MatchError(`where.variant.eq: invalid variant "bogus": validation error`),
		))
	})
})

var _ = Describe("Service.Rename", func() {
	createStatus := func(ctx SpecContext) status.Status[any] {
		GinkgoHelper()
		s := status.Status[any]{
			Key:     uuid.New().String(),
			Name:    "Before",
			Variant: status.VariantInfo,
			Message: "hello",
			Time:    telem.Now(),
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return statusSvc.NewWriter(tx).Set(ctx, &s)
		})).To(Succeed())
		return s
	}
	retrieveStatus := func(ctx SpecContext, key string) status.Status[any] {
		GinkgoHelper()
		var s status.Status[any]
		Expect(statusSvc.NewRetrieve[any]().
			Where(status.MatchKeys[any](key)).
			Entry(&s).
			Exec(ctx, db)).To(Succeed())
		return s
	}

	It("Should rename the status when update is granted", func(ctx SpecContext) {
		u := freshUser(ctx)
		s := createStatus(ctx)
		grantOn(ctx, u.ResourceID(),
			[]access.Action{access.ActionUpdate},
			status.ResourceID(s.Key))
		Expect(rename(
			AuthedCtx(ctx, u),
			apistatus.RenameRequest{Key: s.Key, Name: "After"},
		)).Error().ToNot(HaveOccurred())
		renamed := retrieveStatus(ctx, s.Key)
		Expect(renamed.Name).To(Equal("After"))
		Expect(renamed.Message).To(Equal("hello"))
		Expect(renamed.Variant).To(Equal(status.VariantInfo))
	})

	It("Should reject an empty name", func(ctx SpecContext) {
		u := freshUser(ctx)
		s := createStatus(ctx)
		grantOn(ctx, u.ResourceID(),
			[]access.Action{access.ActionUpdate},
			status.ResourceID(s.Key))
		Expect(rename(
			AuthedCtx(ctx, u),
			apistatus.RenameRequest{Key: s.Key, Name: ""},
		)).Error().To(MatchError(ContainSubstring("name: required")))
		Expect(retrieveStatus(ctx, s.Key).Name).To(Equal("Before"))
	})

	It("Should return a not found error for a missing status", func(ctx SpecContext) {
		u := freshUser(ctx)
		key := uuid.New().String()
		grantOn(ctx, u.ResourceID(),
			[]access.Action{access.ActionUpdate},
			status.ResourceID(key))
		Expect(rename(
			AuthedCtx(ctx, u),
			apistatus.RenameRequest{Key: key, Name: "After"},
		)).Error().To(MatchError(query.ErrNotFound))
	})

	It("Should reject the request when update is not granted", func(ctx SpecContext) {
		u := freshUser(ctx)
		s := createStatus(ctx)
		Expect(rename(
			AuthedCtx(ctx, u),
			apistatus.RenameRequest{Key: s.Key, Name: "After"},
		)).Error().To(denied(u, access.ActionUpdate, s.ResourceID()))
		Expect(retrieveStatus(ctx, s.Key).Name).To(Equal("Before"))
	})
})

var _ = Describe("Service.Delete", func() {
	lookup := func(ctx SpecContext, key status.Key) error {
		return statusSvc.NewRetrieve[any]().
			Where(status.MatchKeys[any](key)).
			Entry(&status.Status[any]{}).
			Exec(ctx, db)
	}

	It("Should reject the request when delete is not granted", func(ctx SpecContext) {
		u := freshUser(ctx)
		s := createStatus(ctx, "delete-denied")
		Expect(del(AuthedCtx(ctx, u), apistatus.DeleteRequest{
			Keys: []status.Key{s.Key},
		})).Error().To(denied(u, access.ActionDelete, s.ResourceID()))
		Expect(lookup(ctx, s.Key)).To(Succeed())
	})

	It("Should delete the status when delete is granted", func(ctx SpecContext) {
		u := freshUser(ctx)
		s := createStatus(ctx, "delete-granted")
		grantOn(
			ctx,
			u.ResourceID(),
			[]access.Action{access.ActionDelete},
			s.ResourceID(),
		)
		MustSucceed(del(AuthedCtx(ctx, u), apistatus.DeleteRequest{
			Keys: []status.Key{s.Key},
		}))
		Expect(lookup(ctx, s.Key)).To(MatchError(query.ErrNotFound))
	})

	Describe("Owned status", func() {
		// createOwned creates a status that a new task holds.
		createOwned := func(ctx SpecContext) (status.Status[any], resource.ID) {
			GinkgoHelper()
			owner := createTask(ctx)
			s := status.Status[any]{
				Owner:   &owner,
				Variant: status.VariantInfo,
				Time:    telem.Now(),
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return statusSvc.NewWriter(tx).Set(ctx, &s)
			})).To(Succeed())
			return s, owner
		}

		It("Should require update on the owner", func(ctx SpecContext) {
			u := freshUser(ctx)
			s, owner := createOwned(ctx)
			grantOn(
				ctx,
				u.ResourceID(),
				[]access.Action{access.ActionDelete},
				s.ResourceID(),
			)
			Expect(del(AuthedCtx(ctx, u), apistatus.DeleteRequest{
				Keys: []status.Key{s.Key},
			})).Error().To(denied(u, access.ActionUpdate, owner))
			Expect(lookup(ctx, s.Key)).To(Succeed())
		})

		It("Should delete the status with delete on it and update on the owner", func(
			ctx SpecContext,
		) {
			u := freshUser(ctx)
			s, owner := createOwned(ctx)
			grantOn(
				ctx,
				u.ResourceID(),
				[]access.Action{access.ActionDelete},
				s.ResourceID(),
			)
			grantOn(ctx, u.ResourceID(), []access.Action{access.ActionUpdate}, owner)
			MustSucceed(del(AuthedCtx(ctx, u), apistatus.DeleteRequest{
				Keys: []status.Key{s.Key},
			}))
			Expect(lookup(ctx, s.Key)).To(MatchError(query.ErrNotFound))
		})
	})
})

var _ = Describe("RetrieveRequest", func() {
	DescribeTable(
		"Should reject a member that the request does not declare",
		func(body string) {
			var req apistatus.RetrieveRequest
			Expect(json.Unmarshal([]byte(body), &req)).To(SatisfyAll(
				MatchError(json.ErrUnknownName),
				MatchError(ContainSubstring(`unknown object member name "keys"`)),
			))
		},
		Entry("at the top", `{"keys":["status-a"]}`),
		Entry("in a nested where", `{"where":{"or":[{"keys":[]}]}}`),
	)
})

var _ = Describe("Retrieve access", func() {
	It("Should leave out the statuses that the subject cannot retrieve", func(
		ctx SpecContext,
	) {
		u := freshUser(ctx)
		granted := createStatus(ctx, "leave-out-granted")
		createStatus(ctx, "leave-out-denied")
		grantOn(
			ctx,
			u.ResourceID(),
			[]access.Action{access.ActionRetrieve},
			granted.ResourceID(),
		)
		res := MustSucceed(retrieve(AuthedCtx(ctx, u), apistatus.RetrieveRequest{}))
		Expect(res.Statuses).To(HaveExactElements(HaveField("Key", granted.Key)))
	})
})
