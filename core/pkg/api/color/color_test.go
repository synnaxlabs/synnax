// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package color_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	apicolor "github.com/synnaxlabs/synnax/pkg/api/color"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	. "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/color"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	xcolor "github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/filter"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var red = xcolor.Color{R: 255, A: 255}

// deleteAfter deletes the named color with the given key after the spec.
func deleteAfter(key color.Key) {
	DeferCleanup(func(ctx SpecContext) {
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.Color.NewWriter(tx).Delete(ctx, key)
		})).To(Succeed())
	})
}

func createColor(ctx SpecContext, name string) color.Named {
	GinkgoHelper()
	n := color.Named{Name: name, Light: red}
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return svc.Color.NewWriter(tx).Create(ctx, &n)
	})).To(Succeed())
	deleteAfter(n.Key)
	return n
}

func retrieveColor(ctx SpecContext, key color.Key) color.Named {
	GinkgoHelper()
	var n color.Named
	Expect(svc.Color.NewRetrieve().
		Where(color.MatchKeys(key)).
		Entry(&n).
		Exec(ctx, db)).To(Succeed())
	return n
}

func colorExists(ctx SpecContext, key color.Key) bool {
	GinkgoHelper()
	return MustSucceed(
		svc.Color.NewRetrieve().Where(color.MatchKeys(key)).Exists(ctx, db),
	)
}

var _ = Describe("Create", func() {
	It(
		"Should create the named colors and stamp the dispatch key onto the broadcast",
		func(ctx SpecContext) {
			u := createUser(ctx)
			grant(ctx, access.ActionCreate, u.ResourceID(), resource.ID{
				Type: resource.TypeColor,
			})
			rec := &Recorder[color.Key, color.Action]{}
			DeferCleanup(svc.Color.Actions().OnChange(rec.Record))
			res := MustSucceed(create(AuthedCtx(ctx, u), apicolor.CreateRequest{
				Colors:      []color.Named{{Name: "Breaker", Light: red}},
				DispatchKey: "dk-create",
			}))
			Expect(res.Colors).To(HaveLen(1))
			deleteAfter(res.Colors[0].Key)
			Expect(retrieveColor(ctx, res.Colors[0].Key)).To(Equal(res.Colors[0]))
			frames := rec.Snapshot()
			Expect(frames).To(HaveLen(1))
			Expect(frames[0].DispatchKey).To(Equal("dk-create"))
		},
	)

	It("Should refuse a create without create on the color", func(ctx SpecContext) {
		u := createUser(ctx)
		n := color.Named{Key: uuid.New(), Name: "Breaker", Light: red}
		Expect(create(AuthedCtx(ctx, u), apicolor.CreateRequest{
			Colors: []color.Named{n},
		})).Error().To(denied(u, access.ActionCreate, n.ResourceID()))
		Expect(colorExists(ctx, n.Key)).To(BeFalse())
	})

	It("Should refuse a named color with no name", func(ctx SpecContext) {
		u := createUser(ctx)
		grant(ctx, access.ActionCreate, u.ResourceID(), resource.ID{
			Type: resource.TypeColor,
		})
		Expect(create(AuthedCtx(ctx, u), apicolor.CreateRequest{
			Colors: []color.Named{{Light: red}},
		})).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("name: required")),
		))
	})
})

