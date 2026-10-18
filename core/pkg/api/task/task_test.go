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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apitask "github.com/synnaxlabs/synnax/pkg/api/task"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/pagerduty"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/query"
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
		grantUpdateOn(ctx, u.OntologyID(), t.OntologyID())
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
		grantUpdateOn(ctx, u.OntologyID(), t.OntologyID())
		Expect(apiSvc.Rename(
			AuthedCtx(ctx, u), nil,
			apitask.RenameRequest{Key: t.Key, Name: ""},
		)).Error().To(MatchError(ContainSubstring("name: required")))
		Expect(retrieveTask(ctx, t.Key).Name).To(Equal("Before"))
	})
	It("Should return a not found error for a missing task", func(ctx SpecContext) {
		u := createUser(ctx)
		key := uuid.New()
		grantUpdateOn(ctx, u.OntologyID(), task.OntologyID(key))
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
