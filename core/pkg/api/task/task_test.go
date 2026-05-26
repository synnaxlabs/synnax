// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package task

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	xstatus "github.com/synnaxlabs/x/status"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// taskTypeOnly is the type-level ontology ID for granting access across all tasks.
var taskTypeOnly = ontology.ID{Type: ontology.ResourceTypeTask}

var _ = Describe("api/task Retrieve IncludeStatus", func() {
	// byTaskKey indexes the retrieved statuses by their owning task key so
	// attachment can be asserted independent of result ordering.
	byTaskKey := func(res RetrieveResponse) map[task.Key]*task.Status {
		out := make(map[task.Key]*task.Status, len(res.Tasks))
		for i := range res.Tasks {
			out[res.Tasks[i].Key] = res.Tasks[i].Status
		}
		return out
	}

	It("Should auto-heal a missing status row instead of failing the retrieve", func(ctx SpecContext) {
		grantOn(ctx, user.OntologyID(author.Key),
			[]access.Action{access.ActionRetrieve}, taskTypeOnly)
		t := task.Task{Key: task.NewKey(testRack.Key, 10), Name: "heal-target"}
		Expect(taskSvc.NewWriter(nil).Create(ctx, &t)).To(Succeed())
		Expect(statusSvc.NewWriter(nil).Delete(ctx, task.OntologyID(t.Key).String())).To(Succeed())

		res := MustSucceed(apiSvc.Retrieve(authedCtx(ctx, author), RetrieveRequest{
			Keys:          []task.Key{t.Key},
			IncludeStatus: true,
		}))
		Expect(res.Tasks).To(HaveLen(1))
		Expect(res.Tasks[0].Status).ToNot(BeNil())
		Expect(res.Tasks[0].Status.Variant).To(Equal(xstatus.VariantWarning))
		Expect(res.Tasks[0].Status.Message).To(Equal("heal-target status unknown"))
		Expect(res.Tasks[0].Status.Details.Task).To(Equal(t.Key))
	})

	It("Should attach each status to its own task when only some rows are missing", func(ctx SpecContext) {
		grantOn(ctx, user.OntologyID(author.Key),
			[]access.Action{access.ActionRetrieve}, taskTypeOnly)
		kept := task.Task{
			Key:  task.NewKey(testRack.Key, 20),
			Name: "keep-status",
			Status: &task.Status{
				Variant: xstatus.VariantSuccess,
				Message: "running",
				Time:    telem.Now(),
			},
		}
		lost := task.Task{Key: task.NewKey(testRack.Key, 21), Name: "lost-status"}
		Expect(taskSvc.NewWriter(nil).Create(ctx, &kept)).To(Succeed())
		Expect(taskSvc.NewWriter(nil).Create(ctx, &lost)).To(Succeed())
		Expect(statusSvc.NewWriter(nil).Delete(ctx, task.OntologyID(lost.Key).String())).To(Succeed())

		res := MustSucceed(apiSvc.Retrieve(authedCtx(ctx, author), RetrieveRequest{
			Keys:          []task.Key{kept.Key, lost.Key},
			IncludeStatus: true,
		}))
		Expect(res.Tasks).To(HaveLen(2))
		statuses := byTaskKey(res)

		Expect(statuses[kept.Key]).ToNot(BeNil())
		Expect(statuses[kept.Key].Variant).To(Equal(xstatus.VariantSuccess))
		Expect(statuses[kept.Key].Message).To(Equal("running"))
		Expect(statuses[kept.Key].Details.Task).To(Equal(kept.Key))

		Expect(statuses[lost.Key]).ToNot(BeNil())
		Expect(statuses[lost.Key].Variant).To(Equal(xstatus.VariantWarning))
		Expect(statuses[lost.Key].Message).To(Equal("lost-status status unknown"))
		Expect(statuses[lost.Key].Details.Task).To(Equal(lost.Key))
	})

	It("Should not attach a status when IncludeStatus is false", func(ctx SpecContext) {
		grantOn(ctx, user.OntologyID(author.Key),
			[]access.Action{access.ActionRetrieve}, taskTypeOnly)
		t := task.Task{Key: task.NewKey(testRack.Key, 30), Name: "no-status"}
		Expect(taskSvc.NewWriter(nil).Create(ctx, &t)).To(Succeed())

		res := MustSucceed(apiSvc.Retrieve(authedCtx(ctx, author), RetrieveRequest{
			Keys: []task.Key{t.Key},
		}))
		Expect(res.Tasks).To(HaveLen(1))
		Expect(res.Tasks[0].Status).To(BeNil())
	})
})
