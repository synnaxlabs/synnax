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
	"strings"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/color"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	xcolor "github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Writer", func() {
	var (
		db     *gorp.DB
		svc    *color.Service
		groups *group.Service
		tx     gorp.Tx
		w      color.Writer
		red    xcolor.Color
	)
	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New()))
		StartRun(db.Run)
		groups = MustOpen(group.OpenService(ctx, group.ServiceConfig{DB: db}))
		svc = MustOpen(
			color.OpenService(ctx, color.ServiceConfig{DB: db, Group: groups}),
		)
		tx = DeferClose(db.OpenTx())
		w = svc.NewWriter(tx)
		red = MustSucceed(xcolor.Parse("#ff0000"))
	})
	retrieve := func(ctx SpecContext, key color.Key) color.Named {
		GinkgoHelper()
		var n color.Named
		Expect(svc.NewRetrieve().
			Where(color.MatchKeys(key)).
			Entry(&n).
			Exec(ctx, tx)).To(Succeed())
		return n
	}
	createGroup := func(ctx SpecContext, scope resource.Type) group.Group {
		GinkgoHelper()
		g := group.Group{Name: "Series", Scope: scope}
		Expect(groups.NewWriter(tx).Create(ctx, &g)).To(Succeed())
		return g
	}
	setGroup := func(v any) color.Action {
		return color.NewSetAction(color.SetPayload{Path: []string{"group"}, Value: v})
	}

	Describe("Position", func() {
		DescribeTable(
			"Should store a fractional index",
			func(ctx SpecContext, position string) {
				n := color.Named{Name: "Breaker", Light: red, Position: &position}
				Expect(w.Create(ctx, &n)).To(Succeed())
				Expect(retrieve(ctx, n.Key).Position).To(Equal(&position))
			},
			Entry("an integer", "a0"),
			Entry("a fraction", "a0V"),
			Entry("a negative integer", "Zz"),
			Entry("a longer integer", "b00"),
		)
		DescribeTable(
			"Should refuse a position that is not a fractional index",
			func(ctx SpecContext, position string) {
				n := color.Named{Name: "Breaker", Light: red, Position: &position}
				Expect(w.Create(ctx, &n)).To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring("position: invalid fractional index")),
				))
			},
			Entry("an empty string", ""),
			Entry("a head with no digits", "a"),
			Entry("an integer that is too short", "b0"),
			Entry("a fraction that ends in zero", "a0V0"),
			Entry("a digit outside base 62", "a0!"),
			Entry("a head that is not a letter", "00"),
			Entry("the smallest integer", "A"+strings.Repeat("0", 26)),
		)
		It("Should refuse a change to a position that is not valid", func(
			ctx SpecContext,
		) {
			n := color.Named{Name: "Breaker", Light: red}
			Expect(w.Create(ctx, &n)).To(Succeed())
			Expect(w.Apply(ctx, n.Key, color.NewSetAction(
				color.SetPayload{Path: []string{"position"}, Value: "a00"},
			))).To(MatchError(ContainSubstring("position: invalid fractional index")))
		})
	})

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
		It("Should create a named color in a palette", func(ctx SpecContext) {
			g := createGroup(ctx, resource.TypeColor)
			n := color.Named{Name: "Breaker", Light: red, Group: &g.Key}
			Expect(w.Create(ctx, &n)).To(Succeed())
			var palette []color.Named
			Expect(svc.NewRetrieve().
				Where(color.MatchGroups(g.Key)).
				Entries(&palette).
				Exec(ctx, tx)).To(Succeed())
			Expect(palette).To(Equal([]color.Named{n}))
		})
		It("Should refuse a group of another scope", func(ctx SpecContext) {
			g := createGroup(ctx, resource.TypeChannel)
			n := color.Named{Name: "Breaker", Light: red, Group: &g.Key}
			Expect(w.Create(ctx, &n)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(
					"group: a color cannot sit in a channel group",
				)),
			))
		})
		It("Should refuse a group that does not exist", func(ctx SpecContext) {
			k := uuid.New()
			n := color.Named{Name: "Breaker", Light: red, Group: &k}
			Expect(w.Create(ctx, &n)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("group: group %s does not exist", k)),
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

	Describe("CreateMany", func() {
		It("Should create each named color with a new key", func(ctx SpecContext) {
			named := []color.Named{
				{Name: "Breaker", Light: red},
				{Name: "Valve", Light: red},
			}
			Expect(w.CreateMany(ctx, named)).To(Succeed())
			for _, n := range named {
				Expect(n.Key).ToNot(Equal(uuid.Nil()))
				Expect(retrieve(ctx, n.Key)).To(Equal(n))
			}
		})
		It("Should stop at the first named color that is not valid", func(
			ctx SpecContext,
		) {
			named := []color.Named{
				{Name: "Breaker", Light: red},
				{Light: red},
				{Name: "Valve", Light: red},
			}
			Expect(w.CreateMany(ctx, named)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("name: required")),
			))
			Expect(named[2].Key).To(Equal(uuid.Nil()))
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
		It("Should move a named color into a palette and out", func(ctx SpecContext) {
			g := createGroup(ctx, resource.TypeColor)
			Expect(w.Apply(ctx, n.Key, setGroup(g.Key))).To(Succeed())
			Expect(retrieve(ctx, n.Key).Group).To(Equal(&g.Key))
			Expect(w.Apply(ctx, n.Key, setGroup(nil))).To(Succeed())
			Expect(retrieve(ctx, n.Key)).To(Equal(n))
		})
		It("Should refuse a move into a group of another scope", func(ctx SpecContext) {
			g := createGroup(ctx, resource.TypeChannel)
			Expect(w.Apply(ctx, n.Key, setGroup(g.Key))).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(
					"group: a color cannot sit in a channel group",
				)),
			))
			Expect(retrieve(ctx, n.Key)).To(Equal(n))
		})
		It("Should keep a palette that holds a named color", func(ctx SpecContext) {
			g := createGroup(ctx, resource.TypeColor)
			Expect(w.Apply(ctx, n.Key, setGroup(g.Key))).To(Succeed())
			Expect(groups.NewWriter(tx).Delete(ctx, g.Key)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(
					"cannot delete group Series because it is not empty",
				)),
			))
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
				Exists(ctx, tx)).To(BeFalse())
		})
		It("Should not return an error for a key with no named color", func(
			ctx SpecContext,
		) {
			Expect(w.Delete(ctx, uuid.New())).To(Succeed())
		})
	})
})
