// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package slack_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	slk "github.com/synnaxlabs/synnax/pkg/service/slack"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Factory", func() {
	Describe("Config", func() {
		Describe("Validate", func() {
			It("Should return an error when Status is nil", func() {
				cfg := slk.FactoryConfig{Device: deviceSvc, Sender: newMockSender()}
				Expect(cfg.Validate()).To(MatchError(ContainSubstring("status")))
			})

			It("Should return an error when Device is nil", func() {
				cfg := slk.FactoryConfig{Status: statusSvc, Sender: newMockSender()}
				Expect(cfg.Validate()).To(MatchError(ContainSubstring("device")))
			})

			It("Should return an error when Sender is nil", func() {
				cfg := slk.FactoryConfig{Status: statusSvc, Device: deviceSvc}
				Expect(cfg.Validate()).To(MatchError(ContainSubstring("sender")))
			})

			It("Should succeed when all required fields are set", func() {
				cfg := slk.FactoryConfig{
					Status: statusSvc, Device: deviceSvc, Sender: newMockSender(),
				}
				Expect(cfg.Validate()).To(Succeed())
			})
		})

		Describe("Override", func() {
			It("Should override nil fields with the provided values", func() {
				sender := newMockSender()
				cfg := slk.FactoryConfig{}.Override(slk.FactoryConfig{
					Status: statusSvc, Device: deviceSvc, Sender: sender,
				})
				Expect(cfg.Status).To(Equal(statusSvc))
				Expect(cfg.Device).To(Equal(deviceSvc))
				Expect(cfg.Sender).To(Equal(sender))
			})
		})
	})

	Describe("New", func() {
		It("Should fail when Device is nil", func() {
			Expect(slk.NewFactory(slk.FactoryConfig{
				Status: statusSvc, Sender: newMockSender(),
			})).Error().To(MatchError(ContainSubstring("device")))
		})

		It("Should use the default sender when Sender is nil", func() {
			Expect(slk.NewFactory(slk.FactoryConfig{
				Status: statusSvc, Device: deviceSvc,
			})).ToNot(BeNil())
		})
	})

	Describe("Factory", func() {
		var (
			sender  *mockSender
			factory driver.Factory
		)

		BeforeEach(func() {
			sender = newMockSender()
			factory = MustSucceed(slk.NewFactory(slk.FactoryConfig{
				Status: statusSvc, Device: deviceSvc, Sender: sender,
			}))
		})

		Describe("ConfigureTask", func() {
			It("Should return ErrTaskNotHandled for non-slack types",
				func(ctx context.Context) {
					t := task.Task{Key: 1, Name: "test", Type: "modbus_read"}
					Expect(factory.ConfigureTask(ctx, t)).Error().
						To(MatchError(driver.ErrTaskNotHandled))
				})

			It("Should return an error for invalid config JSON",
				func(ctx context.Context) {
					t := task.Task{
						Key: 2, Name: "test", Type: slk.AlertTaskType,
						Config: msgpack.EncodedJSON{"invalid": func() {}},
					}
					Expect(factory.ConfigureTask(ctx, t)).Error().
						To(MatchError(ContainSubstring("json")))
				})

			It("Should return a validation error when the channel is missing",
				func(ctx context.Context) {
					cfg := MustSucceed(slk.AlertTaskConfig{
						Device:   "dev",
						Statuses: []string{"s1"},
					}.MsgpackEncodedJSON())
					t := task.Task{
						Key: 3, Name: "test", Type: slk.AlertTaskType, Config: cfg,
					}
					Expect(factory.ConfigureTask(ctx, t)).Error().
						To(MatchError(ContainSubstring("channel")))
				})

			It("Should configure a task without auto-start",
				func(ctx context.Context) {
					cfg := MustSucceed(slk.AlertTaskConfig{
						Device:   createDevice(ctx, "dev-no-start", "xoxb-token"),
						Channel:  "#alerts",
						Statuses: []string{"s1"},
					}.MsgpackEncodedJSON())
					t := task.Task{
						Key: 4, Name: "Slack Test", Type: slk.AlertTaskType, Config: cfg,
					}
					tsk := MustSucceed(factory.ConfigureTask(ctx, t))
					Expect(tsk).ToNot(BeNil())
					Expect(taskStatus(ctx, t.Key).Message).
						To(Equal("Task configured successfully"))
					Expect(tsk.Stop()).To(Succeed())
				})

			It("Should configure and auto-start a task, resolving the device token",
				func(ctx context.Context) {
					cfg := MustSucceed(slk.AlertTaskConfig{
						Device:    createDevice(ctx, "dev-start", "xoxb-token"),
						Channel:   "#alerts",
						Statuses:  []string{"s1"},
						AutoStart: true,
					}.MsgpackEncodedJSON())
					t := task.Task{
						Key: 5, Name: "Slack Test", Type: slk.AlertTaskType, Config: cfg,
					}
					tsk := MustSucceed(factory.ConfigureTask(ctx, t))
					stat := taskStatus(ctx, t.Key)
					Expect(stat.Variant).To(BeEquivalentTo("success"))
					Expect(stat.Message).To(Equal("Task started successfully"))
					Expect(stat.Details.Running).To(BeTrue())
					Expect(tsk.Stop()).To(Succeed())
				})

			It("Should fail to start when the device has no bot token",
				func(ctx context.Context) {
					cfg := MustSucceed(slk.AlertTaskConfig{
						Device:    createDevice(ctx, "dev-empty", ""),
						Channel:   "#alerts",
						Statuses:  []string{"s1"},
						AutoStart: true,
					}.MsgpackEncodedJSON())
					t := task.Task{
						Key: 6, Name: "Slack Test", Type: slk.AlertTaskType, Config: cfg,
					}
					Expect(factory.ConfigureTask(ctx, t)).Error().
						To(MatchError(ContainSubstring("no bot token")))
				})
		})

		Describe("Name", func() {
			It("Should return slack", func() {
				Expect(factory.Name()).To(Equal("slack"))
			})
		})
	})
})

// taskStatus retrieves the current status of the task with the given key.
func taskStatus(ctx context.Context, key task.Key) task.Status {
	var stat task.Status
	Expect(status.NewRetrieve[task.StatusDetails](statusSvc).
		Where(status.MatchKeys[task.StatusDetails](task.OntologyID(key).String())).
		Entry(&stat).Exec(ctx, nil)).To(Succeed())
	return stat
}
