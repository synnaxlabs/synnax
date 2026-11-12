// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package vcs_test

import (
	"context"
	"time"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"github.com/synnaxlabs/gorp"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	apivcs "github.com/synnaxlabs/synnax/pkg/api/vcs"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/pagerduty"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	svcstatus "github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/synnax/pkg/service/vcs"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// route runs one route of the API.
type route = func(context.Context) (any, error)

var (
	status = func(ctx context.Context) (any, error) {
		return shared.api.Status(ctx, apivcs.StatusRequest{})
	}
	connect = func(ctx context.Context) (any, error) {
		return shared.api.Connect(ctx, apivcs.ConnectRequest{})
	}
	fetch = func(ctx context.Context) (any, error) {
		return shared.api.Fetch(ctx, apivcs.FetchRequest{})
	}
	commit = func(ctx context.Context) (any, error) {
		return shared.api.Commit(ctx, apivcs.CommitRequest{Message: "Commit"})
	}
	discard = func(ctx context.Context) (any, error) {
		return shared.api.Discard(ctx, apivcs.DiscardRequest{})
	}
	write = func(ctx context.Context) (any, error) {
		return shared.api.Write(ctx, vcs.Inverse{})
	}
	plan = func(ctx context.Context) (any, error) {
		return shared.api.Plan(ctx, apivcs.PlanRequest{})
	}
	apply = func(ctx context.Context) (any, error) {
		return shared.api.Apply(ctx, apivcs.ApplyRequest{})
	}
	log = func(ctx context.Context) (any, error) {
		return shared.api.Log(ctx, apivcs.LogRequest{})
	}
	show = func(ctx context.Context) (any, error) {
		return shared.api.Show(ctx, apivcs.ShowRequest{})
	}
	configurations = func(ctx context.Context) (any, error) {
		return shared.api.Configurations(ctx, apivcs.ConfigurationsRequest{})
	}
	clone = func(ctx context.Context) (any, error) {
		return shared.api.Clone(ctx, apivcs.CloneRequest{Name: "copy"})
	}
	compare = func(ctx context.Context) (any, error) {
		return shared.api.Compare(ctx, apivcs.CompareRequest{})
	}
	snapshot = func(ctx context.Context) (any, error) {
		return shared.api.Snapshot(ctx, apivcs.SnapshotRequest{Range: "r1"})
	}
	snapshots = func(ctx context.Context) (any, error) {
		return shared.api.Snapshots(ctx, apivcs.SnapshotsRequest{Range: "r1"})
	}
)

var (
	engineer = func() role.Key { return shared.layer.RBAC.Builtin.EngineerKey }
	operator = func() role.Key { return shared.layer.RBAC.Builtin.OperatorKey }
	viewer   = func() role.Key { return shared.layer.RBAC.Builtin.ViewerKey }
)

var _ = Describe("Access", func() {
	DescribeTable(
		"Should check the action of a route on version control",
		func(ctx SpecContext, call route, action access.Action, ran types.GomegaMatcher) {
			u := shared.createUser(ctx)
			Expect(call(AuthedCtx(ctx, u))).Error().
				To(denied(u, action, vcs.ResourceID))
			shared.grant(ctx, u, action, vcs.ResourceID)
			Expect(call(AuthedCtx(ctx, u))).Error().To(ran)
		},
		Entry("connect", connect, access.ActionDelete, SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring(`configuration: "" is not a valid name`)),
		)),
		Entry("fetch", fetch, access.ActionRetrieve, MatchError(vcs.ErrNotStarted)),
		Entry("commit", commit, access.ActionUpdate, MatchError(vcs.ErrNotStarted)),
		Entry("discard", discard, access.ActionUpdate, MatchError(vcs.ErrNotStarted)),
		Entry("write", write, access.ActionUpdate, Succeed()),
		Entry("plan", plan, access.ActionRetrieve, MatchError(vcs.ErrNotStarted)),
		Entry("apply", apply, access.ActionUpdate, MatchError(vcs.ErrNotStarted)),
		Entry("log", log, access.ActionRetrieve, MatchError(vcs.ErrNotStarted)),
		Entry("show", show, access.ActionRetrieve, MatchError(vcs.ErrNotStarted)),
		Entry(
			"configurations",
			configurations,
			access.ActionRetrieve,
			MatchError(vcs.ErrNotStarted),
		),
		Entry("clone", clone, access.ActionUpdate, MatchError(vcs.ErrNotStarted)),
		Entry("compare", compare, access.ActionRetrieve, MatchError(vcs.ErrNotStarted)),
		Entry("snapshot", snapshot, access.ActionCreate, MatchError(vcs.ErrNotStarted)),
		Entry(
			"snapshots",
			snapshots,
			access.ActionRetrieve,
			MatchError(vcs.ErrNotStarted),
		),
	)

	It(
		"Should check retrieve on version control for the status",
		func(ctx SpecContext) {
			u := shared.createUser(ctx)
			Expect(status(AuthedCtx(ctx, u))).Error().
				To(denied(u, access.ActionRetrieve, vcs.ResourceID))
			shared.grant(ctx, u, access.ActionRetrieve, vcs.ResourceID)
			st := MustSucceed(
				shared.api.Status(AuthedCtx(ctx, u), apivcs.StatusRequest{}),
			)
			Expect(st.Enabled).To(BeFalse())
		},
	)

	DescribeTable("Should let a builtin role run the routes it holds",
		func(ctx SpecContext, key func() role.Key, call route) {
			u := shared.createUserWithRole(ctx, key())
			Expect(call(AuthedCtx(ctx, u))).Error().To(MatchError(vcs.ErrNotStarted))
		},
		Entry("Engineer log", engineer, log),
		Entry("Engineer commit", engineer, commit),
		Entry("Engineer clone", engineer, clone),
		Entry("Engineer snapshot", engineer, snapshot),
		Entry("Operator log", operator, log),
		Entry("Operator snapshot", operator, snapshot),
		Entry("Viewer log", viewer, log),
	)

	DescribeTable("Should deny a builtin role the routes it lacks",
		func(ctx SpecContext, key func() role.Key, call route, action access.Action) {
			u := shared.createUserWithRole(ctx, key())
			Expect(call(AuthedCtx(ctx, u))).Error().
				To(denied(u, action, vcs.ResourceID))
		},
		Entry("Engineer connect", engineer, connect, access.ActionDelete),
		Entry("Operator connect", operator, connect, access.ActionDelete),
		Entry("Operator commit", operator, commit, access.ActionUpdate),
		Entry("Operator clone", operator, clone, access.ActionUpdate),
		Entry("Viewer snapshot", viewer, snapshot, access.ActionCreate),
	)

	Describe("Files", func() {
		files := func(ctx context.Context, res string) (apivcs.FilesResponse, error) {
			return shared.api.Files(
				ctx,
				apivcs.FilesRequest{Rev: "head", Resource: res},
			)
		}

		It("Should reject a request with no resource", func(ctx SpecContext) {
			u := shared.createUser(ctx)
			Expect(files(AuthedCtx(ctx, u), "")).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("files: a resource is required")),
			))
		})

		It(
			"Should check retrieve on version control and on the resource",
			func(ctx SpecContext) {
				u := shared.createUser(ctx)
				res := schematic.ResourceID(uuid.New())
				Expect(files(AuthedCtx(ctx, u), res.String())).Error().
					To(denied(u, access.ActionRetrieve, vcs.ResourceID))
				shared.grant(ctx, u, access.ActionRetrieve, vcs.ResourceID)
				Expect(files(AuthedCtx(ctx, u), res.String())).Error().
					To(denied(u, access.ActionRetrieve, res))
				shared.grant(ctx, u, access.ActionRetrieve, res)
				Expect(files(AuthedCtx(ctx, u), res.String())).Error().
					To(MatchError(vcs.ErrNotStarted))
			},
		)
	})

	Describe("Uncommitted changes", func() {
		var (
			e   env
			sch schematic.Schematic
		)
		BeforeEach(func(ctx SpecContext) {
			e = openEnv(ctx)
			p := project.Project{Name: "Project"}
			Expect(e.db.WithTx(ctx, func(tx gorp.Tx) error {
				return e.layer.Project.NewWriter(tx).Create(ctx, &p)
			})).To(Succeed())
			MustSucceed(e.layer.VCS.Connect(ctx, vcs.ConnectRequest{
				Configuration: "main",
				Author:        vcs.Author{Name: "Ada", Email: "ada@synnax.local"},
			}))
			sch = schematic.Schematic{Name: "Valves", Project: &p.Key}
			Expect(e.db.WithTx(ctx, func(tx gorp.Tx) error {
				return e.layer.Schematic.NewWriter(tx).Create(ctx, &sch)
			})).To(Succeed())
			Eventually(func(g Gomega) {
				st := MustSucceed(e.layer.VCS.Status(ctx))
				g.Expect(st.Changes).To(ConsistOf(
					HaveField("Resource", sch.ResourceID().String()),
				))
			}).WithTimeout(5 * time.Second).Should(Succeed())
		})

		It(
			"Should check retrieve on each change that a commit holds",
			func(ctx SpecContext) {
				u := e.createUser(ctx)
				e.grant(ctx, u, access.ActionUpdate, vcs.ResourceID)
				req := apivcs.CommitRequest{Message: "Add the valves"}
				Expect(e.api.Commit(AuthedCtx(ctx, u), req)).Error().
					To(denied(u, access.ActionRetrieve, sch.ResourceID()))
				e.grant(ctx, u, access.ActionRetrieve, sch.ResourceID())
				res := MustSucceed(e.api.Commit(AuthedCtx(ctx, u), req))
				Expect(res.Commit.Message).To(Equal("Add the valves"))
			},
		)

		It("Should check retrieve on each named resource", func(ctx SpecContext) {
			u := e.createUser(ctx)
			e.grant(ctx, u, access.ActionUpdate, vcs.ResourceID)
			req := apivcs.CommitRequest{
				Message:   "Add the valves",
				Resources: []string{sch.ResourceID().String()},
			}
			Expect(e.api.Commit(AuthedCtx(ctx, u), req)).Error().
				To(denied(u, access.ActionRetrieve, sch.ResourceID()))
			e.grant(ctx, u, access.ActionRetrieve, sch.ResourceID())
			MustSucceed(e.api.Commit(AuthedCtx(ctx, u), req))
		})

		It(
			"Should check a change that the status does not show yet",
			func(ctx SpecContext) {
				u := e.createUser(ctx)
				e.grant(ctx, u, access.ActionUpdate, vcs.ResourceID)
				e.grant(ctx, u, access.ActionRetrieve, sch.ResourceID())
				fresh := schematic.Schematic{Name: "Pumps", Project: sch.Project}
				Expect(e.db.WithTx(ctx, func(tx gorp.Tx) error {
					return e.layer.Schematic.NewWriter(tx).Create(ctx, &fresh)
				})).To(Succeed())
				req := apivcs.CommitRequest{Message: "Add the pumps"}
				Expect(e.api.Commit(AuthedCtx(ctx, u), req)).Error().
					To(denied(u, access.ActionRetrieve, fresh.ResourceID()))
			},
		)

		It(
			"Should check delete on a running task before a discard stops it",
			func(ctx SpecContext) {
				r := rack.Rack{Name: "Driver"}
				t := task.Task{
					Name:   "Alerts",
					Type:   pagerduty.AlertTaskType,
					Config: msgpack.EncodedJSON{"routing_key": "rk"},
				}
				running := task.Status{
					Variant: svcstatus.VariantSuccess,
					Message: "Running",
					Time:    telem.Now(),
					Details: task.StatusDetails{Running: true},
				}
				Expect(e.db.WithTx(ctx, func(tx gorp.Tx) error {
					if err := e.layer.Rack.NewWriter(tx).Create(ctx, &r); err != nil {
						return err
					}
					t.Rack = r.Key
					if err := e.layer.Task.NewWriter(tx).Create(ctx, &t); err != nil {
						return err
					}
					running.Owner = new(t.ResourceID())
					return e.layer.Status.NewWriter(tx).Set(ctx, &running)
				})).To(Succeed())
				u := e.createUser(ctx)
				e.grant(ctx, u, access.ActionUpdate, vcs.ResourceID)
				req := apivcs.DiscardRequest{
					Resources: []string{t.ResourceID().String()},
				}
				Expect(e.api.Discard(AuthedCtx(ctx, u), req)).Error().
					To(denied(u, access.ActionDelete, t.ResourceID()))
				Expect(e.layer.Task.NewRetrieve().
					Where(task.MatchKeys(t.Key)).
					Exec(ctx, e.db)).To(Succeed())
			},
		)

		It(
			"Should check retrieve on each change that a snapshot holds",
			func(ctx SpecContext) {
				u := e.createUser(ctx)
				e.grant(ctx, u, access.ActionCreate, vcs.ResourceID)
				req := apivcs.SnapshotRequest{Range: "r1"}
				Expect(e.api.Snapshot(AuthedCtx(ctx, u), req)).Error().
					To(denied(u, access.ActionRetrieve, sch.ResourceID()))
				e.grant(ctx, u, access.ActionRetrieve, sch.ResourceID())
				res := MustSucceed(e.api.Snapshot(AuthedCtx(ctx, u), req))
				Expect(res.Snapshot.Range).To(Equal("r1"))
				Expect(res.Snapshot.Uncommitted).To(Equal(1))
			},
		)
	})
})
