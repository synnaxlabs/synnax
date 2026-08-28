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

var _ = Describe("StatusHandler", func() {
	var (
		t       task.Task
		handler *driver.StatusHandler
	)

	BeforeEach(func() {
		t = task.Task{
			Key:        uuid.New(),
			Name:       "handler-test",
			Type:       "mock",
			ConfigHash: "hash-1",
			Rack:       7,
		}
		handler = driver.NewStatusHandler(statusSvc, t)
	})

	retrieve := func(ctx SpecContext) task.Status {
		GinkgoHelper()
		var stat task.Status
		Expect(statusSvc.NewRetrieve[task.StatusDetails]().
			Where(status.MatchKeys[task.StatusDetails](t.OntologyID().String())).
			Entry(&stat).Exec(ctx, nil)).To(Succeed())
		return stat
	}

	Describe("Send", func() {
		It("should write the transition with the task's facts", func(ctx SpecContext) {
			Expect(handler.Send(
				ctx,
				"cmd-1",
				status.VariantSuccess,
				true,
				"Task started successfully",
			)).To(Succeed())
			stat := retrieve(ctx)
			Expect(stat.Name).To(Equal("handler-test"))
			Expect(stat.Variant).To(Equal(status.VariantSuccess))
			Expect(stat.Message).To(Equal("Task started successfully"))
			Expect(stat.Details.Cmd).To(Equal("cmd-1"))
			Expect(stat.Details.Running).To(BeTrue())
			Expect(stat.Details.ConfigHash).To(Equal("hash-1"))
			Expect(stat.Details.Rack).To(BeEquivalentTo(7))
		})

		It("should clear the description a prior warning set", func(ctx SpecContext) {
			Expect(handler.Warn(ctx, "Runtime error in stage", "details")).To(Succeed())
			Expect(handler.Send(
				ctx,
				"cmd-2",
				status.VariantSuccess,
				false,
				"Task stopped successfully",
			)).To(Succeed())
			stat := retrieve(ctx)
			Expect(stat.Description).To(BeEmpty())
			Expect(stat.Variant).To(Equal(status.VariantSuccess))
		})
	})

	Describe("Warn", func() {
		It("should keep the running state and carry the description",
			func(ctx SpecContext) {
				Expect(handler.Send(
					ctx,
					"cmd-1",
					status.VariantSuccess,
					true,
					"Task started successfully",
				)).To(Succeed())
				Expect(handler.Warn(ctx, "Runtime error in stage", "div by zero")).
					To(Succeed())
				stat := retrieve(ctx)
				Expect(stat.Variant).To(Equal(status.VariantWarning))
				Expect(stat.Message).To(Equal("Runtime error in stage"))
				Expect(stat.Description).To(Equal("div by zero"))
				Expect(stat.Details.Running).To(BeTrue())
				Expect(stat.Details.Cmd).To(Equal(driver.NoCommand))
			},
		)
	})

	Describe("Ack", func() {
		It("should attribute the current status to the command unchanged",
			func(ctx SpecContext) {
				Expect(handler.Warn(ctx, "Task degraded", "sensor offline")).
					To(Succeed())
				Expect(handler.Ack(ctx, "cmd-3", true)).To(Succeed())
				stat := retrieve(ctx)
				Expect(stat.Details.Cmd).To(Equal("cmd-3"))
				// The answer is the task's real state, not a synthesized success.
				Expect(stat.Variant).To(Equal(status.VariantWarning))
				Expect(stat.Message).To(Equal("Task degraded"))
				Expect(stat.Description).To(Equal("sensor offline"))
				Expect(stat.Details.Running).To(BeTrue())
			},
		)

		It("should correct facts another writer left stale", func(ctx SpecContext) {
			// The core blanks these for every task on a rack it thinks is
			// unreachable, without stopping the live instance.
			Expect(statusSvc.NewWriter[task.StatusDetails](nil).
				Set(ctx, &task.Status{
					Key:     t.OntologyID().String(),
					Name:    t.Name,
					Time:    telem.Now(),
					Variant: status.VariantWarning,
					Message: "Rack unreachable",
					Details: task.StatusDetails{Task: t.Key, Running: false},
				})).To(Succeed())
			Expect(handler.Ack(ctx, "cmd-4", true)).To(Succeed())
			stat := retrieve(ctx)
			Expect(stat.Details.Running).To(BeTrue())
			Expect(stat.Details.ConfigHash).To(Equal("hash-1"))
			Expect(stat.Details.Rack).To(BeEquivalentTo(7))
		})

		DescribeTable("should answer with the seeded status before any send",
			func(ctx SpecContext, running bool) {
				Expect(handler.Ack(ctx, "cmd-5", running)).To(Succeed())
				stat := retrieve(ctx)
				Expect(stat.Details.Cmd).To(Equal("cmd-5"))
				Expect(stat.Variant).To(Equal(status.VariantSuccess))
				Expect(stat.Message).To(Equal("Task configured"))
				Expect(stat.Details.Running).To(Equal(running))
			},
			Entry("running", true),
			Entry("stopped", false),
		)
	})
})
