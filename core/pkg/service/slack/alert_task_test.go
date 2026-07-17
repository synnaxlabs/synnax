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
	slk "github.com/synnaxlabs/synnax/pkg/service/slack"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/errors"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("AlertTask", func() {
	Describe("Config", func() {
		Describe("Validate", func() {
			It("Should require a device", func() {
				cfg := slk.AlertTaskConfig{Channel: "#a", Statuses: []string{"s1"}}
				Expect(cfg.Validate()).To(MatchError(ContainSubstring("device")))
			})

			It("Should require a channel", func() {
				cfg := slk.AlertTaskConfig{Device: "d", Statuses: []string{"s1"}}
				Expect(cfg.Validate()).To(MatchError(ContainSubstring("channel")))
			})

			It("Should require at least one status", func() {
				cfg := slk.AlertTaskConfig{Device: "d", Channel: "#a"}
				Expect(cfg.Validate()).To(MatchError(ContainSubstring("statuses")))
			})

			It("Should succeed with a valid config", func() {
				cfg := slk.AlertTaskConfig{
					Device: "d", Channel: "#a", Statuses: []string{"s1"},
				}
				Expect(cfg.Validate()).To(Succeed())
			})
		})

		Describe("MsgpackEncodedJSON", func() {
			It("Should round-trip all fields", func() {
				cfg := slk.AlertTaskConfig{
					Device:    "dev-1",
					Channel:   "#alerts",
					Statuses:  []string{"s1", "s2"},
					AutoStart: true,
				}
				m := MustSucceed(cfg.MsgpackEncodedJSON())
				var decoded slk.AlertTaskConfig
				Expect(m.Unmarshal(&decoded)).To(Succeed())
				Expect(decoded).To(Equal(cfg))
			})
		})
	})

	Describe("Posting", func() {
		var sender *mockSender

		// start configures and auto-starts a task watching the given statuses, and
		// registers its shutdown.
		start := func(ctx context.Context, channel string, statuses ...string) {
			factory := MustSucceed(slk.NewFactory(slk.FactoryConfig{
				Status: statusSvc, Device: deviceSvc, Sender: sender,
			}))
			cfg := MustSucceed(slk.AlertTaskConfig{
				Device:    createDevice(ctx, uniqueKey(), "xoxb-token"),
				Channel:   channel,
				Statuses:  statuses,
				AutoStart: true,
			}.MsgpackEncodedJSON())
			t := task.Task{
				Key: nextTaskKey(), Name: "Slack", Type: slk.AlertTaskType, Config: cfg,
			}
			tsk := MustSucceed(factory.ConfigureTask(ctx, t))
			DeferCleanup(func() { Expect(tsk.Stop()).To(Succeed()) })
		}

		BeforeEach(func() { sender = newMockSender() })

		It("Should post to the configured channel when a watched status changes",
			func(ctx context.Context) {
				key := uniqueKey()
				start(ctx, "#alerts", key)
				setStatus(ctx, key, status.VariantError, "Pump", "overpressure")
				Eventually(sender.getPosts).Should(HaveLen(1))
				post := sender.getPosts()[0]
				Expect(post.token).To(Equal("xoxb-token"))
				Expect(post.msg.Channel).To(Equal("#alerts"))
				Expect(post.msg.Headline).To(Equal("Pump"))
				Expect(post.msg.Body).To(Equal("overpressure"))
				Expect(post.msg.Emoji).ToNot(BeEmpty())
			})

		DescribeTable("Should post on every variant",
			func(ctx context.Context, variant status.Variant) {
				key := uniqueKey()
				start(ctx, "#alerts", key)
				setStatus(ctx, key, variant, "Name", "msg")
				Eventually(sender.getPosts).Should(HaveLen(1))
			},
			Entry("success", status.VariantSuccess),
			Entry("info", status.VariantInfo),
			Entry("warning", status.VariantWarning),
			Entry("error", status.VariantError),
			Entry("loading", status.VariantLoading),
			Entry("disabled", status.VariantDisabled),
		)

		It("Should not post when an unwatched status changes",
			func(ctx context.Context) {
				start(ctx, "#alerts", uniqueKey())
				setStatus(ctx, uniqueKey(), status.VariantError, "Other", "msg")
				Consistently(sender.callCount, "100ms").Should(BeEquivalentTo(0))
			})

		It("Should stop posting after the task is stopped",
			func(ctx context.Context) {
				key := uniqueKey()
				factory := MustSucceed(slk.NewFactory(slk.FactoryConfig{
					Status: statusSvc, Device: deviceSvc, Sender: sender,
				}))
				cfg := MustSucceed(slk.AlertTaskConfig{
					Device:    createDevice(ctx, uniqueKey(), "xoxb-token"),
					Channel:   "#alerts",
					Statuses:  []string{key},
					AutoStart: true,
				}.MsgpackEncodedJSON())
				t := task.Task{
					Key: nextTaskKey(), Name: "Slack", Type: slk.AlertTaskType, Config: cfg,
				}
				tsk := MustSucceed(factory.ConfigureTask(ctx, t))
				Expect(tsk.Stop()).To(Succeed())
				setStatus(ctx, key, status.VariantError, "Name", "msg")
				Consistently(sender.callCount, "100ms").Should(BeEquivalentTo(0))
			})

		It("Should set the task status to error when the sender fails",
			func(ctx context.Context) {
				sender.setError(errors.New("channel_not_found"))
				key := uniqueKey()
				start(ctx, "#alerts", key)
				setStatus(ctx, key, status.VariantError, "Name", "msg")
				Eventually(func(g Gomega) {
					var stat task.Status
					g.Expect(status.NewRetrieve[task.StatusDetails](statusSvc).
						Where(status.MatchKeys[task.StatusDetails](
							task.OntologyID(currentTaskKey).String())).
						Entry(&stat).Exec(ctx, nil)).To(Succeed())
					g.Expect(stat.Variant).To(BeEquivalentTo("error"))
					g.Expect(stat.Message).To(ContainSubstring("channel_not_found"))
				}).Should(Succeed())
			})
	})
})

// currentTaskKey holds the key of the most recently started task, letting the
// sender-failure spec locate its task status.
var currentTaskKey task.Key

var (
	taskKeyCounter uint64 = 100
	deviceKeyCount uint64
)

func nextTaskKey() task.Key {
	taskKeyCounter++
	currentTaskKey = task.Key(taskKeyCounter)
	return currentTaskKey
}

func uniqueKey() string {
	deviceKeyCount++
	return "k-" + task.Key(deviceKeyCount).String()
}
