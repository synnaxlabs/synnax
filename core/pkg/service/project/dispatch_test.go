// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package project_test

import (
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	. "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Dispatch", func() {
	type frame = actions.Scoped[project.Key, project.Action]
	var (
		g group.Group
		p project.Project
	)
	write := func(ctx context.Context, fn func(project.Writer) error) error {
		return db.WithTx(ctx, func(tx gorp.Tx) error {
			return fn(svc.NewWriter(tx).WithDispatchKey("dk"))
		})
	}
	BeforeEach(func(ctx SpecContext) {
		g = group.Group{Name: "Projects", Scope: resource.TypeProject}
		p = project.Project{Name: "Before"}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return groupSvc.NewWriter(tx).Create(ctx, &g)
		})).To(Succeed())
		Expect(write(ctx, func(w project.Writer) error {
			return w.Create(ctx, &p)
		})).To(Succeed())
	})
	set := func(path string, value any) project.Action {
		return project.NewSetAction(
			project.SetPayload{Path: []string{path}, Value: value},
		)
	}
	apply := func(ctx context.Context, acts ...project.Action) error {
		return write(ctx, func(w project.Writer) error {
			_, err := w.Apply(ctx, p.Key, acts...)
			return err
		})
	}
	record := func() *Recorder[project.Key, project.Action] {
		rec := &Recorder[project.Key, project.Action]{}
		DeferCleanup(svc.Actions().OnChange(rec.Record))
		return rec
	}
	retrieve := func(ctx context.Context) project.Project {
		GinkgoHelper()
		var res project.Project
		Expect(svc.NewRetrieve().
			Where(project.MatchKeys(p.Key)).
			Entry(&res).
			Exec(ctx, db)).To(Succeed())
		return res
	}

	It("Should store the edited name and group", func(ctx SpecContext) {
		Expect(apply(ctx, set("name", "After"), set("group", g.Key))).To(Succeed())
		Expect(retrieve(ctx)).
			To(Equal(project.Project{Key: p.Key, Name: "After", Group: &g.Key}))
	})

	It("Should refuse a path outside the editable fields", func(ctx SpecContext) {
		err := apply(ctx, set("key", uuid.New()))
		Expect(err).To(MatchError(validate.ErrValidation))
		Expect(err).To(MatchError(ContainSubstring(`unknown object member name "key"`)))
		Expect(retrieve(ctx)).To(Equal(p))
	})

	It("Should refuse a group that cannot hold a project", func(ctx SpecContext) {
		other := group.Group{Name: "Channels", Scope: resource.TypeChannel}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return groupSvc.NewWriter(tx).Create(ctx, &other)
		})).To(Succeed())
		err := apply(ctx, set("group", other.Key))
		Expect(err).To(MatchError(validate.ErrValidation))
		Expect(err).
			To(MatchError(ContainSubstring("a project cannot sit in a channel group")))
		Expect(retrieve(ctx)).To(Equal(p))
	})

	It("Should broadcast a create and an applied batch", func(ctx SpecContext) {
		rec := record()
		created := project.Project{Name: "Created"}
		Expect(write(ctx, func(w project.Writer) error {
			return w.Create(ctx, &created)
		})).To(Succeed())
		Expect(apply(ctx, set("name", "After"))).To(Succeed())
		frames := rec.Snapshot()
		Expect(frames).To(Equal([]frame{
			{
				Key:         created.Key,
				DispatchKey: "dk",
				Seq:         frames[0].Seq,
				Actions: []project.Action{
					project.NewCreateAction(project.CreatePayload{Project: created}),
				},
			},
			{
				Key:         p.Key,
				DispatchKey: "dk",
				Seq:         frames[1].Seq,
				Actions:     []project.Action{set("name", "After")},
			},
		}))
	})

	DescribeTable("Should broadcast the set of each edit route", func(
		ctx SpecContext,
		edit func(context.Context, project.Writer) error,
		want func() project.Action,
	) {
		rec := record()
		Expect(write(ctx, func(w project.Writer) error { return edit(ctx, w) })).
			To(Succeed())
		frames := rec.Snapshot()
		Expect(frames).To(HaveLen(1))
		Expect(frames[0].Actions).To(Equal([]project.Action{want()}))
	},
		Entry("a rename", func(ctx context.Context, w project.Writer) error {
			return w.Rename(ctx, p.Key, "After")
		}, func() project.Action { return set("name", "After") }),
		Entry("a group change", func(ctx context.Context, w project.Writer) error {
			return w.SetGroup(ctx, &g.Key, p.Key)
		}, func() project.Action { return set("group", &g.Key) }),
	)

	It("Should broadcast one set for each project of a group change", func(
		ctx SpecContext,
	) {
		other := project.Project{Name: "Other"}
		Expect(write(ctx, func(w project.Writer) error {
			return w.Create(ctx, &other)
		})).To(Succeed())
		rec := record()
		Expect(write(ctx, func(w project.Writer) error {
			return w.SetGroup(ctx, &g.Key, p.Key, other.Key)
		})).To(Succeed())
		frames := rec.Snapshot()
		Expect(frames).To(HaveLen(2))
		want := []project.Action{set("group", &g.Key)}
		Expect(frames).To(Equal([]frame{
			{Key: p.Key, DispatchKey: "dk", Seq: frames[0].Seq, Actions: want},
			{Key: other.Key, DispatchKey: "dk", Seq: frames[1].Seq, Actions: want},
		}))
	})
})
