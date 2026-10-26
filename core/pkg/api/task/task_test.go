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
	"encoding/json/v2"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apitask "github.com/synnaxlabs/synnax/pkg/api/task"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/pagerduty"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/filter"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
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
			Where: &task.Where{Key: &filter.Set[task.Key]{
				In: []task.Key{withoutStatus.Key, withStatus.Key},
			}},
			IncludeStatus: true,
		}))
		Expect(res.Tasks).To(HaveLen(2))
		Expect(res.Tasks[0].Key).To(Equal(withoutStatus.Key))
		Expect(res.Tasks[0].Status).To(BeNil())
		Expect(res.Tasks[1].Key).To(Equal(withStatus.Key))
		Expect(res.Tasks[1].Status.Key).To(Equal(withStatus.OntologyID().String()))
	})

	It("Should treat a missing key as a non-match", func(ctx SpecContext) {
		u := createUser(ctx)
		t := createTask(ctx)
		grant(ctx, access.ActionRetrieve, u.OntologyID(), t.OntologyID())
		res := MustSucceed(apiSvc.Retrieve(AuthedCtx(ctx, u), apitask.RetrieveRequest{
			Where: &task.Where{Key: &filter.Set[task.Key]{
				In: []task.Key{t.Key, uuid.New()},
			}},
		}))
		Expect(res.Tasks).To(HaveLen(1))
		Expect(res.Tasks[0].Key).To(Equal(t.Key))
	})

	It("Should return the tasks whose name matches the search term", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		match := task.Task{
			Type:     pagerduty.AlertTaskType,
			Rack:     svc.Rack.EmbeddedKey,
			Name:     "Cryostat pressure",
			Snapshot: true,
		}
		Expect(svc.Task.NewWriter(nil).Create(ctx, &match)).To(Succeed())
		other := createTask(ctx)
		grant(
			ctx, access.ActionRetrieve, u.OntologyID(),
			match.OntologyID(), other.OntologyID(),
		)
		res := MustSucceed(apiSvc.Retrieve(AuthedCtx(ctx, u), apitask.RetrieveRequest{
			SearchTerm: "cryostat",
		}))
		Expect(res.Tasks).To(HaveExactElements(HaveField("Key", match.Key)))
	})

	It("Should return a path error for an invalid regex", func(ctx SpecContext) {
		u := createUser(ctx)
		Expect(apiSvc.Retrieve(AuthedCtx(ctx, u), apitask.RetrieveRequest{
			Where: &task.Where{Type: &filter.String{Regex: new("(a")}},
		})).Error().To(SatisfyAll(
			BeAssignableToTypeOf(validate.PathError{}),
			MatchError(
				"where.type.regex: error parsing regexp: missing closing ): `(a`: validation error",
			),
		))
	})
})

var _ = Describe("RetrieveRequest", func() {
	It("Should reject a request member that it does not declare", func() {
		var req apitask.RetrieveRequest
		Expect(json.Unmarshal([]byte(`{"names":["Task"]}`), &req)).To(SatisfyAll(
			BeAssignableToTypeOf(&json.SemanticError{}),
			MatchError(ContainSubstring(`unknown object member name "names"`)),
		))
	})
})
