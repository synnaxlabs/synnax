// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc_test

import (
	"context"
	"slices"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	. "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/arc/directory"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Directory dispatch", func() {
	type frame = actions.Scoped[directory.Key, directory.Action]
	var (
		root    directory.Key
		a, b    directory.Directory
		lib     directory.Directory
		created []directory.Key
	)
	write := func(ctx context.Context, fn func(arc.Writer) error) error {
		return db.WithTx(ctx, func(tx gorp.Tx) error {
			return fn(svc.NewWriter(tx).WithDispatchKey("dk"))
		})
	}
	createDirectory := func(
		ctx context.Context,
		name string,
		parent directory.Key,
	) directory.Directory {
		GinkgoHelper()
		d := directory.Directory{Name: name}
		Expect(write(ctx, func(w arc.Writer) error {
			return w.CreateDirectory(ctx, &d, parent)
		})).To(Succeed())
		created = append(created, d.Key)
		return d
	}
	BeforeEach(func(ctx SpecContext) {
		created = nil
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			root = rootDirectory(ctx, tx).Key
			return nil
		})).To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			for _, key := range slices.Backward(created) {
				Expect(write(ctx, func(w arc.Writer) error {
					return w.DeleteDirectory(ctx, key)
				})).To(Succeed())
			}
		})
		a = createDirectory(ctx, "dispatch_a", root)
		b = createDirectory(ctx, "dispatch_b", root)
		lib = createDirectory(ctx, "lib", a.Key)
	})
	set := func(path string, value any) directory.Action {
		return directory.NewSetAction(
			directory.SetPayload{Path: []string{path}, Value: value},
		)
	}
	apply := func(
		ctx context.Context,
		key directory.Key,
		acts ...directory.Action,
	) error {
		return write(ctx, func(w arc.Writer) error {
			return w.ApplyDirectory(ctx, key, acts...)
		})
	}
	record := func() *Recorder[directory.Key, directory.Action] {
		rec := &Recorder[directory.Key, directory.Action]{}
		DeferCleanup(svc.OnDirectoryAction(rec.Record))
		return rec
	}
	retrieve := func(ctx context.Context) directory.Directory {
		GinkgoHelper()
		var res directory.Directory
		Expect(svc.NewDirectoryRetrieve().
			Where(directory.MatchKeys(lib.Key)).
			Entry(&res).
			Exec(ctx, db)).To(Succeed())
		return res
	}

	It("Should store the edited name and parent", func(ctx SpecContext) {
		Expect(apply(ctx, lib.Key, set("name", "core"), set("parent", b.Key))).
			To(Succeed())
		Expect(retrieve(ctx)).To(Equal(directory.Directory{
			Key:    lib.Key,
			Name:   "core",
			Parent: &b.Key,
		}))
	})

	DescribeTable("Should refuse an edit", func(
		ctx SpecContext,
		act func(context.Context) directory.Action,
		msg string,
	) {
		err := apply(ctx, lib.Key, act(ctx))
		Expect(err).To(MatchError(validate.ErrValidation))
		Expect(err).To(MatchError(ContainSubstring(msg)))
		Expect(retrieve(ctx)).To(Equal(lib))
	},
		Entry("of the key", func(context.Context) directory.Action {
			return set("key", uuid.New())
		}, `unknown object member name "key"`),
		Entry("that clears the parent", func(context.Context) directory.Action {
			return set("parent", nil)
		}, "parent: required"),
		Entry("to a name that is not an identifier", func(
			context.Context,
		) directory.Action {
			return set("name", "my lib")
		}, "must contain only letters, digits, and underscores"),
		Entry("to a name used in the parent", func(
			ctx context.Context,
		) directory.Action {
			createDirectory(ctx, "taken", a.Key)
			return set("name", "taken")
		}, `name "taken" is already used`),
		Entry("to a parent that uses the name", func(
			ctx context.Context,
		) directory.Action {
			createDirectory(ctx, "lib", b.Key)
			return set("parent", b.Key)
		}, `name "lib" is already used`),
		Entry("to a parent below the directory", func(
			ctx context.Context,
		) directory.Action {
			return set("parent", createDirectory(ctx, "sub", lib.Key).Key)
		}, "would form a cycle"),
		Entry("to a parent that does not exist", func(
			context.Context,
		) directory.Action {
			return set("parent", uuid.New())
		}, "does not exist"),
	)

	It("Should refuse a change to the root", func(ctx SpecContext) {
		err := apply(ctx, root, set("name", "Automations"))
		Expect(err).To(MatchError(validate.ErrValidation))
		Expect(err).
			To(MatchError(ContainSubstring("root Arcs directory cannot be changed")))
	})

	It("Should broadcast a create and an applied batch", func(ctx SpecContext) {
		rec := record()
		d := createDirectory(ctx, "created", b.Key)
		Expect(apply(ctx, lib.Key, set("name", "core"))).To(Succeed())
		frames := rec.Snapshot()
		Expect(frames).To(Equal([]frame{
			{
				Key:         d.Key,
				DispatchKey: "dk",
				Seq:         frames[0].Seq,
				Actions: []directory.Action{
					directory.NewCreateAction(directory.CreatePayload{Directory: d}),
				},
			},
			{
				Key:         lib.Key,
				DispatchKey: "dk",
				Seq:         frames[1].Seq,
				Actions:     []directory.Action{set("name", "core")},
			},
		}))
	})

	DescribeTable("Should broadcast the set that an edit route applies", func(
		ctx SpecContext,
		edit func(context.Context, arc.Writer) error,
		path string,
		value func() any,
	) {
		rec := record()
		Expect(write(ctx, func(w arc.Writer) error {
			return edit(ctx, w)
		})).To(Succeed())
		frames := rec.Snapshot()
		Expect(frames).To(HaveLen(1))
		Expect(frames[0].Key).To(Equal(lib.Key))
		Expect(frames[0].Actions).
			To(Equal([]directory.Action{set(path, value())}))
	},
		Entry("for a rename", func(ctx context.Context, w arc.Writer) error {
			return w.RenameDirectory(ctx, lib.Key, "core")
		}, "name", func() any { return "core" }),
		Entry("for a move", func(ctx context.Context, w arc.Writer) error {
			return w.Move(ctx, a.Key, b.Key, lib.ResourceID())
		}, "parent", func() any { return b.Key }),
	)
})
