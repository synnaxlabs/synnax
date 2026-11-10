// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package table_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	. "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/table"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/union"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Writer", func() {
	// seed creates a 2x2 table named "test" with cells a,b across row 0 and
	// no second row populated; only the writer's persistence path is
	// exercised. Reducer behavior is covered in actions_test.go.
	seed := func(ctx SpecContext) table.Table {
		GinkgoHelper()
		s := table.Table{
			Project: &proj.Key,
			Name:    "test",
			Rows:    []table.Row{{Key: "r0", Size: 30, Cells: []string{"a", "b"}}},
			Columns: []table.Column{{Key: "c0", Size: 80}, {Key: "c1", Size: 100}},
			Cells: map[string]table.CellConfig{
				"a": textCfg("A"),
				"b": textCfg("B"),
			},
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, &s)
		})).To(Succeed())
		return s
	}

	retrieve := func(ctx SpecContext, key table.Key) table.Table {
		GinkgoHelper()
		var res table.Table
		Expect(
			svc.NewRetrieve().Where(table.MatchKeys(key)).Entry(&res).Exec(ctx, db),
		).To(Succeed())
		return res
	}

	Describe("Create", func() {
		It("Should create a Table", func(ctx SpecContext) {
			tx := DeferClose(db.OpenTx())
			t := table.Table{
				Project: &proj.Key,
				Name:    "test",
				Rows:    []table.Row{{Key: "r0", Size: 30, Cells: []string{"a"}}},
				Columns: []table.Column{{Key: "c0", Size: 80}},
				Cells: map[string]table.CellConfig{
					"a": textCfg("hello"),
				},
			}
			Expect(svc.NewWriter(tx).Create(ctx, &t)).To(Succeed())
			Expect(t.Key).ToNot(Equal(uuid.Nil()))
		})

		It(
			"Should return a validation error when the name is empty",
			func(ctx SpecContext) {
				tx := DeferClose(db.OpenTx())
				t := table.Table{Project: &proj.Key}
				Expect(svc.NewWriter(tx).Create(ctx, &t)).
					To(MatchError(ContainSubstring("name: required")))
			},
		)

		It("Should refuse a table whose key exists", func(ctx SpecContext) {
			stored := seed(ctx)
			again := table.Table{Key: stored.Key, Project: &proj.Key, Name: "refused"}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &again)
			})).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("table %s already exists", stored.Key)),
			))
			Expect(retrieve(ctx, stored.Key)).To(Equal(stored))
		})

		It(
			"Should reject a table with neither a project nor a range",
			func(ctx SpecContext) {
				tx := DeferClose(db.OpenTx())
				t := table.Table{Name: "test"}
				Expect(
					svc.NewWriter(tx).Create(ctx, &t),
				).To(MatchError(ContainSubstring(
					"project: must belong to exactly one of a project and a range",
				)))
			},
		)

		DescribeTable(
			"Should reject a table whose rows or columns share a key",
			func(ctx SpecContext, rows []table.Row, cols []table.Column, msg string) {
				tx := DeferClose(db.OpenTx())
				t := table.Table{
					Project: &proj.Key,
					Name:    "test",
					Rows:    rows,
					Columns: cols,
				}
				Expect(svc.NewWriter(tx).Create(ctx, &t)).To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(msg),
				))
			},
			Entry(
				"rows",
				[]table.Row{{Key: "r0", Size: 30}, {Key: "r0", Size: 30}},
				[]table.Column(nil),
				`duplicate row key "r0": validation error`,
			),
			Entry(
				"columns",
				[]table.Row(nil),
				[]table.Column{{Key: "c0", Size: 80}, {Key: "c0", Size: 80}},
				`duplicate column key "c0": validation error`,
			),
		)

		It(
			"Should broadcast the create with the writer's dispatch key after commit",
			func(ctx SpecContext) {
				tx := DeferClose(db.OpenTx())
				rec := &Recorder[table.Key, table.Action]{}
				DeferCleanup(svc.OnAction(rec.Record))
				v := table.Table{Project: &proj.Key, Name: "broadcast"}
				Expect(
					svc.NewWriter(tx).WithDispatchKey("dk-1").Create(ctx, &v),
				).To(Succeed())
				Expect(rec.Snapshot()).To(BeEmpty())
				Expect(tx.Commit(ctx)).To(Succeed())
				seen := rec.Snapshot()
				Expect(seen).To(HaveLen(1))
				Expect(seen[0].Key).To(Equal(v.Key))
				Expect(seen[0].DispatchKey).To(Equal("dk-1"))
				Expect(seen[0].Actions).To(HaveLen(1))
				Expect(seen[0].Actions[0].Type).To(Equal(table.ActionTypeCreate))
			},
		)

		It(
			"Should not broadcast a create whose transaction closes uncommitted",
			func(ctx SpecContext) {
				rec := &Recorder[table.Key, table.Action]{}
				DeferCleanup(svc.OnAction(rec.Record))
				uncommitted := db.OpenTx()
				v := table.Table{Project: &proj.Key, Name: "discarded"}
				Expect(
					svc.NewWriter(uncommitted).Create(ctx, &v),
				).To(Succeed())
				Expect(uncommitted.Close()).To(Succeed())
				Expect(rec.Snapshot()).To(BeEmpty())
			},
		)
	})

	Describe("CreateMany", func() {
		It("Should create multiple tables", func(ctx SpecContext) {
			tx := DeferClose(db.OpenTx())
			tables := []table.Table{
				{Name: "table-1", Project: &proj.Key},
				{Name: "table-2", Project: &proj.Key},
			}
			Expect(svc.NewWriter(tx).CreateMany(ctx, &tables)).To(Succeed())

			var retrieved []table.Table
			Expect(svc.NewRetrieve().Where(table.MatchKeys(
				tables[0].Key,
				tables[1].Key,
			)).Entries(&retrieved).Exec(ctx, tx)).To(Succeed())
			Expect(retrieved).To(HaveLen(2))
		})
	})

	Describe("Apply of a project move", func() {
		It("Should move the table into the project", func(ctx SpecContext) {
			tx := DeferClose(db.OpenTx())
			dest := project.Project{Name: "destination"}
			Expect(projectSvc.NewWriter(tx).Create(ctx, &dest)).To(Succeed())
			v := table.Table{Project: &proj.Key, Name: "moved"}
			Expect(svc.NewWriter(tx).Create(ctx, &v)).To(Succeed())
			Expect(svc.NewWriter(tx).Apply(ctx, v.Key, setProject(dest.Key))).
				To(BeEmpty())
			var res table.Table
			Expect(svc.NewRetrieve().
				Where(table.MatchKeys(v.Key)).
				Entry(&res).
				Exec(ctx, tx)).To(Succeed())
			Expect(res.Project).To(HaveValue(Equal(dest.Key)))
		})

		It("Should reject a project that does not exist", func(ctx SpecContext) {
			tx := DeferClose(db.OpenTx())
			v := table.Table{Project: &proj.Key, Name: "stuck"}
			Expect(svc.NewWriter(tx).Create(ctx, &v)).To(Succeed())
			Expect(svc.NewWriter(tx).Apply(ctx, v.Key, setProject(uuid.New()))).
				Error().To(MatchError(ContainSubstring("does not exist")))
		})
	})

	Describe("Dispatch", func() {
		It(
			"Should reject an action that leaves the table invalid",
			func(ctx SpecContext) {
				s := table.Table{Project: &proj.Key, Name: "test"}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &s)
				})).To(Succeed())
				Expect(dispatch(ctx, s.Key, "dk-1", []table.Action{
					table.NewSetAction(
						table.SetPayload{Path: []string{"name"}, Value: ""},
					),
				})).To(MatchError(ContainSubstring("name: required")))
				Expect(retrieve(ctx, s.Key).Name).To(Equal("test"))
			},
		)

		It("Should reject an action with no payload", func(ctx SpecContext) {
			s := seed(ctx)
			Expect(dispatch(ctx, s.Key, "dk-1", []table.Action{
				{Type: table.ActionTypeSet},
			})).To(MatchError(union.ErrMissingPayload))
			Expect(retrieve(ctx, s.Key).Name).To(Equal("test"))
		})

		It("Should reject an action that duplicates a row key", func(ctx SpecContext) {
			s := seed(ctx)
			Expect(dispatch(ctx, s.Key, "dk-1", []table.Action{
				table.NewAddRowAction(
					table.AddRowPayload{Index: 1, Key: "r0", Size: 30},
				),
			})).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(`duplicate row key "r0"`)),
			))
			Expect(retrieve(ctx, s.Key).Rows).To(HaveLen(1))
		})

		It(
			"Should reject a create without changing the table it names",
			func(ctx SpecContext) {
				target := table.Table{Project: &proj.Key, Name: "target"}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &target)
				})).To(Succeed())
				other := table.Table{Project: &proj.Key, Name: "other"}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &other)
				})).To(Succeed())
				Expect(dispatch(ctx, target.Key, "d1", []table.Action{
					table.NewCreateAction(table.CreatePayload{
						Table: table.Table{Key: other.Key, Name: "overwritten"},
					}),
				})).To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(
						ContainSubstring("only the Core dispatches a table create"),
					),
				))
				Expect(retrieve(ctx, other.Key).Name).To(Equal("other"))
			},
		)

		It("Should rename a Table via a Rename action", func(ctx SpecContext) {
			s := table.Table{Project: &proj.Key, Name: "test"}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &s)
			})).To(Succeed())
			Expect(dispatch(ctx, s.Key, "dk-1", []table.Action{
				table.NewSetAction(
					table.SetPayload{Path: []string{"name"}, Value: "test2"},
				),
			})).To(Succeed())
			Expect(retrieve(ctx, s.Key).Name).To(Equal("test2"))
		})

		It(
			"Should apply a multi-action sequence atomically and persist the result",
			func(ctx SpecContext) {
				s := seed(ctx)
				Expect(dispatch(ctx, s.Key, "dk-1", []table.Action{
					table.NewSetAction(
						table.SetPayload{Path: []string{"name"}, Value: "multi"},
					),
					table.NewAddRowAction(table.AddRowPayload{
						Index: 1,
						Key:   "r1",
						Size:  40,
						Cells: []table.Cell{
							cell("c", textCfg("")),
							cell("d", textCfg("")),
						},
					}),
					table.NewSetCellAction(table.SetCellPayload{
						Cell: cell("c", valueCfg("psi")),
					}),
				})).To(Succeed())
				res := retrieve(ctx, s.Key)
				Expect(res.Name).To(Equal("multi"))
				Expect(res.Rows).To(HaveLen(2))
				Expect(res.Cells["c"]).To(Equal(valueCfg("psi")))
			},
		)

		It(
			"Should notify the action observer once per Dispatch with monotonic seq",
			func(ctx SpecContext) {
				s := seed(ctx)
				rec := &Recorder[table.Key, table.Action]{}
				DeferCleanup(svc.OnAction(rec.Record))
				Expect(dispatch(ctx, s.Key, "dk-1", []table.Action{
					table.NewSetAction(
						table.SetPayload{Path: []string{"name"}, Value: "first"},
					),
				})).To(Succeed())
				Expect(dispatch(ctx, s.Key, "dk-2", []table.Action{
					table.NewSetAction(
						table.SetPayload{Path: []string{"name"}, Value: "second"},
					),
				})).To(Succeed())
				received := rec.Snapshot()
				Expect(received).To(HaveLen(2))
				Expect(received[0].DispatchKey).To(Equal("dk-1"))
				Expect(received[1].DispatchKey).To(Equal("dk-2"))
				Expect(received[1].Seq > received[0].Seq).To(BeTrue())
			},
		)
	})
})
