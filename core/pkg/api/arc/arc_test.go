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
	"time"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/gorp"
	apiarc "github.com/synnaxlabs/synnax/pkg/api/arc"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	rbactestutil "github.com/synnaxlabs/synnax/pkg/service/access/rbac/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	. "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/arc/directory"
	arctask "github.com/synnaxlabs/synnax/pkg/service/arc/task"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/filter"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/spatial"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// scopedAction is a short local alias for the Arc action envelope.
type scopedAction = actions.Scoped[arc.Key, arc.Action]

// createArc persists a fresh Arc and returns it with its key populated. The write
// commits at once so access-control reads can observe the new Arc.
func createArc(ctx context.Context, name string) arc.Arc {
	GinkgoHelper()
	a := arc.Arc{Variant: arc.AutomationArc{
		Base: arc.Base{Name: name, Source: arc.Source{Variant: arc.GraphSource{}}},
	}}
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return arcSvc.NewWriter(tx).Create(ctx, &a)
	})).To(Succeed())
	return a
}

// setRunning writes the status that the task t reports when it starts or stops.
func setRunning(ctx context.Context, t task.Task, running bool) {
	GinkgoHelper()
	stat := task.Status{
		Owner:   new(t.ResourceID()),
		Variant: status.VariantSuccess,
		Message: "Task status",
		Time:    telem.Now(),
		Details: task.StatusDetails{Running: running},
	}
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return svc.Status.NewWriter(tx).Set(ctx, &stat)
	})).To(Succeed())
}

// deploy binds a to the test rack as the Core, with no access check, and returns the
// task of a.
func deploy(ctx context.Context, a arc.Arc) task.Task {
	GinkgoHelper()
	return deployOn(ctx, a, testRack.Key)
}

// deployRunning binds a to a new rack, grants the author update on that rack, and marks
// the task of a as running. A new rack keeps the rack monitor from marking the task as
// not running.
func deployRunning(ctx SpecContext, a arc.Arc) task.Task {
	GinkgoHelper()
	r := rack.Rack{Name: "Driver"}
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return rackSvc.NewWriter(tx).Create(ctx, &r)
	})).To(Succeed())
	grantUpdateOn(ctx, author.ResourceID(), r.ResourceID())
	t := deployOn(ctx, a, r.Key)
	setRunning(ctx, t, true)
	return t
}

func deployOn(ctx context.Context, a arc.Arc, r rack.Key) task.Task {
	GinkgoHelper()
	var tsk *task.Task
	Expect(db.WithTx(ctx, func(tx gorp.Tx) (err error) {
		tsk, err = arcSvc.NewWriter(tx).UpdateTask(
			ctx,
			a.Base().Key,
			arc.TaskUpdate{Rack: new(r)},
		)
		return err
	})).To(Succeed())
	return *tsk
}

