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
	"context"
	"sync"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
)

type statusCall = func(context.Context, *driver.StatusHandler) error

func replyFirst(ctx context.Context, h *driver.StatusHandler) error {
	return h.Reply(
		ctx,
		"cmd-1",
		status.VariantSuccess,
		"Command executed successfully",
		msgpack.EncodedJSON{"attempts": 3.0},
	)
}

func ackFirst(ctx context.Context, h *driver.StatusHandler) error {
	return h.Ack(ctx, "cmd-1", true)
}

func sendFirst(ctx context.Context, h *driver.StatusHandler) error {
	return h.Send(ctx, "cmd-1", status.VariantSuccess, true, "Started task", "")
}

func stopSecond(ctx context.Context, h *driver.StatusHandler) error {
	return h.Send(ctx, "cmd-2", status.VariantSuccess, false, "Stopped task", "")
}

func warnSecond(ctx context.Context, h *driver.StatusHandler) error {
	return h.Warn(ctx, driver.Warning{
		Message:     "Connection lost",
		Description: "The broker closed the connection.",
	})
}

var _ = Describe("StatusHandler", func() {
	var (
		t       task.Task
		handler *driver.StatusHandler
	)

	BeforeEach(func(ctx SpecContext) {
		t = task.Task{
			Key:        uuid.New(),
			Name:       "handler-test",
			Type:       "mock",
			ConfigHash: "hash-1",
			Rack:       7,
		}
		createTask(ctx, t)
		handler = driver.NewStatusHandler(db, statusSvc, t)
	})

	// setUnreadable stores a status owned by t whose details are not task details, so
	// the handler fails to read it back and every write fails.
	setUnreadable := func(ctx SpecContext) {
		GinkgoHelper()
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return statusSvc.NewWriter(tx).Set(ctx, &status.Status[string]{
				Owner:   new(t.ResourceID()),
				Name:    t.Name,
				Time:    telem.Now(),
				Variant: status.VariantInfo,
				Details: "not task details",
			})
		})).To(Succeed())
	}

	deleteStored := func(ctx SpecContext) {
		GinkgoHelper()
		Expect(statusSvc.NewWriter(nil).DeleteOwned(ctx, t.ResourceID())).To(Succeed())
	}

	retrieve := func(ctx SpecContext) task.Status {
		GinkgoHelper()
		var stat task.Status
		Expect(statusSvc.NewRetrieve[task.StatusDetails]().
			Where(status.MatchOwners[task.StatusDetails](t.ResourceID())).
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
				"Started task",
				"",
			)).To(Succeed())
			stat := retrieve(ctx)
			Expect(stat.Name).To(Equal("handler-test"))
			Expect(stat.Variant).To(Equal(status.VariantSuccess))
			Expect(stat.Message).To(Equal("Started task"))
			Expect(stat.Details.Cmd).To(Equal("cmd-1"))
			Expect(stat.Details.Running).To(BeTrue())
			Expect(stat.Details.ConfigHash).To(Equal("hash-1"))
			Expect(stat.Details.Rack).To(BeEquivalentTo(7))
		})

		It("should clear the description a prior warning set", func(ctx SpecContext) {
			Expect(
				handler.Warn(
					ctx,
					driver.Warning{Message: "Node failed", Description: "details"},
				),
			).
				To(Succeed())
			Expect(handler.Send(
				ctx,
				"cmd-2",
				status.VariantSuccess,
				false,
				"Stopped task",
				"",
			)).To(Succeed())
			stat := retrieve(ctx)
			Expect(stat.Description).To(BeEmpty())
			Expect(stat.Variant).To(Equal(status.VariantSuccess))
		})

		It("should clear the data of a prior warning, which a later one cannot restore",
			func(ctx SpecContext) {
				Expect(handler.Warn(ctx, driver.Warning{
					Message: "Skipped 1 of 1 record",
					Data:    msgpack.EncodedJSON{"stats": "old"},
				})).To(Succeed())
				Expect(handler.Send(
					ctx, "cmd-2", status.VariantSuccess, false, "Stopped task", "",
				)).To(Succeed())
				Expect(retrieve(ctx).Details.Data).To(BeEmpty())
				Expect(handler.Warn(
					ctx,
					driver.Warning{Message: "Node failed", Description: "details"},
				)).To(Succeed())
				Expect(handler.ClearWarning(ctx)).To(Succeed())
				Expect(retrieve(ctx).Details.Data).To(BeEmpty())
			})

		It("should write the description it is given", func(ctx SpecContext) {
			Expect(handler.Send(
				ctx,
				"cmd-3",
				status.VariantError,
				false,
				"Failed to start task",
				"The device is unreachable.",
			)).To(Succeed())
			stat := retrieve(ctx)
			Expect(stat.Message).To(Equal("Failed to start task"))
			Expect(stat.Description).To(Equal("The device is unreachable."))
		})

		It("should keep the name the task was renamed to", func(ctx SpecContext) {
			Expect(handler.Send(
				ctx,
				"cmd-1",
				status.VariantSuccess,
				true,
				"Started task",
				"",
			)).To(Succeed())
			Expect(
				statusSvc.NewWriter(nil).RenameOwned(ctx, t.ResourceID(), "renamed"),
			).
				To(Succeed())
			Expect(handler.Ack(ctx, "cmd-2", true)).To(Succeed())
			Expect(retrieve(ctx).Name).To(Equal("renamed"))
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
					"Started task",
					"",
				)).To(Succeed())
				Expect(
					handler.Warn(
						ctx,
						driver.Warning{
							Message:     "Node failed",
							Description: "div by zero",
						},
					),
				).
					To(Succeed())
				stat := retrieve(ctx)
				Expect(stat.Variant).To(Equal(status.VariantWarning))
				Expect(stat.Message).To(Equal("Node failed"))
				Expect(stat.Description).To(Equal("div by zero"))
				Expect(stat.Details.Running).To(BeTrue())
				Expect(stat.Details.Cmd).To(Equal(driver.NoCommand))
			},
		)
	})

	Describe("Warn deduplication", func() {
		It("should write nothing for a warning equal to the active one",
			func(ctx SpecContext) {
				Expect(
					handler.Warn(
						ctx,
						driver.Warning{
							Message:     "Broker unreachable",
							Description: "dial timeout",
						},
					),
				).
					To(Succeed())
				first := retrieve(ctx).Time
				Expect(
					handler.Warn(
						ctx,
						driver.Warning{
							Message:     "Broker unreachable",
							Description: "dial timeout",
						},
					),
				).
					To(Succeed())
				Expect(retrieve(ctx).Time).To(Equal(first))
			},
		)
		It(
			"should write a warning again after its write failed",
			func(ctx SpecContext) {
				setUnreadable(ctx)
				Expect(
					handler.Warn(
						ctx,
						driver.Warning{
							Message:     "Broker unreachable",
							Description: "dial timeout",
						},
					),
				).
					To(MatchError(ContainSubstring("failed to decode")))
				deleteStored(ctx)
				Expect(
					handler.Warn(
						ctx,
						driver.Warning{
							Message:     "Broker unreachable",
							Description: "dial timeout",
						},
					),
				).
					To(Succeed())
				stat := retrieve(ctx)
				Expect(stat.Variant).To(Equal(status.VariantWarning))
				Expect(stat.Message).To(Equal("Broker unreachable"))
				Expect(stat.Description).To(Equal("dial timeout"))
			},
		)
		It("should write a warning whose description changed", func(ctx SpecContext) {
			Expect(
				handler.Warn(
					ctx,
					driver.Warning{
						Message:     "Broker unreachable",
						Description: "dial timeout",
					},
				),
			).
				To(Succeed())
			Expect(
				handler.Warn(
					ctx,
					driver.Warning{
						Message:     "Broker unreachable",
						Description: "refused",
					},
				),
			).To(Succeed())
			Expect(retrieve(ctx).Description).To(Equal("refused"))
		})
	})

	Describe("ClearWarning", func() {
		It("should restore the status that the warning replaced",
			func(ctx SpecContext) {
				Expect(handler.Send(
					ctx,
					"cmd-1",
					status.VariantError,
					true,
					"Failed to send event",
					"outage",
				)).To(Succeed())
				Expect(
					handler.Warn(
						ctx,
						driver.Warning{
							Message:     "Broker unreachable",
							Description: "dial timeout",
						},
					),
				).
					To(Succeed())
				Expect(
					handler.Warn(
						ctx,
						driver.Warning{
							Message:     "Broker unreachable",
							Description: "refused",
						},
					),
				).
					To(Succeed())
				Expect(handler.ClearWarning(ctx)).To(Succeed())
				stat := retrieve(ctx)
				Expect(stat.Variant).To(Equal(status.VariantError))
				Expect(stat.Message).To(Equal("Failed to send event"))
				Expect(stat.Description).To(Equal("outage"))
				Expect(stat.Details.Running).To(BeTrue())
				Expect(stat.Details.Cmd).To(Equal(driver.NoCommand))
			},
		)
		It(
			"should clear a warning again after its write failed",
			func(ctx SpecContext) {
				Expect(
					handler.Warn(
						ctx,
						driver.Warning{
							Message:     "Broker unreachable",
							Description: "dial timeout",
						},
					),
				).
					To(Succeed())
				setUnreadable(ctx)
				Expect(handler.ClearWarning(ctx)).
					To(MatchError(ContainSubstring("failed to decode")))
				deleteStored(ctx)
				Expect(handler.ClearWarning(ctx)).To(Succeed())
				stat := retrieve(ctx)
				Expect(stat.Variant).To(Equal(status.VariantSuccess))
				Expect(stat.Message).To(Equal("Task configured"))
			},
		)
		It("should write nothing when no warning is active", func(ctx SpecContext) {
			Expect(handler.Send(
				ctx,
				"cmd-1",
				status.VariantError,
				false,
				"Failed to run task",
				"boom",
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
				Expect(
					handler.Warn(
						ctx,
						driver.Warning{
							Message:     "Task degraded",
							Description: "sensor offline",
						},
					),
				).
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
					Owner:   new(t.ResourceID()),
					Name:    t.Name,
					Time:    telem.Now(),
					Variant: status.VariantWarning,
					Message: "Rack unreachable",
					Details: task.StatusDetails{Running: false},
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
					"Started task",
					"",
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

	DescribeTable("should store the status of the last transition",
		func(
			ctx SpecContext,
			first, second statusCall,
			variant status.Variant,
			message, cmd string,
			running bool,
		) {
			gated := &gatedDB{
				DB:      db.KV(),
				held:    make(chan struct{}),
				release: make(chan struct{}),
			}
			handler := driver.NewStatusHandler(gorp.Wrap(gated), statusSvc, t)
			Expect(handler.Send(
				ctx, "cmd-0", status.VariantSuccess, true, "Started task", "",
			)).To(Succeed())
			// Cleanup opens the gate for a spec that fails before the release below.
			release := sync.OnceFunc(func() { close(gated.release) })
			DeferCleanup(release)
			gated.gated.Store(true)
			firstDone, secondDone := make(chan error, 1), make(chan error, 1)
			go func() { firstDone <- first(ctx, handler) }()
			Eventually(gated.held).Should(BeClosed())
			go func() { secondDone <- second(ctx, handler) }()
			Consistently(secondDone).ShouldNot(Receive())
			release()
			Eventually(firstDone).Should(Receive(BeNil()))
			Eventually(secondDone).Should(Receive(BeNil()))
			stat := retrieve(ctx)
			Expect(stat.Variant).To(Equal(variant))
			Expect(stat.Message).To(Equal(message))
			Expect(stat.Details.Cmd).To(Equal(cmd))
			Expect(stat.Details.Running).To(Equal(running))
			Expect(stat.Details.Data).To(BeEmpty())
		},
		Entry("when a stop passes a Reply",
			replyFirst, stopSecond,
			status.VariantSuccess, "Stopped task", "cmd-2", false,
		),
		Entry("when a stop passes an Ack",
			ackFirst, stopSecond,
			status.VariantSuccess, "Stopped task", "cmd-2", false,
		),
		Entry("when a warning passes a Send",
			sendFirst, warnSecond,
			status.VariantWarning, "Connection lost", driver.NoCommand, true,
		),
	)

	Describe("ConfigReporter", func() {
		cfgErr := errors.New("bad config")
		reporter := func() driver.ConfigReporter {
			return driver.ConfigReporter{DB: db, Status: statusSvc}
		}
		It("should answer the command with an error status", func(ctx SpecContext) {
			reporter().Report(ctx, t, "cmd-9", false, cfgErr)
			stat := retrieve(ctx)
			Expect(stat.Variant).To(Equal(status.VariantError))
			Expect(stat.Message).To(Equal("Failed to configure task"))
			Expect(stat.Description).To(Equal("bad config"))
			Expect(stat.Details.Cmd).To(Equal("cmd-9"))
			Expect(stat.Details.Running).To(BeFalse())
		})
		It("should write an error status for a task that starts automatically",
			func(ctx SpecContext) {
				reporter().Report(ctx, t, driver.NoCommand, true, cfgErr)
				stat := retrieve(ctx)
				Expect(stat.Variant).To(Equal(status.VariantError))
				Expect(stat.Message).To(Equal("Failed to configure task"))
				Expect(stat.Description).To(Equal("bad config"))
				Expect(stat.Details.Cmd).To(Equal(driver.NoCommand))
			},
		)
		It("should write no status at boot for a task that does not start",
			func(ctx SpecContext) {
				reporter().Report(ctx, t, driver.NoCommand, false, cfgErr)
				Expect(statusSvc.NewRetrieve[any]().
					Where(status.MatchOwners[any](t.ResourceID())).
					Entry(new(status.Status[any])).
					Exec(ctx, nil)).To(MatchError(query.ErrNotFound))
			},
		)
	})
})