var _ = Describe("Retrieve", func() {
	It("Should leave out the named colors that the subject cannot retrieve", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		granted := createColor(ctx, "Breaker")
		createColor(ctx, "Valve")
		grant(ctx, access.ActionRetrieve, u.ResourceID(), granted.ResourceID())
		res := MustSucceed(retrieve(AuthedCtx(ctx, u), apicolor.RetrieveRequest{}))
		Expect(res.Colors).To(Equal([]color.Named{granted}))
	})

	It("Should refuse a key that the subject cannot retrieve", func(ctx SpecContext) {
		u := createUser(ctx)
		n := createColor(ctx, "Breaker")
		Expect(retrieve(AuthedCtx(ctx, u), apicolor.RetrieveRequest{
			Where: &color.Where{Key: &filter.Set[color.Key]{Eq: &n.Key}},
		})).Error().To(denied(u, access.ActionRetrieve, n.ResourceID()))
	})

	It("Should search the named colors by name, closest first", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		closed := createColor(ctx, "Breaker closed")
		breaker := createColor(ctx, "Breaker")
		createColor(ctx, "Valve")
		grant(ctx, access.ActionRetrieve, u.ResourceID(), resource.ID{
			Type: resource.TypeColor,
		})
		res := MustSucceed(retrieve(AuthedCtx(ctx, u), apicolor.RetrieveRequest{
			SearchTerm: "breaker",
		}))
		Expect(res.Colors).To(Equal([]color.Named{breaker, closed}))
	})
})

var _ = Describe("Retrieve by name", func() {
	var (
		u     user.User
		amber color.Named
	)
	BeforeEach(func(ctx SpecContext) {
		u = createUser(ctx)
		grant(ctx, access.ActionRetrieve, u.ResourceID(), resource.ID{
			Type: resource.TypeColor,
		})
		createColor(ctx, "Valve")
		amber = createColor(ctx, "amber")
		createColor(ctx, "Breaker")
	})
	names := func(ctx SpecContext, req apicolor.RetrieveRequest) []string {
		GinkgoHelper()
		res := MustSucceed(retrieve(AuthedCtx(ctx, u), req))
		return lo.Map(res.Colors, func(n color.Named, _ int) string { return n.Name })
	}
	byName := &apicolor.Order{Field: apicolor.OrderFieldName}
	// loose drops the built-in named colors, which sit in a palette.
	loose := &color.Where{Grouped: new(false)}

	It("Should keep the named colors whose name the filter matches", func(
		ctx SpecContext,
	) {
		Expect(names(ctx, apicolor.RetrieveRequest{
			Where: &color.Where{Name: &filter.String{Eq: new("Valve")}},
		})).To(Equal([]string{"Valve"}))
	})

	It("Should sort the named colors by name, ignoring case", func(ctx SpecContext) {
		Expect(names(ctx, apicolor.RetrieveRequest{Where: loose, Order: byName})).
			To(Equal([]string{"amber", "Breaker", "Valve"}))
	})

	It("Should sort the named colors by name in descending order", func(
		ctx SpecContext,
	) {
		Expect(names(ctx, apicolor.RetrieveRequest{
			Where: loose,
			Order: &apicolor.Order{Field: apicolor.OrderFieldName, Descending: true},
		})).To(Equal([]string{"Valve", "Breaker", "amber"}))
	})

	It("Should return a page that starts past the cursor", func(ctx SpecContext) {
		Expect(names(ctx, apicolor.RetrieveRequest{
			Order: byName,
			After: &apicolor.Cursor{Key: amber.Key, Name: new("amber")},
			Limit: new(uint32(1)),
		})).To(Equal([]string{"Breaker"}))
	})

	It("Should return no named colors for a limit of zero", func(ctx SpecContext) {
		Expect(names(ctx, apicolor.RetrieveRequest{Limit: new(uint32(0))})).
			To(BeEmpty())
	})

	It("Should return a path error for an invalid regex", func(ctx SpecContext) {
		Expect(retrieve(AuthedCtx(ctx, u), apicolor.RetrieveRequest{
			Where: &color.Where{Name: &filter.String{Regex: new("(a")}},
		})).Error().To(SatisfyAll(
			BeAssignableToTypeOf(validate.PathError{}),
			MatchError(
				"where.name.regex: error parsing regexp: missing closing ): `(a`: "+
					"validation error",
			),
		))
	})

	It("Should refuse a cursor with no name in an order by name", func(
		ctx SpecContext,
	) {
		Expect(retrieve(AuthedCtx(ctx, u), apicolor.RetrieveRequest{
			Order: byName,
			After: &apicolor.Cursor{Key: amber.Key},
		})).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring(
				"after.name: is required to resume an order by name",
			)),
		))
	})
})

