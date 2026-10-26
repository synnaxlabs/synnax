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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	apistatus "github.com/synnaxlabs/synnax/pkg/api/status"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/filter"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// statusTypeOnly is the type-level ontology ID for granting access across all
// statuses. Row-level grants must match the input key exactly; type-level grants
// cover any input via the ontology.ID IsType matching rule.
var statusTypeOnly = ontology.ID{Type: ontology.ResourceTypeStatus}

var _ = Describe("Service.SetByKeyOrName", func() {
	Describe("authorized requests", func() {
		It(
			"Should create a UUID-keyed row named after the input when nothing matches",
			func(ctx SpecContext) {
				name := "api_set_fresh_" + uuid.New().String()
				grantOn(ctx, author.OntologyID(),
					[]access.Action{access.ActionCreate},
					statusTypeOnly)

				res := MustSucceed(
					apiSvc.SetByKeyOrName(
						AuthedCtx(ctx, author),
						db,
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
						Exec(ctx, nil),
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
				grantOn(ctx, author.OntologyID(),
					[]access.Action{access.ActionUpdate},
					status.OntologyID(key))

				res := MustSucceed(
					apiSvc.SetByKeyOrName(
						AuthedCtx(ctx, author),
						db,
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
				grantOn(ctx, author.OntologyID(),
					[]access.Action{access.ActionUpdate},
					statusTypeOnly)

				res := MustSucceed(
					apiSvc.SetByKeyOrName(
						AuthedCtx(ctx, author),
						db,
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
				grantOn(ctx, author.OntologyID(),
					[]access.Action{access.ActionCreate},
					statusTypeOnly)

				Expect(
					apiSvc.SetByKeyOrName(AuthedCtx(ctx, author), db, apistatus.SetByKeyOrNameRequest{
						KeyOrName: name,
						Message:   "x",
						Variant:   "bogus",
					}),
				).Error().
					To(SatisfyAll(MatchError(validate.ErrValidation), MatchError(ContainSubstring("invalid status variant"))))
			},
		)

		It(
			"Should propagate a validation error for empty input",
			func(ctx SpecContext) {
				grantOn(ctx, author.OntologyID(),
					[]access.Action{access.ActionCreate},
					statusTypeOnly)

				Expect(
					apiSvc.SetByKeyOrName(AuthedCtx(ctx, author), db, apistatus.SetByKeyOrNameRequest{
						KeyOrName: "",
						Message:   "x",
						Variant:   status.VariantInfo,
					}),
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
					apiSvc.SetByKeyOrName(
						AuthedCtx(ctx, anon),
						db,
						apistatus.SetByKeyOrNameRequest{
							KeyOrName: name,
							Message:   "noop",
							Variant:   status.VariantInfo,
						},
					),
				).Error().
					To(MatchError(access.ErrDenied))

				Expect(statusSvc.NewRetrieve[any]().Where(status.MatchKeys[any](name)).
					Entry(&status.Status[any]{}).
					Exec(ctx, nil)).To(MatchError(query.ErrNotFound))
			},
		)

		It(
			"Should deny when a row-level grant does not match the input key",
			func(ctx SpecContext) {
				name := "api_rowlevel_" + uuid.New().String()
				anon := freshUser(ctx)
				grantOn(ctx, anon.OntologyID(),
					[]access.Action{access.ActionCreate},
					status.OntologyID(uuid.New().String()))

				Expect(
					apiSvc.SetByKeyOrName(
						AuthedCtx(ctx, anon),
						db,
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
// tx) so access-control reads can observe the new ontology resource.
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
		grantOn(ctx, author.OntologyID(),
			[]access.Action{access.ActionRetrieve},
			statusTypeOnly)
	})

	It("Should omit a missing key instead of failing", func(ctx SpecContext) {
		s := createStatus(ctx, "api_partial_"+uuid.New().String())
		res := MustSucceed(apiSvc.Retrieve(
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
			res := MustSucceed(apiSvc.Retrieve(
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
			res := MustSucceed(apiSvc.Retrieve(
				AuthedCtx(ctx, author),
				apistatus.RetrieveRequest{SearchTerm: "cryostat"},
			))
			Expect(res.Statuses).To(HaveExactElements(HaveField("Key", s.Key)))
		},
	)

	It("Should return a path error under where for an invalid where", func(
		ctx SpecContext,
	) {
		Expect(apiSvc.Retrieve(AuthedCtx(ctx, author), apistatus.RetrieveRequest{
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
		Expect(statusSvc.NewWriter(nil).Set(ctx, &s)).To(Succeed())
		return s
	}
	retrieveStatus := func(ctx SpecContext, key string) status.Status[any] {
		GinkgoHelper()
		var s status.Status[any]
		Expect(statusSvc.NewRetrieve[any]().
			Where(status.MatchKeys[any](key)).
			Entry(&s).
			Exec(ctx, nil)).To(Succeed())
		return s
	}

	It("Should rename the status when update is granted", func(ctx SpecContext) {
		u := freshUser(ctx)
		s := createStatus(ctx)
		grantOn(ctx, u.OntologyID(),
			[]access.Action{access.ActionUpdate},
			status.OntologyID(s.Key))
		Expect(apiSvc.Rename(
			AuthedCtx(ctx, u), nil,
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
		grantOn(ctx, u.OntologyID(),
			[]access.Action{access.ActionUpdate},
			status.OntologyID(s.Key))
		Expect(apiSvc.Rename(
			AuthedCtx(ctx, u), nil,
			apistatus.RenameRequest{Key: s.Key, Name: ""},
		)).Error().To(MatchError(ContainSubstring("name: required")))
		Expect(retrieveStatus(ctx, s.Key).Name).To(Equal("Before"))
	})

	It("Should return a not found error for a missing status", func(ctx SpecContext) {
		u := freshUser(ctx)
		key := uuid.New().String()
		grantOn(ctx, u.OntologyID(),
			[]access.Action{access.ActionUpdate},
			status.OntologyID(key))
		Expect(apiSvc.Rename(
			AuthedCtx(ctx, u), nil,
			apistatus.RenameRequest{Key: key, Name: "After"},
		)).Error().To(MatchError(query.ErrNotFound))
	})

	It("Should reject the request when update is not granted", func(ctx SpecContext) {
		u := freshUser(ctx)
		s := createStatus(ctx)
		Expect(apiSvc.Rename(
			AuthedCtx(ctx, u), nil,
			apistatus.RenameRequest{Key: s.Key, Name: "After"},
		)).Error().To(MatchError(access.ErrDenied))
		Expect(retrieveStatus(ctx, s.Key).Name).To(Equal("Before"))
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
