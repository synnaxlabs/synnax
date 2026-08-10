// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package driver_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("AckCurrentStatus", func() {
	var t task.Task

	BeforeEach(func() {
		t = task.Task{Key: uuid.New(), Name: "ack-test", Type: "mock"}
	})

	retrieve := func(ctx SpecContext) task.Status {
		GinkgoHelper()
		var stat task.Status
		Expect(status.NewRetrieve[task.StatusDetails](statusSvc).
			Where(status.MatchKeys[task.StatusDetails](t.OntologyID().String())).
			Entry(&stat).Exec(ctx, nil)).To(Succeed())
		return stat
	}

	It("should attribute the task's current status to the command",
		func(ctx SpecContext) {
			Expect(status.NewWriter[task.StatusDetails](statusSvc, nil).
				Set(ctx, &task.Status{
					Key:     t.OntologyID().String(),
					Name:    t.Name,
					Time:    telem.Now(),
					Variant: status.VariantWarning,
					Message: "Task degraded",
					Details: task.NewStatusDetails(t, true),
				})).To(Succeed())

			Expect(driver.AckCurrentStatus(
				ctx,
				statusSvc,
				t,
				"cmd-1",
				true,
			)).To(Succeed())

			stat := retrieve(ctx)
			Expect(stat.Details.Cmd).To(Equal("cmd-1"))
			// The answer is the task's real state, not a synthesized success.
			Expect(stat.Variant).To(Equal(status.VariantWarning))
			Expect(stat.Message).To(Equal("Task degraded"))
			Expect(stat.Details.Running).To(BeTrue())
		},
	)

	It("should correct facts another writer left stale", func(ctx SpecContext) {
		t.ConfigHash = "hash-1"
		t.Rack = 7
		// The core blanks these for every task on a rack it thinks is
		// unreachable, without stopping the live instance.
		Expect(status.NewWriter[task.StatusDetails](statusSvc, nil).
			Set(ctx, &task.Status{
				Key:     t.OntologyID().String(),
				Name:    t.Name,
				Time:    telem.Now(),
				Variant: status.VariantWarning,
				Message: "Rack unreachable",
				Details: task.StatusDetails{Task: t.Key, Running: false},
			})).To(Succeed())

		Expect(driver.AckCurrentStatus(ctx, statusSvc, t, "cmd-3", true)).To(Succeed())

		stat := retrieve(ctx)
		Expect(stat.Details.Running).To(BeTrue())
		Expect(stat.Details.ConfigHash).To(Equal("hash-1"))
		Expect(stat.Details.Rack).To(BeEquivalentTo(7))
		Expect(stat.Message).To(Equal("Rack unreachable"))
	})

	DescribeTable("should seed a status when the task has never written one",
		func(ctx SpecContext, running bool) {
			t.Key = uuid.New()
			Expect(driver.AckCurrentStatus(
				ctx,
				statusSvc,
				t,
				"cmd-2",
				running,
			)).To(Succeed())
			stat := retrieve(ctx)
			Expect(stat.Details.Cmd).To(Equal("cmd-2"))
			Expect(stat.Variant).To(Equal(status.VariantSuccess))
			Expect(stat.Details.Running).To(Equal(running))
		},
		Entry("running", true),
		Entry("stopped", false),
	)
})
