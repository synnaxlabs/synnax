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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Commander", func() {
	var (
		t       task.Task
		handler *driver.StatusHandler
	)

	BeforeEach(func(ctx SpecContext) {
		r := rack.Rack{Name: "commander-rack"}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return rackService.NewWriter(tx).Create(ctx, &r)
		})).To(Succeed())
		t = task.Task{Key: uuid.New(), Name: "commander", Type: "mock", Rack: r.Key}
		createTask(ctx, t)
		handler = driver.NewStatusHandler(db, statusSvc, t)
		Expect(handler.Ack(ctx, driver.NoCommand, true)).To(Succeed())
	})

	open := func(
		exec func(context.Context, task.Command) (msgpack.EncodedJSON, error),
	) *driver.Commander {
		GinkgoHelper()
		c := MustSucceed(driver.NewCommander(driver.CommanderConfig{
			Status:  handler,
			Exec:    exec,
			Message: "Listed files",
		}))
		DeferCleanup(func() { Expect(c.Stop(false)).To(Succeed()) })
		return c
	}

	run := func(ctx context.Context, c *driver.Commander, cmdType string) task.Command {
		GinkgoHelper()
		cmd := task.Command{Task: t.Key, Type: cmdType, Key: uuid.New().String()}
		Expect(c.Exec(ctx, cmd)).To(Succeed())
		return cmd
	}

	answer := func(ctx context.Context, cmd task.Command) task.Status {
		GinkgoHelper()
		var stat task.Status
		Eventually(func(g Gomega) {
			stat = retrieveStatus[task.StatusDetails](ctx, t.ResourceID())
			g.Expect(stat.Details.Cmd).To(Equal(cmd.Key))
		}).Should(Succeed())
		return stat
	}

	Describe("NewCommander", func() {
		It("Should require a status handler, an exec function, and a message", func() {
			Expect(
				driver.NewCommander(driver.CommanderConfig{}),
			).Error().
				To(MatchError(And(
					ContainSubstring("status: must be non-nil"),
					ContainSubstring("exec: must be non-nil"),
					ContainSubstring("message: required"),
				)))
		})
	})

	Describe("Exec", func() {
		It(
			"Should answer a command with its data and the message",
			func(ctx SpecContext) {
				c := open(
					func(context.Context, task.Command) (msgpack.EncodedJSON, error) {
						return msgpack.EncodedJSON{"entries": []any{}}, nil
					},
				)
				stat := answer(ctx, run(ctx, c, "list"))
				Expect(stat.Variant).To(Equal(status.VariantSuccess))
				Expect(stat.Message).To(Equal("Listed files"))
				Expect(
					stat.Details.Data,
				).To(Equal(msgpack.EncodedJSON{"entries": []any{}}))
				Expect(stat.Details.Running).To(BeTrue())
			},
		)

		It("Should answer a failure with its message and description",
			func(ctx SpecContext) {
				c := open(
					func(context.Context, task.Command) (msgpack.EncodedJSON, error) {
						return nil, driver.Failure{
							Message:     "Failed to list files",
							Description: "/x does not exist.",
						}
					},
				)
				stat := answer(ctx, run(ctx, c, "list"))
				Expect(stat.Variant).To(Equal(status.VariantError))
				Expect(stat.Message).To(Equal("Failed to list files"))
				Expect(stat.Description).To(Equal("/x does not exist."))
			})

		It("Should answer a command that exec does not support with an error",
			func(ctx SpecContext) {
				c := open(
					func(context.Context, task.Command) (msgpack.EncodedJSON, error) {
						return nil, driver.ErrUnsupportedCommand
					},
				)
				stat := answer(ctx, run(ctx, c, "restart"))
				Expect(stat.Variant).To(Equal(status.VariantError))
				Expect(stat.Message).To(Equal("Failed to execute command"))
				Expect(stat.Description).
					To(Equal("The task does not accept the 'restart' command."))
			})
	})

	Describe("Stop", func() {
		It("Should cancel the commands and wait for them, with no answer",
			func(ctx SpecContext) {
				started := make(chan struct{})
				c := open(func(
					ctx context.Context,
					_ task.Command,
				) (msgpack.EncodedJSON, error) {
					close(started)
					<-ctx.Done()
					return nil, ctx.Err()
				})
				run(ctx, c, "list")
				Eventually(started).Should(BeClosed())
				Expect(c.Stop(false)).To(Succeed())
				stat := retrieveStatus[task.StatusDetails](ctx, t.ResourceID())
				Expect(stat.Details.Cmd).To(Equal(driver.NoCommand))
			})
	})
})
