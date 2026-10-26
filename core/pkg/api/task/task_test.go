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
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/pagerduty"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/query"
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
		grant(ctx, access.ActionUpdate, u.OntologyID(), t.OntologyID())
		Expect(apiSvc.Rename(
			AuthedCtx(ctx, u), nil,
			apitask.RenameRequest{Key: t.Key, Name: "After"},
		)).Error().ToNot(HaveOccurred())
		renamed := retrieveTask(ctx, t.Key)
		Expect(renamed.Name).To(Equal("After"))
		Expect(renamed.Snapshot).To(BeTrue())
		Expect(renamed.Config).To(Equal(t.Config))
	})
	It("Should reject an empty name", func(ctx SpecContext) {
		u := createUser(ctx)
		t := createTask(ctx)
		grant(ctx, access.ActionUpdate, u.OntologyID(), t.OntologyID())
		Expect(apiSvc.Rename(
			AuthedCtx(ctx, u), nil,
			apitask.RenameRequest{Key: t.Key, Name: ""},
		)).Error().To(MatchError(ContainSubstring("name: required")))
		Expect(retrieveTask(ctx, t.Key).Name).To(Equal("Before"))
	})
	It("Should return a not found error for a missing task", func(ctx SpecContext) {
		u := createUser(ctx)
		key := uuid.New()
		grant(ctx, access.ActionUpdate, u.OntologyID(), task.OntologyID(key))
		Expect(apiSvc.Rename(
			AuthedCtx(ctx, u), nil,
			apitask.RenameRequest{Key: key, Name: "After"},
		)).Error().To(MatchError(query.ErrNotFound))
	})
	It("Should reject the request when update is not granted", func(ctx SpecContext) {
		u := createUser(ctx)
		t := createTask(ctx)
		Expect(apiSvc.Rename(
			AuthedCtx(ctx, u), nil,
			apitask.RenameRequest{Key: t.Key, Name: "After"},
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
		grant(ctx, access.ActionUpdate, u.OntologyID(), t.OntologyID())
		res := MustSucceed(apiSvc.Dispatch(
			AuthedCtx(ctx, u),
			nil,
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
	It("Should reject the request when update is not granted", func(ctx SpecContext) {
		u := createUser(ctx)
		t := createDraft(ctx)
		Expect(apiSvc.Dispatch(
			AuthedCtx(ctx, u),
			nil,
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
		grant(ctx, access.ActionUpdate, u.OntologyID(), task.OntologyID(key))
		Expect(apiSvc.Dispatch(
			AuthedCtx(ctx, u),
			nil,
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
			grant(ctx, access.ActionCreate, u.OntologyID(), ontology.ID{
				Type: ontology.ResourceTypeTask,
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
		Expect(svc.Status.NewWriter(nil).Delete(
			ctx, withoutStatus.OntologyID().String(),
		)).To(Succeed())
		grant(
			ctx, access.ActionRetrieve, u.OntologyID(),
			withStatus.OntologyID(), withoutStatus.OntologyID(),
		)
		res := MustSucceed(apiSvc.Retrieve(AuthedCtx(ctx, u), apitask.RetrieveRequest{
			Keys:          []task.Key{withoutStatus.Key, withStatus.Key},
			IncludeStatus: true,
		}))
		Expect(res.Tasks).To(HaveLen(2))
		Expect(res.Tasks[0].Key).To(Equal(withoutStatus.Key))
		Expect(res.Tasks[0].Status).To(BeNil())
		Expect(res.Tasks[1].Key).To(Equal(withStatus.Key))
		Expect(res.Tasks[1].Status.Key).To(Equal(withStatus.OntologyID().String()))
	})
})
