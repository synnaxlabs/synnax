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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
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
		handler = driver.NewStatusHandler(db, statusSvc, t)
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

	Describe("Warn deduplication", func() {
		It("should write nothing for a warning equal to the active one",
			func(ctx SpecContext) {
				Expect(handler.Warn(ctx, "Broker unreachable", "dial timeout")).
					To(Succeed())
				first := retrieve(ctx).Time
				Expect(handler.Warn(ctx, "Broker unreachable", "dial timeout")).
					To(Succeed())
				Expect(retrieve(ctx).Time).To(Equal(first))
			},
		)
		It("should write a warning whose description changed", func(ctx SpecContext) {
			Expect(handler.Warn(ctx, "Broker unreachable", "dial timeout")).
				To(Succeed())
			Expect(handler.Warn(ctx, "Broker unreachable", "refused")).To(Succeed())
			Expect(retrieve(ctx).Description).To(Equal("refused"))
		})
	})

	Describe("ClearWarning", func() {
		It("should restore the status that the warning replaced",
			func(ctx SpecContext) {
				Expect(handler.Send(
					ctx,
					"cmd-1",
					status.VariantSuccess,
					true,
					"Task started successfully",
				)).To(Succeed())
				Expect(handler.Warn(ctx, "Broker unreachable", "dial timeout")).
					To(Succeed())
				Expect(handler.Warn(ctx, "Broker unreachable", "refused")).
					To(Succeed())
				Expect(handler.ClearWarning(ctx)).To(Succeed())
				stat := retrieve(ctx)
				Expect(stat.Variant).To(Equal(status.VariantSuccess))
				Expect(stat.Message).To(Equal("Task started successfully"))
				Expect(stat.Description).To(BeEmpty())
				Expect(stat.Details.Running).To(BeTrue())
				Expect(stat.Details.Cmd).To(Equal(driver.NoCommand))
			},
		)
		It("should write nothing when no warning is active", func(ctx SpecContext) {
			Expect(handler.Send(
				ctx,
				"cmd-1",
				status.VariantError,
				false,
				"Task failed",
			)).To(Succeed())
			Expect(handler.ClearWarning(ctx)).To(Succeed())
			stat := retrieve(ctx)
			Expect(stat.Variant).To(Equal(status.VariantError))
			Expect(stat.Details.Cmd).To(Equal("cmd-1"))
		})
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
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return statusSvc.NewWriter(tx).Set(ctx, &task.Status{
					Key:     t.OntologyID().String(),
					Name:    t.Name,
					Time:    telem.Now(),
					Variant: status.VariantWarning,
					Message: "Rack unreachable",
					Details: task.StatusDetails{Task: t.Key, Running: false},
				})
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

	Describe("Reply", func() {
		It("should answer the command with data and keep the running state",
			func(ctx SpecContext) {
				Expect(handler.Send(
					ctx,
					"cmd-1",
					status.VariantSuccess,
					true,
					"Task started successfully",
				)).To(Succeed())
				Expect(handler.Reply(
					ctx,
					"cmd-2",
					status.VariantError,
					"connection refused",
					msgpack.EncodedJSON{"attempts": 3.0},
				)).To(Succeed())
				stat := retrieve(ctx)
				Expect(stat.Variant).To(Equal(status.VariantError))
				Expect(stat.Message).To(Equal("connection refused"))
				Expect(stat.Details.Cmd).To(Equal("cmd-2"))
				Expect(stat.Details.Running).To(BeTrue())
				Expect(stat.Details.Data).To(HaveKeyWithValue("attempts", 3.0))
			},
		)
		It("should not carry the reply into later statuses", func(ctx SpecContext) {
			Expect(handler.Reply(
				ctx,
				"cmd-2",
				status.VariantError,
				"connection refused",
				msgpack.EncodedJSON{"attempts": 3.0},
			)).To(Succeed())
			Expect(handler.Ack(ctx, "cmd-3", false)).To(Succeed())
			stat := retrieve(ctx)
			Expect(stat.Variant).To(Equal(status.VariantSuccess))
			Expect(stat.Message).To(Equal("Task configured"))
			Expect(stat.Details.Data).To(BeEmpty())
		})
	})

	Describe("ReportConfigError", func() {
		cfgErr := errors.New("bad config")
		It("should answer the command with an error status", func(ctx SpecContext) {
			driver.ReportConfigError(
				ctx,
				alamos.Instrumentation{},
				db,
				statusSvc,
				t,
				"cmd-9",
				false,
				cfgErr,
			)
			stat := retrieve(ctx)
			Expect(stat.Variant).To(Equal(status.VariantError))
			Expect(stat.Message).To(Equal("bad config"))
			Expect(stat.Details.Cmd).To(Equal("cmd-9"))
			Expect(stat.Details.Running).To(BeFalse())
		})
		It("should write an error status for a task that starts automatically",
			func(ctx SpecContext) {
				driver.ReportConfigError(
					ctx,
					alamos.Instrumentation{},
					db,
					statusSvc,
					t,
					driver.NoCommand,
					true,
					cfgErr,
				)
				Expect(retrieve(ctx).Variant).To(Equal(status.VariantError))
			},
		)
		It("should write no status at boot for a task that does not start",
			func(ctx SpecContext) {
				driver.ReportConfigError(
					ctx,
					alamos.Instrumentation{},
					db,
					statusSvc,
					t,
					driver.NoCommand,
					false,
					cfgErr,
				)
				Expect(statusSvc.NewRetrieve[task.StatusDetails]().
					Where(status.MatchKeys[task.StatusDetails](
						t.OntologyID().String(),
					)).
					Exec(ctx, nil)).To(MatchError(query.ErrNotFound))
			},
		)
	})
})
