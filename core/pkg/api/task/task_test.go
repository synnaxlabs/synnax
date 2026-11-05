// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package task_test

import (
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apitask "github.com/synnaxlabs/synnax/pkg/api/task"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/pagerduty"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

func createTask(ctx SpecContext) task.Task {
	GinkgoHelper()
	t := task.Task{
		Type:     pagerduty.AlertTaskType,
		Rack:     svc.Rack.EmbeddedKey,
		Name:     "Before",
		Snapshot: true,
	}
	Expect(svc.Task.NewWriter(nil).Create(ctx, &t)).To(Succeed())
	return t
}

func retrieveTask(ctx SpecContext, key task.Key) task.Task {
	GinkgoHelper()
	var t task.Task
	Expect(svc.Task.NewRetrieve().
		Where(task.MatchKeys(key)).
		Entry(&t).
		Exec(ctx, nil)).To(Succeed())
	return t
}

var _ = Describe("Rename", func() {
	It("Should rename the task when update is granted", func(ctx SpecContext) {
		u := createUser(ctx)
		t := createTask(ctx)
		grant(ctx, access.ActionUpdate, u.ResourceID(), t.ResourceID())
		Expect(apiSvc.Rename(
			AuthedCtx(ctx, u), apitask.RenameRequest{Key: t.Key, Name: "After"},
		)).Error().ToNot(HaveOccurred())
		renamed := retrieveTask(ctx, t.Key)
		Expect(renamed.Name).To(Equal("After"))
		Expect(renamed.Snapshot).To(BeTrue())
		Expect(renamed.Config).To(Equal(t.Config))
	})
	It("Should reject an empty name", func(ctx SpecContext) {
		u := createUser(ctx)
		t := createTask(ctx)
		grant(ctx, access.ActionUpdate, u.ResourceID(), t.ResourceID())
		Expect(apiSvc.Rename(
			AuthedCtx(ctx, u), apitask.RenameRequest{Key: t.Key, Name: ""},
		)).Error().To(MatchError(ContainSubstring("name: required")))
		Expect(retrieveTask(ctx, t.Key).Name).To(Equal("Before"))
	})
	It("Should return a not found error for a missing task", func(ctx SpecContext) {
		u := createUser(ctx)
		key := uuid.New()
		grant(ctx, access.ActionUpdate, u.ResourceID(), task.ResourceID(key))
		Expect(apiSvc.Rename(
			AuthedCtx(ctx, u), apitask.RenameRequest{Key: key, Name: "After"},
		)).Error().To(MatchError(query.ErrNotFound))
	})
	It("Should reject the request when update is not granted", func(ctx SpecContext) {
		u := createUser(ctx)
		t := createTask(ctx)
		Expect(apiSvc.Rename(
			AuthedCtx(ctx, u), apitask.RenameRequest{Key: t.Key, Name: "After"},
		)).Error().To(MatchError(access.ErrDenied))
		Expect(retrieveTask(ctx, t.Key).Name).To(Equal("Before"))
	})
})

var _ = Describe("Dispatch", func() {
	createDraft := func(ctx SpecContext) task.Task {
		GinkgoHelper()
		t := task.Task{
			Type: pagerduty.AlertTaskType,
			Rack: svc.Rack.EmbeddedKey,
			Name: "Alerts",
		}
		Expect(svc.Task.NewWriter(nil).Create(ctx, &t)).To(Succeed())
		return t
	}
	setRoutingKey := []task.Action{task.NewSetAction(task.SetPayload{
		Path:  []string{"config", "routing_key"},
		Value: "rk-2",
	})}
	It("Should apply the actions and return the ones the Core added", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		t := createDraft(ctx)
		grant(ctx, access.ActionUpdate, u.ResourceID(), t.ResourceID())
		grant(ctx, access.ActionRetrieve, u.ResourceID(), t.ResourceID())
		res := MustSucceed(apiSvc.Dispatch(
			AuthedCtx(ctx, u),
			apitask.DispatchRequest{
				Key:         t.Key,
				DispatchKey: "dk",
				Actions:     setRoutingKey,
			},
		))
		stored := retrieveTask(ctx, t.Key)
		Expect(stored.Config["routing_key"]).To(Equal("rk-2"))
		Expect(stored.ConfigHash).ToNot(Equal(t.ConfigHash))
		Expect(res.Actions).To(Equal([]task.Action{task.NewSetAction(task.SetPayload{
			Path:  []string{"config_hash"},
			Value: stored.ConfigHash,
		})}))
	})
	It("Should return no actions to a subject that cannot read the task", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		t := createDraft(ctx)
		grant(ctx, access.ActionUpdate, u.ResourceID(), t.ResourceID())
		res := MustSucceed(apiSvc.Dispatch(
			AuthedCtx(ctx, u),
			apitask.DispatchRequest{
				Key:         t.Key,
				DispatchKey: "dk",
				Actions:     setRoutingKey,
			},
		))
		Expect(res.Actions).To(BeEmpty())
		Expect(retrieveTask(ctx, t.Key).Config["routing_key"]).To(Equal("rk-2"))
	})
	It("Should reject the request when update is not granted", func(ctx SpecContext) {
		u := createUser(ctx)
		t := createDraft(ctx)
		Expect(apiSvc.Dispatch(
			AuthedCtx(ctx, u),
			apitask.DispatchRequest{
				Key:         t.Key,
				DispatchKey: "dk",
				Actions:     setRoutingKey,
			},
		)).Error().To(MatchError(access.ErrDenied))
		Expect(retrieveTask(ctx, t.Key).ConfigHash).To(Equal(t.ConfigHash))
	})
	It("Should return the dispatch error", func(ctx SpecContext) {
		u := createUser(ctx)
		key := uuid.New()
		grant(ctx, access.ActionUpdate, u.ResourceID(), task.ResourceID(key))
		Expect(apiSvc.Dispatch(
			AuthedCtx(ctx, u),
			apitask.DispatchRequest{
				Key:         key,
				DispatchKey: "dk",
				Actions:     setRoutingKey,
			},
		)).Error().To(MatchError(query.ErrNotFound))
	})
})

