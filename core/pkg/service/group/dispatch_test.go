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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	. "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Dispatch", func() {
	type frame = actions.Scoped[group.Key, group.Action]
	var (
		db            *gorp.DB
		svc           *group.Service
		parent, child group.Group
		moved         []group.Key
	)
	write := func(ctx context.Context, fn func(group.Writer) error) error {
		return db.WithTx(ctx, func(tx gorp.Tx) error {
			return fn(svc.NewWriter(tx).WithDispatchKey("dk"))
		})
	}
	create := func(ctx context.Context, g *group.Group) {
		GinkgoHelper()
		Expect(write(ctx, func(w group.Writer) error { return w.Create(ctx, g) })).
			To(Succeed())
	}
	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New()))
		moved = nil
		svc = MustOpen(group.OpenService(ctx, group.ServiceConfig{
			DB: db,
			MoveMembers: func(
				_ context.Context,
				_ gorp.Tx,
				groups []group.Key,
				_ *resource.ID,
			) error {
				moved = append(moved, groups...)
				return nil
			},
		}))
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, project]{
			DB:   db,
			Name: "project",
		}))
		Expect(gorp.NewCreate[string, project]().
			Entry(&project{Key: "p1"}).
			Exec(ctx, db)).To(Succeed())
		parent = group.Group{Name: "Parent", Scope: resource.TypeProject}
		create(ctx, &parent)
		child = group.Group{Name: "Child", Scope: resource.TypeProject}
		create(ctx, &child)
	})
	set := func(path string, value any) group.Action {
		return group.NewSetAction(group.SetPayload{Path: []string{path}, Value: value})
	}
	setParent := func(id *resource.ID) group.Action {
		return group.NewSetParentAction(group.SetParentPayload{Parent: id})
	}
	apply := func(ctx context.Context, acts ...group.Action) error {
		return write(ctx, func(w group.Writer) error {
			return w.Apply(ctx, child.Key, acts...)
		})
	}
	record := func() *Recorder[group.Key, group.Action] {
		rec := &Recorder[group.Key, group.Action]{}
		DeferCleanup(svc.Actions().OnChange(rec.Record))
		return rec
	}
	retrieve := func(ctx context.Context) group.Group {
		GinkgoHelper()
		var res group.Group
		Expect(svc.NewRetrieve().
			Where(group.MatchKeys(child.Key)).
			Entry(&res).
			Exec(ctx, db)).To(Succeed())
		return res
	}

	It("Should store the edited name and parent", func(ctx SpecContext) {
		id := parent.ResourceID()
		Expect(apply(ctx, set("name", "Renamed"), setParent(&id))).To(Succeed())
		Expect(retrieve(ctx)).To(Equal(group.Group{
			Key:    child.Key,
			Name:   "Renamed",
			Scope:  resource.TypeProject,
			Parent: &id,
		}))
	})

	It("Should refuse a path to the parent", func(ctx SpecContext) {
		err := apply(ctx, set("parent", parent.ResourceID()))
		Expect(err).To(MatchError(validate.ErrValidation))
		Expect(
			err,
		).To(MatchError(ContainSubstring(`unknown object member name "parent"`)))
		Expect(retrieve(ctx)).To(Equal(child))
	})

	It("Should move the records of a group under a new top parent", func(
		ctx SpecContext,
	) {
		Expect(apply(ctx, setParent(new(parent.ResourceID())))).To(Succeed())
		Expect(moved).To(BeEmpty())
		Expect(apply(ctx, setParent(&resource.ID{
			Type: resource.TypeProject,
			Key:  "p1",
		}))).To(Succeed())
		Expect(moved).To(Equal([]group.Key{child.Key}))
	})

	It("Should broadcast a create and an applied batch", func(ctx SpecContext) {
		rec := record()
		created := group.Group{Name: "Created", Scope: resource.TypeChannel}
		create(ctx, &created)
		Expect(apply(ctx, set("name", "Renamed"))).To(Succeed())
		frames := rec.Snapshot()
		Expect(frames).To(Equal([]frame{
			{
				Key:         created.Key,
				DispatchKey: "dk",
				Seq:         frames[0].Seq,
				Actions: []group.Action{
					group.NewCreateAction(group.CreatePayload{Group: created}),
				},
			},
			{
				Key:         child.Key,
				DispatchKey: "dk",
				Seq:         frames[1].Seq,
				Actions:     []group.Action{set("name", "Renamed")},
			},
		}))
	})

	DescribeTable("Should broadcast the action of each edit route", func(
		ctx SpecContext,
		edit func(context.Context, group.Writer) error,
		want func() group.Action,
	) {
		rec := record()
		Expect(write(ctx, func(w group.Writer) error { return edit(ctx, w) })).
			To(Succeed())
		frames := rec.Snapshot()
		Expect(frames).To(HaveLen(1))
		Expect(frames[0].Actions).To(Equal([]group.Action{want()}))
	},
		Entry("a rename", func(ctx context.Context, w group.Writer) error {
			return w.Rename(ctx, child.Key, "Renamed")
		}, func() group.Action { return set("name", "Renamed") }),
		Entry("a parent change", func(ctx context.Context, w group.Writer) error {
			return w.SetParent(ctx, new(parent.ResourceID()), child.Key)
		}, func() group.Action { return setParent(new(parent.ResourceID())) }),
	)
})
