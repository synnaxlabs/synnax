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
	"github.com/synnaxlabs/x/search"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Table", func() {
	var (
		db    *gorp.DB
		table *directory.Table
	)
	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New()))
		table = MustOpen(directory.OpenTable(ctx, directory.TableConfig{DB: db}))
	})
	create := func(
		ctx context.Context,
		name string,
		parent *directory.Key,
	) directory.Directory {
		GinkgoHelper()
		d := directory.Directory{Name: name, Parent: parent}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return table.NewWriter(tx).Create(ctx, &d)
		})).To(Succeed())
		return d
	}
	retrieve := func(ctx context.Context, f directory.Filter) []directory.Directory {
		GinkgoHelper()
		var res []directory.Directory
		Expect(table.NewRetrieve().Where(f).Entries(&res).Exec(ctx, db)).
			To(Succeed())
		return res
	}

	Describe("OpenTable", func() {
		It("Should reject a config with no DB", func(ctx SpecContext) {
			Expect(directory.OpenTable(ctx)).Error().
				To(MatchError(ContainSubstring("db: must be non-nil")))
		})
	})

	Describe("Changes", func() {
		It("Should return the directory that a commit creates", func(ctx SpecContext) {
			var names []string
			DeferCleanup(
				db.Observe().OnChange(func(ctx context.Context, c gorp.Commit) {
					for ch := range table.Changes(ctx, c) {
						names = append(names, ch.Value.Name)
					}
				}),
			)
			create(ctx, "lib", nil)
			Expect(names).To(Equal([]string{"lib"}))
		})
	})

	Describe("Observe", func() {
		It("Should notify observers of a created directory", func(ctx SpecContext) {
			var names []string
			DeferCleanup(table.Observe().OnChange(func(
				_ context.Context,
				r gorp.TxReader[directory.Key, directory.Directory],
			) {
				for ch := range r {
					names = append(names, ch.Value.Name)
				}
			}))
			create(ctx, "lib", nil)
			Eventually(func() []string { return names }).Should(Equal([]string{"lib"}))
		})
	})

	Describe("Create", func() {
		It("Should give a directory with no key a new key", func(ctx SpecContext) {
			d := create(ctx, "lib", nil)
			Expect(d.Key).ToNot(Equal(uuid.Nil()))
			Expect(retrieve(ctx, directory.MatchKeys(d.Key))).
				To(ConsistOf(HaveField("Name", "lib")))
		})

		It("Should refuse a directory whose key exists", func(ctx SpecContext) {
			d := create(ctx, "lib", nil)
			dup := directory.Directory{Key: d.Key, Name: "other"}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return table.NewWriter(tx).Create(ctx, &dup)
			})).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("arc_directory %s already exists", d.Key)),
			))
			Expect(retrieve(ctx, directory.MatchKeys(d.Key))).
				To(ConsistOf(HaveField("Name", "lib")))
		})

		It("Should reject a directory with no name", func(ctx SpecContext) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return table.NewWriter(tx).Create(ctx, &directory.Directory{})
			})).
				To(MatchError(ContainSubstring("name: required")))
		})
	})

	Describe("Update", func() {
		It("Should store the changed directory", func(ctx SpecContext) {
			parent := create(ctx, "parent", nil)
			d := create(ctx, "lib", nil)
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return table.NewWriter(tx).Update(ctx, d.Key, func(
					d directory.Directory,
				) (directory.Directory, error) {
					d.Name, d.Parent = "core", &parent.Key
					return d, nil
				})
			})).To(Succeed())
			Expect(retrieve(ctx, directory.MatchKeys(d.Key))).To(ConsistOf(
				directory.Directory{Key: d.Key, Name: "core", Parent: &parent.Key},
			))
		})

		It("Should reject an invalid directory", func(ctx SpecContext) {
			d := create(ctx, "lib", nil)
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return table.NewWriter(tx).Update(ctx, d.Key, func(
					d directory.Directory,
				) (directory.Directory, error) {
					d.Name = ""
					return d, nil
				})
			})).To(MatchError(ContainSubstring("name: required")))
		})

		It("Should reject a move below itself", func(ctx SpecContext) {
			top := create(ctx, "top", nil)
			child := create(ctx, "child", &top.Key)
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return table.NewWriter(tx).Update(ctx, top.Key, func(
					d directory.Directory,
				) (directory.Directory, error) {
					d.Parent = &child.Key
					return d, nil
				})
			})).
				To(MatchError(ContainSubstring("would form a cycle")))
		})
	})

	Describe("Delete", func() {
		It("Should delete an empty directory", func(ctx SpecContext) {
			d := create(ctx, "lib", nil)
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return table.NewWriter(tx).Delete(ctx, d.Key)
			})).To(Succeed())
			Expect(table.NewRetrieve().
				Where(directory.MatchKeys(d.Key)).
				Entry(&directory.Directory{}).
				Exec(ctx, db)).To(MatchError(query.ErrNotFound))
		})

		It("Should reject a directory that holds a directory", func(ctx SpecContext) {
			d := create(ctx, "lib", nil)
			create(ctx, "inner", &d.Key)
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return table.NewWriter(tx).Delete(ctx, d.Key)
			})).
				To(MatchError(ContainSubstring(`directory "lib" is not empty`)))
		})
	})

	Describe("Resources", func() {
		named := func(d directory.Directory) resource.Resource {
			return resource.Resource{ID: d.ResourceID(), Name: d.Name}
		}

		It("Should name each directory by its name", func(ctx SpecContext) {
			d := create(ctx, "lib", nil)
			Expect(table.Resources().Retrieve(ctx, db, []resource.ID{d.ResourceID()})).
				To(Equal([]resource.Resource{named(d)}))
		})

		It("Should leave out an ID that names no directory", func(ctx SpecContext) {
			d := create(ctx, "lib", nil)
			missing := directory.ResourceID(uuid.New())
			Expect(table.Resources().Retrieve(
				ctx,
				db,
				[]resource.ID{missing, d.ResourceID()},
			)).To(Equal([]resource.Resource{named(d)}))
		})

		It(
			"Should return a validation error for a malformed key",
			func(ctx SpecContext) {
				Expect(table.Resources().Retrieve(
					ctx,
					db,
					[]resource.ID{{Type: resource.TypeArcDirectory, Key: "bad"}},
				)).Error().To(And(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring(`invalid Arc directory key "bad"`)),
				))
			},
		)

		It("Should list each directory by its name", func(ctx SpecContext) {
			lib := create(ctx, "lib", nil)
			sub := create(ctx, "sub", &lib.Key)
			Expect(table.Resources().List(ctx, db, 10, nil)).
				To(ConsistOf(named(lib), named(sub)))
		})

		It(
			"Should return the matching directories, closest match first",
			func(ctx SpecContext) {
				a := create(ctx, "turbopump test", nil)
				b := create(ctx, "turbopump", nil)
				q := search.Query[resource.ID]{Term: "turbopump"}
				Expect(MustSucceed(table.Resources().Search(ctx, db, q)).Keys).
					To(Equal([]resource.ID{b.ResourceID(), a.ResourceID()}))
			},
		)

		It(
			"Should not match a staged directory outside its transaction",
			func(ctx SpecContext) {
				tx := DeferClose(db.OpenTx())
				d := directory.Directory{Name: "injector"}
				Expect(table.NewWriter(tx).Create(ctx, &d)).To(Succeed())
				q := search.Query[resource.ID]{Term: "injector"}
				Expect(MustSucceed(table.Resources().Search(ctx, tx, q)).Keys).
					To(Equal([]resource.ID{d.ResourceID()}))
				Expect(MustSucceed(table.Resources().Search(ctx, db, q)).Keys).
					To(BeEmpty())
			},
		)
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