var _ = Describe("Create", func() {
	It(
		"Should stamp the dispatch key onto the create broadcast",
		func(ctx SpecContext) {
			u := createUser(ctx)
			grant(ctx, access.ActionCreate, u.ResourceID(), resource.ID{
				Type: resource.TypeTask,
			})
			var keys []string
			DeferCleanup(svc.Task.OnAction(func(
				_ context.Context,
				f actions.Scoped[task.Key, task.Action],
			) {
				keys = append(keys, f.DispatchKey)
			}))
			res := MustSucceed(
				apiSvc.Create(AuthedCtx(ctx, u), nil, apitask.CreateRequest{
					Tasks: []task.Task{{
						Type: pagerduty.AlertTaskType,
						Rack: svc.Rack.EmbeddedKey,
						Name: "Created",
					}},
					DispatchKey: "dk-create",
				}),
			)
			Expect(res.Tasks).To(HaveLen(1))
			Expect(keys).To(Equal([]string{"dk-create"}))
		},
	)
})

var _ = Describe("Retrieve", func() {
	It("Should attach only each task's own status", func(ctx SpecContext) {
		u := createUser(ctx)
		withStatus := createTask(ctx)
		withoutStatus := createTask(ctx)
		w := svc.Status.NewWriter(nil)
		Expect(w.DeleteOwned(ctx, withoutStatus.ResourceID())).To(Succeed())
		grant(
			ctx, access.ActionRetrieve, u.ResourceID(),
			withStatus.ResourceID(), withoutStatus.ResourceID(),
		)
		res := MustSucceed(apiSvc.Retrieve(AuthedCtx(ctx, u), apitask.RetrieveRequest{
			Keys:          []task.Key{withoutStatus.Key, withStatus.Key},
			IncludeStatus: true,
		}))
		Expect(res.Tasks).To(HaveLen(2))
		Expect(res.Tasks[0].Key).To(Equal(withoutStatus.Key))
		Expect(res.Tasks[0].Status).To(BeNil())
		Expect(res.Tasks[1].Key).To(Equal(withStatus.Key))
		Expect(res.Tasks[1].Status.Owner).To(HaveValue(Equal(withStatus.ResourceID())))
	})

	It("Should keep only the tasks that run the given Arcs", func(ctx SpecContext) {
		u := createUser(ctx)
		grant(ctx, access.ActionRetrieve, u.ResourceID(), resource.ID{
			Type: resource.TypeTask,
		})
		w := svc.Arc.NewWriter(nil)
		deployed := arc.Arc{Name: "Deployed", Mode: arc.ModeGraph}
		other := arc.Arc{Name: "Other", Mode: arc.ModeGraph}
		Expect(w.Create(ctx, &deployed)).To(Succeed())
		Expect(w.Create(ctx, &other)).To(Succeed())
		update := arc.TaskUpdate{Rack: new(svc.Rack.EmbeddedKey)}
		tsk := MustSucceed(w.UpdateTask(ctx, deployed.Key, update))
		MustSucceed(w.UpdateTask(ctx, other.Key, update))
		res := MustSucceed(apiSvc.Retrieve(AuthedCtx(ctx, u), apitask.RetrieveRequest{
			Arcs: []arc.Key{deployed.Key},
		}))
		Expect(res.Tasks).To(HaveLen(1))
		Expect(res.Tasks[0].Key).To(Equal(tsk.Key))
	})
})

var _ = Describe("Copy", func() {
	It("Should place a snapshot copy in a range", func(ctx SpecContext) {
		u := createUser(ctx)
		t := createTask(ctx)
		grant(ctx, access.ActionRetrieve, u.ResourceID(), resource.ID{
			Type: resource.TypeTask,
		})
		grant(ctx, access.ActionCreate, u.ResourceID(), resource.ID{
			Type: resource.TypeTask,
		})
		r := ranger.Range{
			Name:      "Snapshot Range",
			TimeRange: telem.TimeRange{Start: telem.Now(), End: telem.TimeStampMax},
		}
		Expect(svc.Ranger.NewWriter(nil).Create(ctx, &r)).To(Succeed())
		res := MustSucceed(apiSvc.Copy(AuthedCtx(ctx, u), nil, apitask.CopyRequest{
			Key:      t.Key,
			Name:     "Copy",
			Snapshot: true,
			Range:    &r.Key,
		}))
		Expect(res.Task.Range).To(Equal(&r.Key))
		found := MustSucceed(apiSvc.Retrieve(AuthedCtx(ctx, u), apitask.RetrieveRequest{
			Ranges: []ranger.Key{r.Key},
		}))
		Expect(found.Tasks).To(HaveLen(1))
		Expect(found.Tasks[0].Key).To(Equal(res.Task.Key))
	})
})
