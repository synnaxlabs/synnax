// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package group_test

import (
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Sibling breaker", Ordered, ContinueOnFailure, func() {
	var (
		db  *gorp.DB
		svc *group.Service
	)
	BeforeAll(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New()))
		StartRun(db.Run)
		svc = MustOpen(group.OpenService(ctx, group.ServiceConfig{DB: db}))
	})
	unique := func(prefix string) string { return prefix + " " + uuid.New().String() }
	write := func(ctx context.Context, f func(w group.Writer) error) error {
		return db.WithTx(ctx, func(tx gorp.Tx) error { return f(svc.NewWriter(tx)) })
	}
	create := func(
		ctx context.Context,
		name string,
		scope resource.Type,
		parent *group.Group,
	) (group.Group, error) {
		g := group.Group{Name: name, Scope: scope}
		if parent != nil {
			g.Parent = new(parent.ResourceID())
		}
		err := write(ctx, func(w group.Writer) error { return w.Create(ctx, &g) })
		return g, err
	}
	createErr := func(
		ctx context.Context,
		name string,
		scope resource.Type,
		parent *group.Group,
	) error {
		_, err := create(ctx, name, scope, parent)
		return err
	}
	taken := MatchError(ContainSubstring("already holds the name"))

	It("Should refuse two root folders of the channel scope with one name", func(
		ctx SpecContext,
	) {
		name := unique("root")
		MustSucceed(create(ctx, name, resource.TypeChannel, nil))
		Expect(createErr(ctx, name, resource.TypeChannel, nil)).To(taken)
		Expect(
			createErr(ctx, name, resource.TypeChannel, nil),
		).To(MatchError(validate.ErrValidation))
	})

	It("Should keep a folder writable when it is renamed to its own name", func(
		ctx SpecContext,
	) {
		g := MustSucceed(create(ctx, unique("self"), resource.TypeChannel, nil))
		Expect(write(ctx, func(w group.Writer) error {
			return w.Rename(ctx, g.Key, g.Name)
		})).To(Succeed())
	})

	It("Should refuse a rename onto a sibling sent in another normal form", func(
		ctx SpecContext,
	) {
		suffix := uuid.New().String()
		MustSucceed(create(ctx, "\u00C5 "+suffix, resource.TypeChannel, nil))
		g := MustSucceed(create(ctx, unique("nfc"), resource.TypeChannel, nil))
		Expect(write(ctx, func(w group.Writer) error {
			return w.Rename(ctx, g.Key, "A\u030A "+suffix)
		})).To(taken)
		Expect(createErr(ctx, "A\u030A "+suffix, resource.TypeChannel, nil)).To(taken)
	})

	It(
		"Should refuse a folder moved into a parent that holds a folder of its name",
		func(
			ctx SpecContext,
		) {
			name := unique("moved")
			parent := MustSucceed(
				create(ctx, unique("parent"), resource.TypeChannel, nil),
			)
			MustSucceed(create(ctx, name, resource.TypeChannel, &parent))
			g := MustSucceed(create(ctx, name, resource.TypeChannel, nil))
			id := parent.ResourceID()
			Expect(write(ctx, func(w group.Writer) error {
				return w.SetParent(ctx, &id, g.Key)
			})).To(taken)
			deep := MustSucceed(
				create(ctx, unique("deep"), resource.TypeChannel, &parent),
			)
			Expect(write(ctx, func(w group.Writer) error {
				return w.SetParent(ctx, nil, deep.Key)
			})).To(Succeed())
		},
	)

	It("Should take a name back after a delete in one transaction", func(
		ctx SpecContext,
	) {
		name := unique("recreate")
		g := MustSucceed(create(ctx, name, resource.TypeChannel, nil))
		Expect(write(ctx, func(w group.Writer) error {
			if err := w.Delete(ctx, g.Key); err != nil {
				return err
			}
			again := group.Group{Name: name, Scope: resource.TypeChannel}
			return w.Create(ctx, &again)
		})).To(Succeed())
	})

	It("Should swap the names of two sibling folders in one transaction", func(
		ctx SpecContext,
	) {
		a := MustSucceed(create(ctx, unique("a"), resource.TypeChannel, nil))
		b := MustSucceed(create(ctx, unique("b"), resource.TypeChannel, nil))
		Expect(write(ctx, func(w group.Writer) error {
			if err := w.Rename(ctx, a.Key, unique("tmp")); err != nil {
				return err
			}
			if err := w.Rename(ctx, b.Key, a.Name); err != nil {
				return err
			}
			return w.Rename(ctx, a.Key, b.Name)
		})).To(Succeed())
	})

	It("Should allow repeated names in the groups of other scopes", func(
		ctx SpecContext,
	) {
		for _, scope := range []resource.Type{
			resource.TypeProject,
			resource.TypeColor,
			resource.TypeSchematicSymbol,
		} {
			name := unique("other")
			MustSucceed(create(ctx, name, scope, nil))
			MustSucceed(create(ctx, name, scope, nil))
		}
	})

	It("Should refuse an empty or blank name in the channel scope", func(
		ctx SpecContext,
	) {
		Expect(
			createErr(ctx, "", resource.TypeChannel, nil),
		).To(MatchError(validate.ErrValidation))
		Expect(
			createErr(ctx, "   ", resource.TypeChannel, nil),
		).To(MatchError(validate.ErrValidation))
		Expect(
			createErr(ctx, "a\u200Bb", resource.TypeChannel, nil),
		).To(MatchError(validate.ErrValidation))
	})

	It(
		"Should retrieve a root folder that CreateOrRetrieve names in another normal form",
		func(
			ctx SpecContext,
		) {
			suffix := uuid.New().String()
			stored := MustSucceed(
				create(ctx, "\u00C5 "+suffix, resource.TypeChannel, nil),
			)
			got := MustSucceed(svc.CreateOrRetrieve(
				ctx, "A\u030A "+suffix, resource.TypeChannel,
			))
			Expect(got.Key).To(Equal(stored.Key))
		},
	)
})
