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
	"fmt"
	"strings"
	"time"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/gorp"
	apiarc "github.com/synnaxlabs/synnax/pkg/api/arc"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	. "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	arctask "github.com/synnaxlabs/synnax/pkg/service/arc/task"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/spatial"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// scopedAction is a short local alias for the Arc action envelope.
type scopedAction = actions.Scoped[arc.Key, arc.Action]

// createArc persists a fresh Arc and returns it with its key populated. Writes
// commit immediately (nil tx) so access-control reads can observe the new
// ontology resource.
func createArc(ctx context.Context, name string) arc.Arc {
	GinkgoHelper()
	a := arc.Arc{Name: name, Mode: arc.ModeGraph}
	Expect(arcSvc.NewWriter(nil).Create(ctx, &a)).To(Succeed())
	return a
}

var _ = Describe("Service", func() {
	Describe("Retrieve", func() {
		It(
			"Should omit missing keys from a multi-key retrieve instead of failing",
			func(ctx SpecContext) {
				a := createArc(ctx, "partial-survivor")
				grantOn(
					ctx,
					author.OntologyID(),
					access.ActionRetrieve,
					arc.OntologyID(a.Key),
				)
				res := MustSucceed(
					apiSvc.Retrieve(AuthedCtx(ctx, author), apiarc.RetrieveRequest{
						Keys:                []arc.Key{a.Key, uuid.New()},
						IgnoreNotFoundError: true,
					}),
				)
				Expect(res.Arcs).To(HaveLen(1))
				Expect(res.Arcs[0].Key).To(Equal(a.Key))
			},
		)

		It(
			"Should return an empty result when every requested key is missing",
			func(ctx SpecContext) {
				res := MustSucceed(
					apiSvc.Retrieve(AuthedCtx(ctx, author), apiarc.RetrieveRequest{
						Keys:                []arc.Key{uuid.New()},
						IgnoreNotFoundError: true,
					}),
				)
				Expect(res.Arcs).To(BeEmpty())
			},
		)
	})

	Describe("Create", func() {
		It(
			"Should materialize the stored document, not echo a client-supplied Raw",
			func(ctx SpecContext) {
				grantOn(
					ctx,
					author.OntologyID(),
					access.ActionCreate,
					ontology.ID{Type: ontology.ResourceTypeArc},
				)
				a := apiarc.Arc{Name: "lying-raw", Mode: arc.ModeText}
				a.Text.Doc = text.Create("a -> b")
				a.Text.Raw = "a -> c"
				res := MustSucceed(
					apiSvc.Create(AuthedCtx(ctx, author), db, apiarc.CreateRequest{
						Arcs: []apiarc.Arc{a},
					}),
				)
				Expect(res.Arcs[0].Text.Raw).To(Equal("a -> b"))
			},
		)

		It("Should stamp the dispatch key onto the create broadcast", func(
			ctx SpecContext,
		) {
			grantOn(
				ctx,
				author.OntologyID(),
				access.ActionCreate,
				ontology.ID{Type: ontology.ResourceTypeArc},
			)
			rec := &Recorder[arc.Key, arc.Action]{}
			DeferCleanup(arcSvc.OnAction(rec.Record))
			res := MustSucceed(
				apiSvc.Create(AuthedCtx(ctx, author), db, apiarc.CreateRequest{
					Arcs:        []apiarc.Arc{{Name: "broadcast", Mode: arc.ModeGraph}},
					DispatchKey: "dk-create",
				}),
			)
			frames := rec.Snapshot()
			Expect(frames).To(HaveLen(1))
			Expect(frames[0].Key).To(Equal(res.Arcs[0].Key))
			Expect(frames[0].DispatchKey).To(Equal("dk-create"))
		})
	})

	Describe("Dispatch", func() {
		Describe("access control", func() {
			It(
				"Should reject the request with access.ErrDenied when the subject has no policy",
				func(ctx SpecContext) {
					a := createArc(ctx, "no-policy")
					Expect(apiSvc.Dispatch(
						AuthedCtx(ctx, author), db, apiarc.DispatchRequest{
							Key:         a.Key,
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
					grantUpdateOn(ctx, author.OntologyID(), a.OntologyID())
					Expect(apiSvc.Dispatch(
						AuthedCtx(ctx, author), db, apiarc.DispatchRequest{
							Key:         a.Key,
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
						Where(arc.MatchKeys(a.Key)).
						Entry(&res).Exec(ctx, nil)).To(Succeed())
					Expect(res.Graph.Nodes).To(HaveLen(1))
					Expect(
						res.Graph.Nodes[0].Position,
					).To(Equal(spatial.XY{X: 1, Y: 2}))
				},
			)

			It(
				"Should reject when the subject's policy targets a different Arc",
				func(ctx SpecContext) {
					a := createArc(ctx, "policy-target")
					b := createArc(ctx, "no-policy-target")
					grantUpdateOn(ctx, author.OntologyID(), a.OntologyID())
					Expect(apiSvc.Dispatch(
						AuthedCtx(ctx, author), db, apiarc.DispatchRequest{
							Key:         b.Key,
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

		Describe("delegation to Writer.Dispatch", func() {
			It(
				"Should apply a multi-action sequence to the target Arc",
				func(ctx SpecContext) {
					a := createArc(ctx, "multi-action")
					grantUpdateOn(ctx, author.OntologyID(), a.OntologyID())
					Expect(apiSvc.Dispatch(
						AuthedCtx(ctx, author), db, apiarc.DispatchRequest{
							Key:         a.Key,
							DispatchKey: "sess-1",
							Actions: []arc.Action{
								arc.NewSetNodeAction(
									arc.SetNodePayload{Node: graph.Node{Key: "n1"}},
								),
								arc.NewSetNodeAction(
									arc.SetNodePayload{Node: graph.Node{Key: "n2"}},
								),
								arc.NewRenameAction(
									arc.RenamePayload{Name: "renamed-multi"},
								),
							},
						},
					)).Error().To(Succeed())
					var res arc.Arc
					Expect(arcSvc.NewRetrieve().
						Where(arc.MatchKeys(a.Key)).
						Entry(&res).Exec(ctx, nil)).To(Succeed())
					Expect(res.Graph.Nodes).To(HaveLen(2))
					Expect(res.Name).To(Equal("renamed-multi"))
				},
			)

			It(
				"Should bubble up query.ErrNotFound when the target Arc does not exist",
				func(ctx SpecContext) {
					missing := uuid.New()
					grantUpdateOn(ctx, author.OntologyID(), arc.OntologyID(missing))
					Expect(apiSvc.Dispatch(
						AuthedCtx(ctx, author), db, apiarc.DispatchRequest{
							Key:         missing,
							DispatchKey: "sess-1",
							Actions: []arc.Action{
								arc.NewRenameAction(arc.RenamePayload{
									Name: "ghost",
								}),
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
					grantUpdateOn(ctx, author.OntologyID(), a.OntologyID())
					seen := make(chan scopedAction, 1)
					DeferCleanup(
						arcSvc.OnAction(func(_ context.Context, sa scopedAction) {
							seen <- sa
						}),
					)
					Expect(apiSvc.Dispatch(
						AuthedCtx(ctx, author), db, apiarc.DispatchRequest{
							Key:         a.Key,
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
					Expect(got.Key).To(Equal(a.Key))
					Expect(got.DispatchKey).To(Equal("session-marker-xyz"))
					Expect(got.Seq).To(BeNumerically(">", uint64(0)))
					Expect(got.Actions).To(HaveLen(1))
				},
			)
		})
	})

	Describe("Delete", func() {
		withTimeout := func(timeout time.Duration) *apiarc.Service {
			GinkgoHelper()
			return MustSucceed(apiarc.NewService(apiarc.ServiceConfig{
				RBAC:   rbacSvc,
				Arc:    arcSvc,
				Status: svc.Status,
				Stopper: MustSucceed(task.NewStopper(task.StopperConfig{
					Task:    svc.Task,
					Status:  svc.Status,
					Framer:  svc.Framer,
					Timeout: timeout,
				})),
			}))
		}
		deleteArcs := func(
			ctx context.Context,
			api *apiarc.Service,
			keys ...arc.Key,
		) error {
			return db.WithTx(ctx, func(tx gorp.Tx) error {
				_, err := api.Delete(
					AuthedCtx(ctx, author), tx, apiarc.DeleteRequest{Keys: keys},
				)
				return err
			})
		}
		createChannel := func(ctx SpecContext, ch channel.Channel) channel.Channel {
			GinkgoHelper()
			ch.Name = "arc_" + strings.ReplaceAll(uuid.New().String(), "-", "_")
			ch.Leaseholder = 1
			Expect(svc.Channel.NewWriter(nil).Create(ctx, &ch)).To(Succeed())
			return ch
		}
		// createTask stores a child task of a on rackKey.
		createTask := func(
			ctx SpecContext,
			a arc.Arc,
			rackKey rack.Key,
			stat *task.Status,
		) task.Task {
			GinkgoHelper()
			t := task.Task{
				Rack:   rackKey,
				Name:   a.Name,
				Type:   arc.TaskType,
				Config: msgpack.EncodedJSON{"arc_key": a.Key.String()},
				Status: stat,
			}
			Expect(svc.Task.NewWriter(nil).Create(ctx, &t)).To(Succeed())
			Expect(otg.NewWriter(nil).DefineRelationships(
				ctx, a.OntologyID(), ontology.RelationshipTypeParentOf, t.OntologyID(),
			)).To(Succeed())
			return t
		}
		retrieveRunning := func(g Gomega, ctx SpecContext, t task.Task) bool {
			GinkgoHelper()
			var stat task.Status
			g.Expect(svc.Status.NewRetrieve[task.StatusDetails]().
				Where(status.MatchKeys[task.StatusDetails](t.OntologyID().String())).
				Entry(&stat).
				Exec(ctx, nil)).To(Succeed())
			return stat.Details.Running
		}
		// startArc runs an Arc on the runtime in the Core until it writes to data.
		startArc := func(
			ctx SpecContext,
			data channel.Channel,
		) (arc.Arc, task.Task) {
			GinkgoHelper()
			a := arc.Arc{
				Name: "running-" + uuid.New().String(),
				Mode: arc.ModeText,
				Text: text.Text{
					Raw: fmt.Sprintf("interval(period=10ms) -> %s", data.Name),
				},
			}
			Expect(arcSvc.NewWriter(nil).Create(ctx, &a)).To(Succeed())
			var coreRack rack.Rack
			Expect(rackSvc.NewRetrieve().
				Where(rack.MatchNames("Node 1")).
				Entry(&coreRack).
				Exec(ctx, nil)).To(Succeed())
			t := createTask(ctx, a, coreRack.Key, nil)
			w := MustSucceed(svc.Framer.OpenWriter(ctx, framer.WriterConfig{
				Keys:  channel.Keys{svc.Task.CommandChannelKey()},
				Start: telem.Now(),
			}))
			Expect(w.Write(frame.NewUnary(
				svc.Task.CommandChannelKey(),
				MustSucceed(telem.NewJSONSeriesV(task.Command{
					Task: t.Key,
					Type: "start",
					Key:  uuid.New().String(),
					Args: msgpack.EncodedJSON{},
				})),
			))).To(BeTrue())
			Expect(w.Close()).To(Succeed())
			Eventually(func(g Gomega) {
				g.Expect(retrieveRunning(g, ctx, t)).To(BeTrue())
			}).Should(Succeed())
			return a, t
		}
		// A status with no runtime behind it. A stop of this task is never answered.
		fakeStatus := func(running bool) *task.Status {
			return &task.Status{
				Variant: status.VariantSuccess,
				Message: "Task status",
				Time:    telem.Now(),
				Details: task.StatusDetails{Running: running},
			}
		}
		createIdleArc := func(ctx SpecContext, running bool) (arc.Arc, task.Task) {
			GinkgoHelper()
			a := createArc(ctx, "idle-"+uuid.New().String())
			grantOn(ctx, author.OntologyID(), access.ActionDelete, a.OntologyID())
			return a, createTask(ctx, a, testRack.Key, fakeStatus(running))
		}
		expectGone := func(ctx SpecContext, a arc.Arc, t task.Task) {
			GinkgoHelper()
			Expect(arcSvc.NewRetrieve().Where(arc.MatchKeys(a.Key)).Exec(ctx, nil)).
				To(MatchError(query.ErrNotFound))
			Expect(svc.Task.NewRetrieve().Where(task.MatchKeys(t.Key)).Exec(ctx, nil)).
				To(MatchError(query.ErrNotFound))
		}
		createRuntimeChannels := func(ctx SpecContext) (channel.Channel, channel.Channel) {
			GinkgoHelper()
			index := createChannel(
				ctx, channel.Channel{DataType: telem.TimestampT, IsIndex: true},
			)
			data := createChannel(ctx, channel.Channel{
				DataType:   telem.Uint8T,
				LocalIndex: index.LocalKey,
			})
			return index, data
		}

		It(
			"Should release the channels of a running Arc with only Arc delete access",
			func(ctx SpecContext) {
				index, data := createRuntimeChannels(ctx)
				a, t := startArc(ctx, data)
				grantOn(ctx, author.OntologyID(), access.ActionDelete, a.OntologyID())
				Expect(
					deleteArcs(ctx, withTimeout(10*time.Second), a.Key),
				).To(Succeed())
				Expect(svc.Channel.NewWriter(nil).Delete(ctx, data.Key(), false)).
					To(Succeed())
				Expect(svc.Channel.NewWriter(nil).Delete(ctx, index.Key(), false)).
					To(Succeed())
				expectGone(ctx, a, t)
			},
		)
		It("Should not stop the task without Arc delete access", func(ctx SpecContext) {
			_, data := createRuntimeChannels(ctx)
			a, t := startArc(ctx, data)
			Expect(deleteArcs(ctx, apiSvc, a.Key)).To(MatchError(access.ErrDenied))
			Consistently(func(g Gomega) {
				g.Expect(retrieveRunning(g, ctx, t)).To(BeTrue())
			}, 200*time.Millisecond).Should(Succeed())
			grantOn(ctx, author.OntologyID(), access.ActionDelete, a.OntologyID())
			Expect(deleteArcs(ctx, apiSvc, a.Key)).To(Succeed())
		})
		It("Should stop only the tasks of the requested Arc", func(ctx SpecContext) {
			_, firstData := createRuntimeChannels(ctx)
			_, otherData := createRuntimeChannels(ctx)
			first, firstTask := startArc(ctx, firstData)
			other, otherTask := startArc(ctx, otherData)
			grantOn(ctx, author.OntologyID(), access.ActionDelete, first.OntologyID())
			Expect(deleteArcs(ctx, apiSvc, first.Key)).To(Succeed())
			expectGone(ctx, first, firstTask)
			Consistently(func(g Gomega) {
				g.Expect(retrieveRunning(g, ctx, otherTask)).To(BeTrue())
			}, 200*time.Millisecond).Should(Succeed())
			grantOn(ctx, author.OntologyID(), access.ActionDelete, other.OntologyID())
			Expect(deleteArcs(ctx, apiSvc, other.Key)).To(Succeed())
		})
		It("Should not wait for a stopped Arc", func(ctx SpecContext) {
			a, t := createIdleArc(ctx, false)
			start := time.Now()
			Expect(deleteArcs(ctx, withTimeout(10*time.Second), a.Key)).To(Succeed())
			Expect(time.Since(start)).To(BeNumerically("<", time.Second))
			expectGone(ctx, a, t)
		})
		It(
			"Should hold the timeout across every task and delete",
			func(ctx SpecContext) {
				const timeout = 200 * time.Millisecond
				first, firstTask := createIdleArc(ctx, true)
				other, otherTask := createIdleArc(ctx, true)
				start := time.Now()
				Expect(
					deleteArcs(ctx, withTimeout(timeout), first.Key, other.Key),
				).To(Succeed())
				Expect(time.Since(start)).To(BeNumerically("<", 2*timeout))
				expectGone(ctx, first, firstTask)
				expectGone(ctx, other, otherTask)
			},
		)
		It("Should keep the Arc when the caller cancels", func(ctx SpecContext) {
			a, t := createIdleArc(ctx, true)
			reqCtx, cancel := context.WithCancel(ctx)
			defer time.AfterFunc(50*time.Millisecond, cancel).Stop()
			Expect(deleteArcs(reqCtx, withTimeout(10*time.Second), a.Key)).
				To(MatchError(context.Canceled))
			Expect(arcSvc.NewRetrieve().Where(arc.MatchKeys(a.Key)).Exec(ctx, nil)).
				To(Succeed())
			Expect(svc.Task.NewRetrieve().Where(task.MatchKeys(t.Key)).Exec(ctx, nil)).
				To(Succeed())
		})
	})

	Describe("UpdateTask", func() {
		grantBind := func(ctx SpecContext, a arc.Arc) {
			GinkgoHelper()
			grantUpdateOn(ctx, author.OntologyID(), a.OntologyID())
			grantOn(
				ctx,
				author.OntologyID(),
				access.ActionCreate,
				ontology.ID{Type: ontology.ResourceTypeTask},
			)
		}

		It(
			"Should reject the request when the subject has no policy",
			func(ctx SpecContext) {
				a := createArc(ctx, "set-rack-no-policy")
				Expect(
					apiSvc.UpdateTask(
						AuthedCtx(ctx, author),
						db,
						apiarc.UpdateTaskRequest{
							Key:  a.Key,
							Rack: new(testRack.Key),
						},
					),
				).Error().
					To(MatchError(access.ErrDenied))
			},
		)

		It(
			"Should reject when the subject may update the arc but not create tasks",
			func(ctx SpecContext) {
				a := createArc(ctx, "set-rack-no-task-policy")
				grantUpdateOn(ctx, author.OntologyID(), a.OntologyID())
				Expect(
					apiSvc.UpdateTask(
						AuthedCtx(ctx, author),
						db,
						apiarc.UpdateTaskRequest{
							Key:  a.Key,
							Rack: new(testRack.Key),
						},
					),
				).Error().
					To(MatchError(access.ErrDenied))
			},
		)

		It("Should bind the rack and return the task", func(ctx SpecContext) {
			a := createArc(ctx, "set-rack-ok")
			grantBind(ctx, a)
			res := MustSucceed(
				apiSvc.UpdateTask(AuthedCtx(ctx, author), db, apiarc.UpdateTaskRequest{
					Key:  a.Key,
					Rack: new(testRack.Key),
				}),
			)
			Expect(res.Task).ToNot(BeNil())
			Expect(res.Task.Rack).To(Equal(testRack.Key))
			Expect(res.Task.Config).To(HaveKeyWithValue("arc_key", a.Key.String()))
			Expect(res.Task.Config).To(HaveKey("hash"))
		})

		It("Should clear the rack with task delete permission", func(ctx SpecContext) {
			a := createArc(ctx, "clear-rack-ok")
			grantBind(ctx, a)
			grantOn(
				ctx,
				author.OntologyID(),
				access.ActionDelete,
				ontology.ID{Type: ontology.ResourceTypeTask},
			)
			deployed := MustSucceed(
				apiSvc.UpdateTask(AuthedCtx(ctx, author), db, apiarc.UpdateTaskRequest{
					Key:  a.Key,
					Rack: new(testRack.Key),
				}),
			)
			Expect(deployed.Task).ToNot(BeNil())
			res := MustSucceed(
				apiSvc.UpdateTask(AuthedCtx(ctx, author), db, apiarc.UpdateTaskRequest{
					Key:  a.Key,
					Rack: new(rack.Key(0)),
				}),
			)
			Expect(res.Task).To(BeNil())
			Expect(arcSvc.NewRetrieve().
				Where(arc.MatchKeys(a.Key)).
				Exec(ctx, nil)).To(Succeed())
		})

		It(
			"Should reject a performance change without task update permission",
			func(ctx SpecContext) {
				a := createArc(ctx, "performance-no-task-policy")
				grantBind(ctx, a)
				MustSucceed(
					apiSvc.UpdateTask(
						AuthedCtx(ctx, author),
						db,
						apiarc.UpdateTaskRequest{
							Key:  a.Key,
							Rack: new(testRack.Key),
						},
					),
				)
				Expect(
					apiSvc.UpdateTask(
						AuthedCtx(ctx, author),
						db,
						apiarc.UpdateTaskRequest{
							Key:         a.Key,
							Performance: new(arctask.PerformanceHigh),
						},
					),
				).Error().
					To(MatchError(access.ErrDenied))
			},
		)

		It(
			"Should set the performance with task update permission",
			func(ctx SpecContext) {
				a := createArc(ctx, "performance-ok")
				grantBind(ctx, a)
				grantOn(
					ctx,
					author.OntologyID(),
					access.ActionUpdate,
					ontology.ID{Type: ontology.ResourceTypeTask},
				)
				MustSucceed(
					apiSvc.UpdateTask(
						AuthedCtx(ctx, author),
						db,
						apiarc.UpdateTaskRequest{
							Key:  a.Key,
							Rack: new(testRack.Key),
						},
					),
				)
				res := MustSucceed(
					apiSvc.UpdateTask(
						AuthedCtx(ctx, author),
						db,
						apiarc.UpdateTaskRequest{
							Key:         a.Key,
							Performance: new(arctask.PerformanceHigh),
						},
					),
				)
				Expect(res.Task.Rack).To(Equal(testRack.Key))
				Expect(res.Task.Config).To(HaveKeyWithValue("performance", "high"))
			},
		)

		It("Should bubble up not found for a nonexistent arc", func(ctx SpecContext) {
			missing := uuid.New()
			grantUpdateOn(ctx, author.OntologyID(), arc.OntologyID(missing))
			grantOn(
				ctx,
				author.OntologyID(),
				access.ActionCreate,
				ontology.ID{Type: ontology.ResourceTypeTask},
			)
			Expect(
				apiSvc.UpdateTask(AuthedCtx(ctx, author), db, apiarc.UpdateTaskRequest{
					Key:  missing,
					Rack: new(testRack.Key),
				}),
			).Error().
				To(MatchError(query.ErrNotFound))
		})
	})
})
