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
	"github.com/synnaxlabs/synnax/pkg/service/color"
	xcolor "github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Writer", func() {
	var (
		db  *gorp.DB
		svc *color.Service
		w   color.Writer
		red xcolor.Color
	)
	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New()))
		svc = MustOpen(color.OpenService(ctx, color.ServiceConfig{DB: db}))
		w = svc.NewWriter(db)
		red = MustSucceed(xcolor.Parse("#ff0000"))
	})
	retrieve := func(ctx SpecContext, key color.Key) color.Named {
		GinkgoHelper()
		var n color.Named
		Expect(svc.NewRetrieve().
			Where(color.MatchKeys(key)).
			Entry(&n).
			Exec(ctx, db)).To(Succeed())
		return n
	}

	Describe("Create", func() {
		It("Should create a named color with a new key", func(ctx SpecContext) {
			n := color.Named{Name: "Breaker", Light: red}
			Expect(w.Create(ctx, &n)).To(Succeed())
			Expect(n.Key).ToNot(Equal(uuid.Nil()))
			Expect(retrieve(ctx, n.Key)).To(Equal(n))
		})
		It("Should refuse a named color with no name", func(ctx SpecContext) {
			Expect(w.Create(ctx, &color.Named{Light: red})).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("name: required")),
			))
		})
		It("Should refuse a named color whose key exists", func(ctx SpecContext) {
			n := color.Named{Name: "Breaker", Light: red}
			Expect(w.Create(ctx, &n)).To(Succeed())
			again := color.Named{Key: n.Key, Name: "Valve", Light: red}
			Expect(w.Create(ctx, &again)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("color %s already exists", n.Key)),
			))
		})
	})

	Describe("Apply", func() {
		var n color.Named
		BeforeEach(func(ctx SpecContext) {
			n = color.Named{Name: "Breaker", Light: red}
			Expect(w.Create(ctx, &n)).To(Succeed())
		})
		It("Should store the result of the actions", func(ctx SpecContext) {
			Expect(w.Apply(
				ctx,
				n.Key,
				color.NewSetAction(color.SetPayload{
					Path:  []string{"name"},
					Value: "Valve",
				}),
				color.NewSetAction(color.SetPayload{
					Path:  []string{"dark"},
					Value: "#00ff00",
				}),
			)).To(Succeed())
			green := MustSucceed(xcolor.Parse("#00ff00"))
			Expect(retrieve(ctx, n.Key)).To(Equal(color.Named{
				Key:   n.Key,
				Name:  "Valve",
				Light: red,
				Dark:  &green,
			}))
		})
		It("Should refuse a result that is not valid", func(ctx SpecContext) {
			Expect(w.Apply(
				ctx,
				n.Key,
				color.NewSetAction(color.SetPayload{Path: []string{"name"}, Value: ""}),
			)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("name: required")),
			))
			Expect(retrieve(ctx, n.Key)).To(Equal(n))
		})
		It("Should refuse a create action", func(ctx SpecContext) {
			Expect(w.Apply(
				ctx,
				n.Key,
				color.NewCreateAction(color.CreatePayload{Named: n}),
			)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(
					"only the Core dispatches a named color create",
				)),
			))
		})
		It("Should return an error for a key with no named color", func(
			ctx SpecContext,
		) {
			Expect(w.Apply(
				ctx,
				uuid.New(),
				color.NewSetAction(
					color.SetPayload{Path: []string{"name"}, Value: "A"},
				),
			)).To(MatchError(query.ErrNotFound))
		})
	})

	Describe("Delete", func() {
		It("Should delete the named colors", func(ctx SpecContext) {
			a := color.Named{Name: "Breaker", Light: red}
			b := color.Named{Name: "Valve", Light: red}
			Expect(w.Create(ctx, &a)).To(Succeed())
			Expect(w.Create(ctx, &b)).To(Succeed())
			Expect(w.Delete(ctx, a.Key, b.Key)).To(Succeed())
			Expect(svc.NewRetrieve().Where(color.MatchKeys(a.Key, b.Key)).
				Exists(ctx, db)).To(BeFalse())
		})
		It("Should not return an error for a key with no named color", func(
			ctx SpecContext,
		) {
			Expect(w.Delete(ctx, uuid.New())).To(Succeed())
		})
	})
})
