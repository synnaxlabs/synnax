// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package directory_test

import (
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/arc/directory"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Table", func() {
	var (
		table *directory.Table
		w     directory.Writer
	)
	BeforeEach(func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		table = MustOpen(directory.OpenTable(ctx, directory.TableConfig{DB: db}))
		w = table.NewWriter(nil)
	})
	create := func(
		ctx context.Context,
		name string,
		parent *directory.Key,
	) directory.Directory {
		GinkgoHelper()
		d := directory.Directory{Name: name, Parent: parent}
		Expect(w.Create(ctx, &d)).To(Succeed())
		return d
	}
	retrieve := func(ctx context.Context, f directory.Filter) []directory.Directory {
		GinkgoHelper()
		var res []directory.Directory
		Expect(table.NewRetrieve().Where(f).Entries(&res).Exec(ctx, nil)).
			To(Succeed())
		return res
	}

	Describe("OpenTable", func() {
		It("Should reject a config with no DB", func(ctx SpecContext) {
			Expect(directory.OpenTable(ctx)).Error().
				To(MatchError(ContainSubstring("db: must be non-nil")))
		})
	})

	Describe("Create", func() {
		It("Should give a directory with no key a new key", func(ctx SpecContext) {
			d := create(ctx, "lib", nil)
			Expect(d.Key).ToNot(Equal(uuid.Nil()))
			Expect(retrieve(ctx, directory.MatchKeys(d.Key))).
				To(ConsistOf(HaveField("Name", "lib")))
		})

		It("Should reject a key in use", func(ctx SpecContext) {
			d := create(ctx, "lib", nil)
			dup := directory.Directory{Key: d.Key, Name: "other"}
			Expect(w.Create(ctx, &dup)).To(MatchError(ContainSubstring(
				"directory " + d.Key.String() + " already exists",
			)))
		})

		It("Should reject a directory with no name", func(ctx SpecContext) {
			Expect(w.Create(ctx, &directory.Directory{})).
				To(MatchError(ContainSubstring("name: required")))
		})
	})

	Describe("Rename", func() {
		It("Should give the directory the new name", func(ctx SpecContext) {
			d := create(ctx, "lib", nil)
			Expect(w.Rename(ctx, d.Key, "core")).To(Succeed())
			Expect(retrieve(ctx, directory.MatchKeys(d.Key))).
				To(ConsistOf(HaveField("Name", "core")))
		})
	})

	Describe("SetParent", func() {
		It("Should move the directories into the parent", func(ctx SpecContext) {
			parent := create(ctx, "parent", nil)
			a, b := create(ctx, "a", nil), create(ctx, "b", nil)
			Expect(w.SetParent(ctx, parent.Key, a.Key, b.Key)).To(Succeed())
			Expect(retrieve(ctx, directory.MatchParents(parent.Key))).To(ConsistOf(
				HaveField("Key", a.Key),
				HaveField("Key", b.Key),
			))
		})

		It("Should reject a move below itself", func(ctx SpecContext) {
			top := create(ctx, "top", nil)
			child := create(ctx, "child", &top.Key)
			Expect(w.SetParent(ctx, child.Key, top.Key)).
				To(MatchError(ContainSubstring("would form a cycle")))
		})
	})

	Describe("Delete", func() {
		It("Should delete an empty directory", func(ctx SpecContext) {
			d := create(ctx, "lib", nil)
			Expect(w.Delete(ctx, d.Key)).To(Succeed())
			Expect(table.NewRetrieve().
				Where(directory.MatchKeys(d.Key)).
				Entry(&directory.Directory{}).
				Exec(ctx, nil)).To(MatchError(query.ErrNotFound))
		})

		It("Should reject a directory that holds a directory", func(ctx SpecContext) {
			d := create(ctx, "lib", nil)
			create(ctx, "inner", &d.Key)
			Expect(w.Delete(ctx, d.Key)).
				To(MatchError(ContainSubstring(`directory "lib" is not empty`)))
		})
	})

	Describe("MatchNested", func() {
		DescribeTable(
			"Should match by whether another directory holds the directory",
			func(ctx SpecContext, nested bool, expected string) {
				top := create(ctx, "top", nil)
				create(ctx, "inner", &top.Key)
				Expect(retrieve(ctx, directory.MatchNested(nested))).
					To(ConsistOf(HaveField("Name", expected)))
			},
			Entry("nested", true, "inner"),
			Entry("at the root", false, "top"),
		)
	})
})

var _ = Describe("ResourceID", func() {
	It("Should round-trip a key through its resource ID", func() {
		key := uuid.New()
		id := directory.ResourceID(key)
		Expect(id.Type).To(Equal(resource.TypeArcDirectory))
		Expect(directory.KeyFromResourceID(id)).To(Equal(key))
	})

	DescribeTable(
		"Should reject an invalid resource ID",
		func(id resource.ID, msg string) {
			Expect(directory.KeyFromResourceID(id)).Error().To(And(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(msg)),
			))
		},
		Entry(
			"of another type",
			resource.ID{Type: resource.TypeChannel, Key: uuid.New().String()},
			`must be an Arc directory, got "channel"`,
		),
		Entry(
			"with a key that is not a UUID",
			resource.ID{Type: resource.TypeArcDirectory, Key: "bad"},
			`invalid Arc directory key "bad"`,
		),
	)
})