var _ = Describe("Dispatch", func() {
	rename := []color.Action{color.NewSetAction(color.SetPayload{
		Path:  []string{"name"},
		Value: "Valve",
	})}
	It("Should apply the actions when update is granted", func(ctx SpecContext) {
		u := createUser(ctx)
		n := createColor(ctx, "Breaker")
		grant(ctx, access.ActionUpdate, u.ResourceID(), n.ResourceID())
		MustSucceed(dispatch(
			AuthedCtx(ctx, u),
			apicolor.DispatchRequest{Key: n.Key, DispatchKey: "dk", Actions: rename},
		))
		Expect(retrieveColor(ctx, n.Key).Name).To(Equal("Valve"))
	})
	It("Should refuse the actions when update is not granted", func(ctx SpecContext) {
		u := createUser(ctx)
		n := createColor(ctx, "Breaker")
		Expect(dispatch(
			AuthedCtx(ctx, u),
			apicolor.DispatchRequest{Key: n.Key, DispatchKey: "dk", Actions: rename},
		)).Error().To(denied(u, access.ActionUpdate, n.ResourceID()))
		Expect(retrieveColor(ctx, n.Key).Name).To(Equal("Breaker"))
	})
	It("Should return the error of the dispatch", func(ctx SpecContext) {
		u := createUser(ctx)
		key := uuid.New()
		grant(ctx, access.ActionUpdate, u.ResourceID(), color.ResourceID(key))
		Expect(dispatch(
			AuthedCtx(ctx, u),
			apicolor.DispatchRequest{Key: key, DispatchKey: "dk", Actions: rename},
		)).Error().To(MatchError(query.ErrNotFound))
	})
})

var _ = Describe("Delete", func() {
	It("Should delete the named colors when delete is granted", func(ctx SpecContext) {
		u := createUser(ctx)
		n := createColor(ctx, "Breaker")
		grant(ctx, access.ActionDelete, u.ResourceID(), n.ResourceID())
		MustSucceed(del(AuthedCtx(ctx, u), apicolor.DeleteRequest{
			Keys: []color.Key{n.Key},
		}))
		Expect(colorExists(ctx, n.Key)).To(BeFalse())
	})
	It("Should refuse the delete without delete on the color", func(ctx SpecContext) {
		u := createUser(ctx)
		n := createColor(ctx, "Breaker")
		Expect(del(AuthedCtx(ctx, u), apicolor.DeleteRequest{
			Keys: []color.Key{n.Key},
		})).Error().To(denied(u, access.ActionDelete, n.ResourceID()))
		Expect(colorExists(ctx, n.Key)).To(BeTrue())
	})
})

var _ = Describe("RetrieveBuiltins", func() {
	It(
		"Should return the shipped values of the built-in colors",
		func(ctx SpecContext) {
			res := MustSucceed(apiSvc.RetrieveBuiltins(ctx, struct{}{}))
			Expect(res.Colors).To(Equal(color.Builtins()))
			Expect(res.Colors).To(HaveLen(16))
		},
	)

	It("Should return the shipped values after a built-in is edited", func(
		ctx SpecContext,
	) {
		shipped := color.Builtins()[0]
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.Color.NewWriter(tx).Apply(
				ctx,
				shipped.Key,
				color.NewSetAction(
					color.SetPayload{Path: []string{"name"}, Value: "Edited"},
				),
			)
		})).To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.Color.NewWriter(tx).Apply(
					ctx,
					shipped.Key,
					color.NewSetAction(
						color.SetPayload{Path: []string{"name"}, Value: shipped.Name},
					),
				)
			})).To(Succeed())
		})
		res := MustSucceed(apiSvc.RetrieveBuiltins(ctx, struct{}{}))
		Expect(res.Colors[0].Name).To(Equal(shipped.Name))
	})
})
