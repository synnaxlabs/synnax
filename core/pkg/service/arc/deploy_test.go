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
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Deploy", func() {
	var a arc.Arc

	createArc := func(ctx SpecContext, raw string) {
		a = arc.Arc{Name: "deployable", Mode: arc.ModeText, Text: newText(raw)}
		Expect(svc.NewWriter(tx).Create(ctx, &a)).To(Succeed())
	}

	retrieveTask := func(ctx SpecContext, key task.Key) task.Task {
		var tsk task.Task
		Expect(taskSvc.NewRetrieve().
			Where(task.MatchKeys(key)).
			Entry(&tsk).
			Exec(ctx, tx)).To(Succeed())
		return tsk
	}

	It("Should create a task on the rack stamped with the arc's hash", func(ctx SpecContext) {
		createArc(ctx, "a -> b")
		tsk := MustSucceed(svc.NewWriter(tx).Deploy(ctx, a.Key, testRack.Key))
		Expect(tsk).ToNot(BeNil())
		Expect(tsk.Rack).To(Equal(testRack.Key))
		Expect(tsk.Type).To(Equal(arc.TaskType))
		Expect(tsk.Name).To(Equal(a.Name))
		Expect(tsk.Config).To(HaveKeyWithValue("arc_key", a.Key.String()))
		Expect(tsk.Config).To(HaveKey("hash"))
		stored := retrieveTask(ctx, tsk.Key)
		Expect(stored.ConfigHash).ToNot(BeEmpty())
	})

	It("Should reuse the task on redeploy to the same rack", func(ctx SpecContext) {
		createArc(ctx, "a -> b")
		w := svc.NewWriter(tx)
		first := MustSucceed(w.Deploy(ctx, a.Key, testRack.Key))
		second := MustSucceed(w.Deploy(ctx, a.Key, testRack.Key))
		Expect(second.Key).To(Equal(first.Key))
		stored := retrieveTask(ctx, first.Key)
		Expect(stored.ConfigHash).To(Equal(retrieveTask(ctx, second.Key).ConfigHash))
	})

	It("Should keep the task's identity on a rack move", func(ctx SpecContext) {
		createArc(ctx, "a -> b")
		other := &rack.Rack{Name: "Other Rack"}
		Expect(rackSvc.NewWriter(tx).Create(ctx, other)).To(Succeed())
		w := svc.NewWriter(tx)
		first := MustSucceed(w.Deploy(ctx, a.Key, testRack.Key))
		moved := MustSucceed(w.Deploy(ctx, a.Key, other.Key))
		Expect(moved.Key).To(Equal(first.Key))
		Expect(retrieveTask(ctx, first.Key).Rack).To(Equal(other.Key))
	})

	It("Should restamp the hash when the program changed", func(ctx SpecContext) {
		createArc(ctx, "a -> b")
		w := svc.NewWriter(tx)
		first := MustSucceed(w.Deploy(ctx, a.Key, testRack.Key))
		firstHash := retrieveTask(ctx, first.Key).ConfigHash
		edited := arc.Arc{Key: a.Key, Name: a.Name, Mode: arc.ModeText, Text: newText("a -> c")}
		Expect(w.Create(ctx, &edited)).To(Succeed())
		second := MustSucceed(w.Deploy(ctx, a.Key, testRack.Key))
		Expect(retrieveTask(ctx, second.Key).ConfigHash).ToNot(Equal(firstHash))
	})

	It("Should undeploy by deleting the task", func(ctx SpecContext) {
		createArc(ctx, "a -> b")
		w := svc.NewWriter(tx)
		tsk := MustSucceed(w.Deploy(ctx, a.Key, testRack.Key))
		Expect(MustSucceed(w.Deploy(ctx, a.Key, 0))).To(BeNil())
		Expect(taskSvc.NewRetrieve().Where(task.MatchKeys(tsk.Key)).Exec(ctx, tx)).
			To(MatchError(query.ErrNotFound))
	})

	It("Should be a no-op to undeploy an arc with no task", func(ctx SpecContext) {
		createArc(ctx, "a -> b")
		Expect(MustSucceed(svc.NewWriter(tx).Deploy(ctx, a.Key, 0))).To(BeNil())
	})

	It("Should reject undeploying a running arc", func(ctx SpecContext) {
		createArc(ctx, "a -> b")
		w := svc.NewWriter(tx)
		tsk := MustSucceed(w.Deploy(ctx, a.Key, testRack.Key))
		stored := retrieveTask(ctx, tsk.Key)
		stat := task.Status{
			Key:     stored.OntologyID().String(),
			Name:    stored.Name,
			Variant: status.VariantSuccess,
			Message: "running",
			Time:    telem.Now(),
			Details: task.NewStatusDetails(stored, true),
		}
		Expect(status.NewWriter[task.StatusDetails](statusSvc, tx).Set(ctx, &stat)).
			To(Succeed())
		Expect(w.Deploy(ctx, a.Key, 0)).Error().
			To(MatchError(validate.ErrValidation))
		Expect(taskSvc.NewRetrieve().Where(task.MatchKeys(tsk.Key)).Exec(ctx, tx)).
			To(Succeed())
	})

	It("Should return not found for a nonexistent arc", func(ctx SpecContext) {
		Expect(svc.NewWriter(tx).Deploy(ctx, uuid.New(), testRack.Key)).Error().
			To(MatchError(query.ErrNotFound))
	})
})
