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
	"github.com/synnaxlabs/gorp"
	apicolor "github.com/synnaxlabs/synnax/pkg/api/color"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	. "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/color"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
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
				Nameds:      []color.Named{{Name: "Breaker", Light: red}},
				DispatchKey: "dk-create",
			}))
			Expect(res.Nameds).To(HaveLen(1))
			deleteAfter(res.Nameds[0].Key)
			Expect(retrieveColor(ctx, res.Nameds[0].Key)).To(Equal(res.Nameds[0]))
			frames := rec.Snapshot()
			Expect(frames).To(HaveLen(1))
			Expect(frames[0].DispatchKey).To(Equal("dk-create"))
		},
	)

	It("Should refuse a create without create on the color", func(ctx SpecContext) {
		u := createUser(ctx)
		n := color.Named{Key: uuid.New(), Name: "Breaker", Light: red}
		Expect(create(AuthedCtx(ctx, u), apicolor.CreateRequest{
			Nameds: []color.Named{n},
		})).Error().To(denied(u, access.ActionCreate, n.ResourceID()))
		Expect(colorExists(ctx, n.Key)).To(BeFalse())
	})

	It("Should refuse a named color with no name", func(ctx SpecContext) {
		u := createUser(ctx)
		grant(ctx, access.ActionCreate, u.ResourceID(), resource.ID{
			Type: resource.TypeColor,
		})
		Expect(create(AuthedCtx(ctx, u), apicolor.CreateRequest{
			Nameds: []color.Named{{Light: red}},
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
		Expect(res.Nameds).To(Equal([]color.Named{granted}))
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
		Expect(res.Nameds).To(Equal([]color.Named{breaker, closed}))
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