var _ = Describe("Service", func() {
	Describe("Retrieve", func() {
		It("Should leave out the automations that the subject cannot retrieve", func(
			ctx SpecContext,
		) {
			u := MustSucceed(rbactestutil.CreateUser(
				ctx,
				db,
				rbacSvc,
				userSvc,
				user.User{Name: "leave-out"},
			))
			granted := createArc(ctx, "leave-out-granted")
			createArc(ctx, "leave-out-denied")
			grantOn(ctx, u.ResourceID(), access.ActionRetrieve, granted.ResourceID())
			res := MustSucceed(retrieve(AuthedCtx(ctx, u), apiarc.RetrieveRequest{}))
			Expect(res.Arcs).To(HaveExactElements(WithTransform(
				func(a arc.Arc) arc.Key { return a.Base().Key },
				Equal(granted.Base().Key),
			)))
		})

		It("Should include a module's compile status", func(ctx SpecContext) {
			module := arc.Arc{Variant: arc.ModuleArc{Base: arc.Base{
				Name: "status_m",
				Source: arc.Source{Variant: arc.TextSource{
					Text: text.Text{Raw: "export func f() i64 { return 1 }"},
				}},
			}}}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return arcSvc.NewWriter(tx).Create(ctx, &module)
			})).To(Succeed())
			DeferCleanup(func(ctx SpecContext) {
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return arcSvc.NewWriter(tx).Delete(ctx, module.Base().Key)
				})).To(Succeed())
			})
			grantOn(
				ctx,
				author.ResourceID(),
				access.ActionRetrieve,
				module.ResourceID(),
			)
			req := apiarc.RetrieveRequest{
				Where: &arc.Where{
					Key: &filter.Set[arc.Key]{In: []arc.Key{module.Base().Key}},
				},
			}
			res := MustSucceed(retrieve(AuthedCtx(ctx, author), req))
			Expect(res.Arcs[0].Variant.(arc.ModuleArc).Status).To(BeNil())
			req.IncludeStatus = true
			Eventually(func(g Gomega) {
				res, err := retrieve(AuthedCtx(ctx, author), req)
				g.Expect(err).ToNot(HaveOccurred())
				stat := res.Arcs[0].Variant.(arc.ModuleArc).Status
				g.Expect(stat).ToNot(BeNil())
				g.Expect(stat.Message).To(Equal("Compiled"))
			}).WithTimeout(5 * time.Second).Should(Succeed())
		})

		It("Should include what changed in a deployed automation", func(
			ctx SpecContext,
		) {
			a := createArc(ctx, "changes_a")
			grantOn(ctx, author.ResourceID(), access.ActionRetrieve, a.ResourceID())
			req := apiarc.RetrieveRequest{
				Where: &arc.Where{
					Key: &filter.Set[arc.Key]{In: []arc.Key{a.Base().Key}},
				},
				IncludeChanges: true,
			}
			tsk := deploy(ctx, a)
			res := MustSucceed(retrieve(AuthedCtx(ctx, author), req))
			Expect(res.Arcs[0].Variant.(arc.AutomationArc).Changes).To(BeEmpty())
			stat := &task.Status{
				Owner:   new(task.ResourceID(tsk.Key)),
				Variant: status.VariantSuccess,
				Time:    telem.Now(),
				Details: task.StatusDetails{Running: true, ConfigHash: tsk.ConfigHash},
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return statusSvc.NewWriter(tx).Set(ctx, stat)
			})).To(Succeed())
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return arcSvc.NewWriter(tx).Apply(
					ctx,
					a.Base().Key,
					arc.NewSetNodeAction(arc.SetNodePayload{Node: graph.Node{Key: "n1"}}),
				)
			})).To(Succeed())
			Eventually(func(g Gomega) {
				res, err := retrieve(AuthedCtx(ctx, author), req)
				g.Expect(err).ToNot(HaveOccurred())
				g.Expect(res.Arcs[0].Variant.(arc.AutomationArc).Changes).To(Equal(
					[]arc.Change{
						{Type: arc.TypeAutomation, Kind: arc.ChangeKindChanged},
					},
				))
			}).Should(Succeed())
			req.IncludeChanges = false
			res = MustSucceed(retrieve(AuthedCtx(ctx, author), req))
			Expect(res.Arcs[0].Variant.(arc.AutomationArc).Changes).To(BeNil())
		})

		It(
			"Should compile the graph of a graph-mode automation",
			func(ctx SpecContext) {
				a := arc.Arc{Variant: arc.AutomationArc{Base: arc.Base{
					Name: "compiled_graph",
					Source: arc.Source{Variant: arc.GraphSource{Graph: graph.Graph{
						Functions: []ir.Function{{
							Key: "add",
							Inputs: types.Params{
								{Name: "a", Type: types.I64()},
								{Name: "b", Type: types.I64()},
							},
							Outputs: types.Params{
								{Name: ir.DefaultOutputParam, Type: types.I64()},
							},
							Body: ir.Body{Raw: "{ return a + b }"},
						}},
					}}},
				}}}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return arcSvc.NewWriter(tx).Create(ctx, &a)
				})).To(Succeed())
				DeferCleanup(func(ctx SpecContext) {
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return arcSvc.NewWriter(tx).Delete(ctx, a.Base().Key)
					})).To(Succeed())
				})
				grantOn(ctx, author.ResourceID(), access.ActionRetrieve, a.ResourceID())
				res := MustSucceed(
					retrieve(AuthedCtx(ctx, author), apiarc.RetrieveRequest{
						Where: &arc.Where{
							Key: &filter.Set[arc.Key]{In: []arc.Key{a.Base().Key}},
						},
						IncludeProgram: true,
					}),
				)
				Expect(res.Arcs).To(HaveLen(1))
				automation, ok := res.Arcs[0].Variant.(arc.AutomationArc)
				Expect(ok).To(BeTrue())
				Expect(automation.Program).ToNot(BeNil())
				Expect(automation.Program.Functions).To(HaveLen(1))
			},
		)

		It(
			"Should return an automation's program with its module code",
			func(ctx SpecContext) {
				module := arc.Arc{Variant: arc.ModuleArc{Base: arc.Base{
					Name: "um_api_m",
					Source: arc.Source{Variant: arc.TextSource{
						Text: text.Text{Raw: "export func f() i64 { return 1 }"},
					}},
				}}}
				automation := arc.Arc{Variant: arc.AutomationArc{Base: arc.Base{
					Name: "um_api_a",
					Source: arc.Source{Variant: arc.TextSource{
						Text: text.Text{
							Raw: "import um_api_m\n\nfunc g() i64 { return um_api_m.f() }",
						},
					}},
				}}}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					w := arcSvc.NewWriter(tx)
					if err := w.Create(ctx, &module); err != nil {
						return err
					}
					return w.Create(ctx, &automation)
				})).To(Succeed())
				DeferCleanup(func(ctx SpecContext) {
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return arcSvc.NewWriter(tx).Delete(
							ctx, automation.Base().Key, module.Base().Key,
						)
					})).To(Succeed())
				})
				grantOn(
					ctx,
					author.ResourceID(),
					access.ActionRetrieve,
					automation.ResourceID(),
				)
				res := MustSucceed(
					retrieve(AuthedCtx(ctx, author), apiarc.RetrieveRequest{
						Where: &arc.Where{
							Key: &filter.Set[arc.Key]{
								In: []arc.Key{automation.Base().Key},
							},
						},
						IncludeProgram: true,
					}),
				)
				Expect(res.Arcs).To(HaveLen(1))
				a, ok := res.Arcs[0].Variant.(arc.AutomationArc)
				Expect(ok).To(BeTrue())
				_, found := a.Program.Functions.Find("um_api_m.f")
				Expect(found).To(BeTrue())
				Expect(a.Program.WASM).ToNot(BeEmpty())
			},
		)

		It(
			"Should compile automations and leave modules uncompiled",
			func(ctx SpecContext) {
				automation := arc.Arc{Variant: arc.AutomationArc{Base: arc.Base{
					Name: "compiled",
					Source: arc.Source{
						Variant: arc.TextSource{Text: text.Text{Raw: "func f() {}"}},
					},
				}}}
				module := arc.Arc{Variant: arc.ModuleArc{Base: arc.Base{
					Name: "not_compiled",
					Source: arc.Source{
						Variant: arc.TextSource{
							Text: text.Text{Raw: "this does not compile"},
						},
					},
				}}}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return arcSvc.NewWriter(tx).Create(ctx, &automation)
				})).To(Succeed())
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return arcSvc.NewWriter(tx).Create(ctx, &module)
				})).To(Succeed())
				DeferCleanup(func(ctx SpecContext) {
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return arcSvc.NewWriter(tx).Delete(
							ctx, automation.Base().Key, module.Base().Key,
						)
					})).To(Succeed())
				})
				grantOn(
					ctx,
					author.ResourceID(),
					access.ActionRetrieve,
					automation.ResourceID(),
					module.ResourceID(),
				)
				res := MustSucceed(
					retrieve(AuthedCtx(ctx, author), apiarc.RetrieveRequest{
						Where: &arc.Where{
							Key: &filter.Set[arc.Key]{
								In: []arc.Key{automation.Base().Key, module.Base().Key},
							},
						},
						IncludeProgram: true,
					}),
				)
				Expect(res.Arcs).To(HaveLen(2))
				for _, a := range res.Arcs {
					switch v := a.Variant.(type) {
					case arc.AutomationArc:
						Expect(v.Program).ToNot(BeNil())
					case arc.ModuleArc:
						Expect(v.Name).To(Equal("not_compiled"))
					default:
						Fail("unexpected variant")
					}
				}
			},
		)

		It(
			"Should omit missing keys from a multi-key retrieve instead of failing",
			func(ctx SpecContext) {
				a := createArc(ctx, "partial-survivor")
				missing := uuid.New()
				grantOn(
					ctx,
					author.ResourceID(),
					access.ActionRetrieve,
					arc.ResourceID(a.Base().Key),
					arc.ResourceID(missing),
				)
				res := MustSucceed(
					retrieve(AuthedCtx(ctx, author), apiarc.RetrieveRequest{
						Where: &arc.Where{
							Key: &filter.Set[arc.Key]{
								In: []arc.Key{a.Base().Key, missing},
							},
						},
					}),
				)
				Expect(res.Arcs).To(HaveLen(1))
				Expect(res.Arcs[0].Base().Key).To(Equal(a.Base().Key))
			},
		)

		It(
			"Should return an empty result when every requested key is missing",
			func(ctx SpecContext) {
				missing := uuid.New()
				grantOn(
					ctx,
					author.ResourceID(),
					access.ActionRetrieve,
					arc.ResourceID(missing),
				)
				res := MustSucceed(
					retrieve(AuthedCtx(ctx, author), apiarc.RetrieveRequest{
						Where: &arc.Where{
							Key: &filter.Set[arc.Key]{In: []arc.Key{missing}},
						},
					}),
				)
				Expect(res.Arcs).To(BeEmpty())
			},
		)

		It(
			"Should deny the retrieve before compiling the program",
			func(ctx SpecContext) {
				a := arc.Arc{Variant: arc.AutomationArc{Base: arc.Base{
					Name: "uncompilable",
					Source: arc.Source{Variant: arc.TextSource{
						Text: text.Text{Doc: text.Create("a ->")},
					}},
				}}}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return arcSvc.NewWriter(tx).Create(ctx, &a)
				})).To(Succeed())
				key := a.Base().Key
				Expect(retrieve(AuthedCtx(ctx, author), apiarc.RetrieveRequest{
					Where:          &arc.Where{Key: &filter.Set[arc.Key]{Eq: &key}},
					IncludeProgram: true,
				})).Error().To(MatchError(access.ErrDenied))
			},
		)
	})

	Describe("Create", func() {
		newArc := func(name string) apiarc.Arc {
			return apiarc.Arc{Variant: arc.AutomationArc{Base: arc.Base{
				Key:    uuid.New(),
				Name:   name,
				Source: arc.Source{Variant: arc.GraphSource{}},
			}}}
		}
		// grantCreate grants what a create of a new Arc needs: create on the Arc and
		// update on the root directory, which the Arc sits in.
		grantCreate := func(ctx SpecContext) {
			GinkgoHelper()
			grantOn(
				ctx,
				author.ResourceID(),
				access.ActionCreate,
				resource.ID{Type: resource.TypeArc},
			)
			grantUpdateOn(ctx, author.ResourceID(), directory.ResourceID(root))
		}

		It(
			"Should materialize the stored document, not echo a client-supplied Raw",
			func(ctx SpecContext) {
				grantCreate(ctx)
				a := apiarc.Arc{Variant: arc.AutomationArc{Base: arc.Base{
					Name: "lying-raw",
					Source: arc.Source{
						Variant: arc.TextSource{
							Text: text.Text{Doc: text.Create("a -> b"), Raw: "a -> c"},
						},
					},
				}}}
				res := MustSucceed(
					create(AuthedCtx(ctx, author), apiarc.CreateRequest{
						Arcs: []apiarc.Arc{a},
					}),
				)
				Expect(textOf(res.Arcs[0].Base()).Raw).To(Equal("a -> b"))
			},
		)

		It("Should stamp the dispatch key onto the create broadcast", func(
			ctx SpecContext,
		) {
			grantCreate(ctx)
			rec := &Recorder[arc.Key, arc.Action]{}
			DeferCleanup(arcSvc.OnAction(rec.Record))
			res := MustSucceed(
				create(AuthedCtx(ctx, author), apiarc.CreateRequest{
					Arcs:        []apiarc.Arc{newArc("broadcast")},
					DispatchKey: "dk-create",
				}),
			)
			frames := rec.Snapshot()
			Expect(frames).To(HaveLen(1))
			Expect(frames[0].Key).To(Equal(res.Arcs[0].Base().Key))
			Expect(frames[0].DispatchKey).To(Equal("dk-create"))
		})

		It("Should deny a caller without create access", func(ctx SpecContext) {
			grantUpdateOn(ctx, author.ResourceID(), directory.ResourceID(root))
			a := newArc("create-denied")
			Expect(create(AuthedCtx(ctx, author), apiarc.CreateRequest{
				Arcs: []apiarc.Arc{a},
			})).Error().To(denied(author, access.ActionCreate, a.ResourceID()))
			Expect(arcSvc.NewRetrieve().
				Where(arc.MatchKeys(a.Base().Key)).
				Exists(ctx, db)).To(BeFalse())
		})

		It(
			"Should deny a caller without update access to the root directory",
			func(ctx SpecContext) {
				grantOn(
					ctx,
					author.ResourceID(),
					access.ActionCreate,
					resource.ID{Type: resource.TypeArc},
				)
				a := newArc("create-no-root")
				Expect(create(AuthedCtx(ctx, author), apiarc.CreateRequest{
					Arcs: []apiarc.Arc{a},
				})).Error().To(
					denied(author, access.ActionUpdate, directory.ResourceID(root)),
				)
				Expect(arcSvc.NewRetrieve().
					Where(arc.MatchKeys(a.Base().Key)).
					Exists(ctx, db)).To(BeFalse())
			},
		)
	})

	Describe("Delete", func() {
		It(
			"Should delete an Arc with delete access and update on its parent",
			func(ctx SpecContext) {
				a := createArc(ctx, "delete-ok")
				grantOn(ctx, author.ResourceID(), access.ActionDelete, a.ResourceID())
				grantUpdateOn(ctx, author.ResourceID(), directory.ResourceID(root))
				MustSucceed(del(AuthedCtx(ctx, author), apiarc.DeleteRequest{
					Keys: []arc.Key{a.Base().Key},
				}))
				Expect(arcSvc.NewRetrieve().
					Where(arc.MatchKeys(a.Base().Key)).
					Exists(ctx, db)).To(BeFalse())
			},
		)

		It("Should deny a caller without delete access", func(ctx SpecContext) {
			a := createArc(ctx, "delete-denied")
			grantUpdateOn(ctx, author.ResourceID(), directory.ResourceID(root))
			Expect(del(AuthedCtx(ctx, author), apiarc.DeleteRequest{
				Keys: []arc.Key{a.Base().Key},
			})).Error().To(denied(author, access.ActionDelete, a.ResourceID()))
			Expect(arcSvc.NewRetrieve().
				Where(arc.MatchKeys(a.Base().Key)).
				Exists(ctx, db)).To(BeTrue())
		})

		It(
			"Should deny a caller without update access to the directory of the Arc",
			func(ctx SpecContext) {
				a := createArc(ctx, "delete-no-root")
				grantOn(ctx, author.ResourceID(), access.ActionDelete, a.ResourceID())
				Expect(del(AuthedCtx(ctx, author), apiarc.DeleteRequest{
					Keys: []arc.Key{a.Base().Key},
				})).Error().To(
					denied(author, access.ActionUpdate, directory.ResourceID(root)),
				)
				Expect(arcSvc.NewRetrieve().
					Where(arc.MatchKeys(a.Base().Key)).
					Exists(ctx, db)).To(BeTrue())
			},
		)

		Describe("an Arc with a task", func() {
			// grantAllButTaskDelete grants what deleting a deployed Arc needs except
			// delete on its task: delete on the Arc and the task's status, and update
			// on the directory, the task that holds the status, and the rack.
			grantAllButTaskDelete := func(ctx SpecContext, a arc.Arc) {
				GinkgoHelper()
				grantOn(
					ctx,
					author.ResourceID(),
					access.ActionDelete,
					a.ResourceID(),
					resource.ID{Type: resource.TypeStatus},
				)
				grantUpdateOn(
					ctx,
					author.ResourceID(),
					directory.ResourceID(root),
					resource.ID{Type: resource.TypeTask},
					testRack.ResourceID(),
				)
			}

			It("Should deny a caller without delete access to the task", func(
				ctx SpecContext,
			) {
				a := createArc(ctx, "delete-deployed-denied")
				tsk := deploy(ctx, a)
				grantAllButTaskDelete(ctx, a)
				Expect(del(AuthedCtx(ctx, author), apiarc.DeleteRequest{
					Keys: []arc.Key{a.Base().Key},
				})).Error().To(
					denied(author, access.ActionDelete, task.ResourceID(tsk.Key)),
				)
				Expect(taskSvc.NewRetrieve().
					Where(task.MatchKeys(tsk.Key)).
					Exists(ctx, db)).To(BeTrue())
			})

			It("Should delete the Arc and its task with delete access to both", func(
				ctx SpecContext,
			) {
				a := createArc(ctx, "delete-deployed-ok")
				tsk := deploy(ctx, a)
				grantAllButTaskDelete(ctx, a)
				grantOn(
					ctx,
					author.ResourceID(),
					access.ActionDelete,
					task.ResourceID(tsk.Key),
				)
				MustSucceed(del(AuthedCtx(ctx, author), apiarc.DeleteRequest{
					Keys: []arc.Key{a.Base().Key},
				}))
				Expect(arcSvc.NewRetrieve().
					Where(arc.MatchKeys(a.Base().Key)).
					Exists(ctx, db)).To(BeFalse())
				Expect(taskSvc.NewRetrieve().
					Where(task.MatchKeys(tsk.Key)).
					Exists(ctx, db)).To(BeFalse())
			})

			It("Should not stop a running task without delete access to it", func(
				ctx SpecContext,
			) {
				a := createArc(ctx, "delete-running-denied")
				tsk := deployRunning(ctx, a)
				grantAllButTaskDelete(ctx, a)
				start := time.Now()
				Expect(del(AuthedCtx(ctx, author), apiarc.DeleteRequest{
					Keys: []arc.Key{a.Base().Key},
				})).Error().To(
					denied(author, access.ActionDelete, task.ResourceID(tsk.Key)),
				)
				Expect(time.Since(start)).To(BeNumerically("<", time.Second))
			})

			It("Should take no write lock while it waits for a task to stop", func(
				ctx SpecContext,
			) {
				a := createArc(ctx, "delete-running-ok")
				tsk := deployRunning(ctx, a)
				grantAllButTaskDelete(ctx, a)
				grantOn(
					ctx,
					author.ResourceID(),
					access.ActionDelete,
					task.ResourceID(tsk.Key),
				)
				deleted := make(chan error, 1)
				start := time.Now()
				go func() {
					_, err := del(AuthedCtx(ctx, author), apiarc.DeleteRequest{
						Keys: []arc.Key{a.Base().Key},
					})
					deleted <- err
				}()
				time.Sleep(100 * time.Millisecond)
				setRunning(ctx, tsk, false)
				Expect(time.Since(start)).To(BeNumerically("<", time.Second))
				Eventually(deleted).Should(Receive(Not(HaveOccurred())))
				Expect(time.Since(start)).To(BeNumerically("<", 2*time.Second))
				Expect(arcSvc.NewRetrieve().
					Where(arc.MatchKeys(a.Base().Key)).
					Exists(ctx, db)).To(BeFalse())
			})

			It("Should keep the Arc when the caller cancels", func(ctx SpecContext) {
				a := createArc(ctx, "delete-running-canceled")
				tsk := deployRunning(ctx, a)
				grantAllButTaskDelete(ctx, a)
				grantOn(
					ctx,
					author.ResourceID(),
					access.ActionDelete,
					task.ResourceID(tsk.Key),
				)
				reqCtx, cancel := context.WithCancel(ctx)
				defer time.AfterFunc(50*time.Millisecond, cancel).Stop()
				Expect(del(AuthedCtx(reqCtx, author), apiarc.DeleteRequest{
					Keys: []arc.Key{a.Base().Key},
				})).Error().To(MatchError(context.Canceled))
				Expect(arcSvc.NewRetrieve().
					Where(arc.MatchKeys(a.Base().Key)).
					Exists(ctx, db)).To(BeTrue())
			})
		})
	})

	Describe("Dispatch", func() {
		Describe("access control", func() {
			It(
				"Should reject the request with access.ErrDenied when the subject has no policy",
				func(ctx SpecContext) {
					a := createArc(ctx, "no-policy")
					Expect(dispatch(
						AuthedCtx(ctx, author), apiarc.DispatchRequest{
							Key:         a.Base().Key,
							DispatchKey: "sess-1",
							Actions: []arc.Action{
								arc.NewRemoveNodeAction(arc.RemoveNodePayload{
									Key: "n1",
								}),
							},
						},
					)).Error().To(MatchError(access.ErrDenied))
				},
			)

			It(
				"Should accept the request when the subject's policy permits update on the target Arc",
				func(ctx SpecContext) {
					a := createArc(ctx, "with-policy")
					grantUpdateOn(ctx, author.ResourceID(), a.ResourceID())
					Expect(dispatch(
						AuthedCtx(ctx, author), apiarc.DispatchRequest{
							Key:         a.Base().Key,
							DispatchKey: "sess-1",
							Actions: []arc.Action{
								arc.NewSetNodeAction(arc.SetNodePayload{
									Node: graph.Node{
										Key:      "n1",
										Position: spatial.XY{X: 1, Y: 2},
									},
								}),
							},
						},
					)).Error().To(Succeed())
					var res arc.Arc
					Expect(arcSvc.NewRetrieve().
						Where(arc.MatchKeys(a.Base().Key)).
						Entry(&res).Exec(ctx, db)).To(Succeed())
					Expect(graphOf(res.Base()).Nodes).To(HaveLen(1))
					Expect(
						graphOf(res.Base()).Nodes[0].Position,
					).To(Equal(spatial.XY{X: 1, Y: 2}))
				},
			)

			It(
				"Should reject when the subject's policy targets a different Arc",
				func(ctx SpecContext) {
					a := createArc(ctx, "policy-target")
					b := createArc(ctx, "no-policy-target")
					grantUpdateOn(ctx, author.ResourceID(), a.ResourceID())
					Expect(dispatch(
						AuthedCtx(ctx, author), apiarc.DispatchRequest{
							Key:         b.Base().Key,
							DispatchKey: "sess-1",
							Actions: []arc.Action{
								arc.NewRemoveNodeAction(arc.RemoveNodePayload{
									Key: "n1",
								}),
							},
						},
					)).Error().To(MatchError(access.ErrDenied))
				},
			)
		})

		Describe("delegation to Writer.Apply", func() {
			It(
				"Should apply a multi-action sequence to the target Arc",
				func(ctx SpecContext) {
					a := createArc(ctx, "multi-action")
					grantUpdateOn(ctx, author.ResourceID(), a.ResourceID())
					Expect(dispatch(
						AuthedCtx(ctx, author), apiarc.DispatchRequest{
							Key:         a.Base().Key,
							DispatchKey: "sess-1",
							Actions: []arc.Action{
								arc.NewSetNodeAction(
									arc.SetNodePayload{Node: graph.Node{Key: "n1"}},
								),
								arc.NewSetNodeAction(
									arc.SetNodePayload{Node: graph.Node{Key: "n2"}},
								),
								arc.NewSetAction(
									arc.SetPayload{
										Path:  []string{"name"},
										Value: "renamed-multi",
									},
								),
							},
						},
					)).Error().To(Succeed())
					var res arc.Arc
					Expect(arcSvc.NewRetrieve().
						Where(arc.MatchKeys(a.Base().Key)).
						Entry(&res).Exec(ctx, db)).To(Succeed())
					Expect(graphOf(res.Base()).Nodes).To(HaveLen(2))
					Expect(res.Base().Name).To(Equal("renamed-multi"))
				},
			)

			It(
				"Should bubble up query.ErrNotFound when the target Arc does not exist",
				func(ctx SpecContext) {
					missing := uuid.New()
					grantUpdateOn(ctx, author.ResourceID(), arc.ResourceID(missing))
					Expect(dispatch(
						AuthedCtx(ctx, author), apiarc.DispatchRequest{
							Key:         missing,
							DispatchKey: "sess-1",
							Actions: []arc.Action{
								arc.NewSetAction(
									arc.SetPayload{
										Path:  []string{"name"},
										Value: "ghost",
									},
								),
							},
						},
					)).Error().To(MatchError(query.ErrNotFound))
				},
			)
		})

		Describe("subject identity propagation", func() {
			It(
				"Should pass the DispatchKey verbatim into the action observer",
				func(ctx SpecContext) {
					a := createArc(ctx, "session-propagation")
					grantUpdateOn(ctx, author.ResourceID(), a.ResourceID())
					seen := make(chan scopedAction, 1)
					DeferCleanup(
						arcSvc.OnAction(func(_ context.Context, sa scopedAction) {
							seen <- sa
						}),
					)
					Expect(dispatch(
						AuthedCtx(ctx, author), apiarc.DispatchRequest{
							Key:         a.Base().Key,
							DispatchKey: "session-marker-xyz",
							Actions: []arc.Action{
								arc.NewSetNodeAction(arc.SetNodePayload{
									Node: graph.Node{Key: "n1"},
								}),
							},
						},
					)).Error().To(Succeed())
					var got scopedAction
					Eventually(seen).Should(Receive(&got))
					Expect(got.Key).To(Equal(a.Base().Key))
					Expect(got.DispatchKey).To(Equal("session-marker-xyz"))
					Expect(got.Seq).To(BeNumerically(">", uint64(0)))
					Expect(got.Actions).To(HaveLen(1))
				},
			)
		})
	})

	Describe("UpdateTask", func() {
		// grantTask grants what a bind needs other than retrieve on the arc: create on
		// the task and its status, and update on the rack, which the task sits in.
		grantTask := func(ctx SpecContext) {
			GinkgoHelper()
			grantOn(
				ctx,
				author.ResourceID(),
				access.ActionCreate,
				resource.ID{Type: resource.TypeTask},
				resource.ID{Type: resource.TypeStatus},
			)
			grantUpdateOn(ctx, author.ResourceID(), testRack.ResourceID())
		}
		bind := func(ctx SpecContext, a arc.Arc) (apiarc.UpdateTaskResponse, error) {
			return updateTask(AuthedCtx(ctx, author), apiarc.UpdateTaskRequest{
				Key:  a.Base().Key,
				Rack: new(testRack.Key),
			})
		}

		It(
			"Should reject the request when the subject has no policy",
			func(ctx SpecContext) {
				a := createArc(ctx, "set-rack-no-policy")
				Expect(bind(ctx, a)).Error().To(MatchError(access.ErrDenied))
			},
		)

		It(
			"Should reject when the subject may update the arc but not create tasks",
			func(ctx SpecContext) {
				a := createArc(ctx, "set-rack-no-task-policy")
				grantUpdateOn(ctx, author.ResourceID(), a.ResourceID())
				Expect(bind(ctx, a)).Error().To(MatchError(access.ErrDenied))
			},
		)

		It(
			"Should reject a bind without retrieve access to the arc",
			func(ctx SpecContext) {
				a := createArc(ctx, "set-rack-no-arc-retrieve")
				grantTask(ctx)
				Expect(bind(ctx, a)).Error().
					To(denied(author, access.ActionRetrieve, a.ResourceID()))
			},
		)

		It(
			"Should reject a bind without update access to the rack",
			func(ctx SpecContext) {
				a := createArc(ctx, "set-rack-no-rack-update")
				grantOn(ctx, author.ResourceID(), access.ActionRetrieve, a.ResourceID())
				grantOn(
					ctx,
					author.ResourceID(),
					access.ActionCreate,
					resource.ID{Type: resource.TypeTask},
					resource.ID{Type: resource.TypeStatus},
				)
				Expect(bind(ctx, a)).Error().
					To(denied(author, access.ActionUpdate, testRack.ResourceID()))
			},
		)

		It("Should bind the rack and return the task", func(ctx SpecContext) {
			a := createArc(ctx, "set-rack-ok")
			grantOn(ctx, author.ResourceID(), access.ActionRetrieve, a.ResourceID())
			grantTask(ctx)
			res := MustSucceed(bind(ctx, a))
			Expect(res.Task).ToNot(BeNil())
			Expect(res.Task.Rack).To(Equal(testRack.Key))
			Expect(
				res.Task.Config,
			).To(HaveKeyWithValue("arc_key", a.Base().Key.String()))
			Expect(res.Task.Config).To(HaveKey("hash"))
		})

		It("Should clear the rack with task delete permission", func(ctx SpecContext) {
			a := createArc(ctx, "clear-rack-ok")
			deploy(ctx, a)
			grantOn(
				ctx,
				author.ResourceID(),
				access.ActionDelete,
				resource.ID{Type: resource.TypeTask},
				resource.ID{Type: resource.TypeStatus},
			)
			grantUpdateOn(ctx, author.ResourceID(), testRack.ResourceID())
			res := MustSucceed(
				updateTask(AuthedCtx(ctx, author), apiarc.UpdateTaskRequest{
					Key:  a.Base().Key,
					Rack: new(rack.Key(0)),
				}),
			)
			Expect(res.Task).To(BeNil())
			Expect(arcSvc.NewRetrieve().
				Where(arc.MatchKeys(a.Base().Key)).
				Exec(ctx, db)).To(Succeed())
		})

		It(
			"Should reject a clear without task delete permission",
			func(ctx SpecContext) {
				a := createArc(ctx, "clear-rack-no-task-delete")
				deploy(ctx, a)
				grantUpdateOn(
					ctx,
					author.ResourceID(),
					resource.ID{Type: resource.TypeTask},
					testRack.ResourceID(),
				)
				Expect(updateTask(AuthedCtx(ctx, author), apiarc.UpdateTaskRequest{
					Key:  a.Base().Key,
					Rack: new(rack.Key(0)),
				})).Error().To(MatchError(access.ErrDenied))
			},
		)

		It(
			"Should reject a performance change without task update permission",
			func(ctx SpecContext) {
				a := createArc(ctx, "performance-no-task-policy")
				deploy(ctx, a)
				grantUpdateOn(ctx, author.ResourceID(), a.ResourceID())
				Expect(
					updateTask(AuthedCtx(ctx, author), apiarc.UpdateTaskRequest{
						Key:         a.Base().Key,
						Performance: new(arctask.PerformanceHigh),
					}),
				).Error().To(MatchError(access.ErrDenied))
			},
		)

		It(
			"Should set the performance with task update permission",
			func(ctx SpecContext) {
				a := createArc(ctx, "performance-ok")
				deploy(ctx, a)
				grantUpdateOn(
					ctx,
					author.ResourceID(),
					resource.ID{Type: resource.TypeTask},
				)
				res := MustSucceed(
					updateTask(AuthedCtx(ctx, author), apiarc.UpdateTaskRequest{
						Key:         a.Base().Key,
						Performance: new(arctask.PerformanceHigh),
					}),
				)
				Expect(res.Task.Rack).To(Equal(testRack.Key))
				Expect(res.Task.Config).To(HaveKeyWithValue("performance", "high"))
			},
		)

		It("Should bubble up not found for a nonexistent arc", func(ctx SpecContext) {
			Expect(updateTask(AuthedCtx(ctx, author), apiarc.UpdateTaskRequest{
				Key:  uuid.New(),
				Rack: new(testRack.Key),
			})).Error().To(MatchError(query.ErrNotFound))
		})
	})
})
