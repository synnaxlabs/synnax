// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package pagerduty_test

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	pd "github.com/synnaxlabs/synnax/pkg/service/pagerduty"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/binary"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Factory", func() {
	Describe("Config", func() {
		Describe("Validate", func() {
			It("Should return an error when Status is nil", func() {
				cfg := pd.FactoryConfig{Sender: newMockSender()}
				Expect(cfg.Validate()).To(MatchError(ContainSubstring("status")))
			})

			It("Should return an error when Sender is nil", func() {
				cfg := pd.FactoryConfig{Status: statusSvc}
				Expect(cfg.Validate()).To(MatchError(ContainSubstring("sender")))
			})

			It("Should return an error mentioning both fields when both are nil",
				func() {
					Expect(pd.FactoryConfig{}.Validate()).To(And(
						MatchError(ContainSubstring("status")),
						MatchError(ContainSubstring("sender")),
					))
				},
			)

			It("Should succeed when both Status and Sender are set", func() {
				cfg := pd.FactoryConfig{
					Status: statusSvc,
					Sender: newMockSender(),
				}
				Expect(cfg.Validate()).To(Succeed())
			})
		})

		Describe("Override", func() {
			It("Should override nil fields with the provided values", func() {
				sender := newMockSender()
				cfg := pd.FactoryConfig{}.Override(pd.FactoryConfig{
					Status: statusSvc,
					Sender: sender,
				})
				Expect(cfg.Status).To(Equal(statusSvc))
				Expect(cfg.Sender).To(Equal(sender))
			})

			It("Should preserve existing fields when the override has nil values",
				func() {
					sender := newMockSender()
					cfg := pd.FactoryConfig{
						Status: statusSvc,
						Sender: sender,
					}.Override(pd.FactoryConfig{})
					Expect(cfg.Status).To(Equal(statusSvc))
					Expect(cfg.Sender).To(Equal(sender))
				},
			)
		})
	})

	Describe("New", func() {
		It("Should fail when Status is nil", func() {
			Expect(pd.NewFactory(pd.FactoryConfig{
				Sender: newMockSender(),
			})).Error().To(MatchError(ContainSubstring("status")))
		})

		It("Should use the default event sender when Sender is nil", func() {
			Expect(pd.NewFactory(pd.FactoryConfig{Status: statusSvc})).ToNot(BeNil())
		})
	})

	Describe("Factory", func() {
		var (
			sender  *mockEventSender
			factory driver.Factory
			ctx     driver.Context
		)

		BeforeEach(func(specCtx SpecContext) {
			sender = newMockSender()
			factory = MustSucceed(pd.NewFactory(pd.FactoryConfig{
				Status: statusSvc,
				Sender: sender,
			}))
			ctx = driver.NewContext(specCtx, statusSvc)
		})

		Describe("ConfigureTask", func() {
			It("Should return ErrTaskNotHandled for non-pagerduty types",
				func() {
					t := task.Task{Key: 1, Name: "test", Type: "modbus_read"}
					Expect(factory.ConfigureTask(ctx, t)).Error().
						To(MatchError(driver.ErrTaskNotHandled))
				},
			)

			It("Should return an error for invalid config JSON", func() {
				t := task.Task{
					Key:    1,
					Name:   "test",
					Type:   pd.AlertTaskType,
					Config: binary.MsgpackEncodedJSON{"invalid": func() {}},
				}
				Expect(factory.ConfigureTask(ctx, t)).Error().
					To(MatchError(ContainSubstring("json")))
			})

			It("Should return a validation error for invalid routing key length",
				func() {
					cfg := MustSucceed(pd.AlertTaskConfig{
						RoutingKey: "tooshort",
						Alerts: []pd.AlertConfig{
							{Status: "test-status", Enabled: true},
						},
					}.MsgpackEncodedJSON())
					t := task.Task{
						Key: 1, Name: "test", Type: pd.AlertTaskType,
						Config: cfg,
					}
					Expect(factory.ConfigureTask(ctx, t)).Error().
						To(MatchError(ContainSubstring("routing_key")))
				},
			)

			It("Should return a validation error when no alerts are enabled",
				func() {
					cfg := MustSucceed(pd.AlertTaskConfig{
						RoutingKey: strings.Repeat("a", 32),
						Alerts: []pd.AlertConfig{
							{Status: "test-status", Enabled: false},
						},
					}.MsgpackEncodedJSON())
					t := task.Task{
						Key: 1, Name: "test", Type: pd.AlertTaskType,
						Config: cfg,
					}
					Expect(factory.ConfigureTask(ctx, t)).Error().
						To(MatchError(ContainSubstring("alerts")))
				},
			)

			It("Should configure a task successfully without auto-start",
				func() {
					cfg := MustSucceed(pd.AlertTaskConfig{
						RoutingKey: strings.Repeat("a", 32),
						AutoStart:  false,
						Alerts: []pd.AlertConfig{
							{Status: "test-status", Enabled: true},
						},
					}.MsgpackEncodedJSON())
					t := task.Task{
						Key: 1, Name: "PagerDuty Test",
						Type: pd.AlertTaskType, Config: cfg,
					}
					tsk := MustSucceed(factory.ConfigureTask(ctx, t))
					Expect(tsk).ToNot(BeNil())
					var stat task.Status
					Expect(status.NewRetrieve[task.StatusDetails](statusSvc).
						WhereKeys(task.OntologyID(t.Key).String()).
						Entry(&stat).Exec(ctx, nil)).To(Succeed())
					Expect(stat.Variant).To(BeEquivalentTo("success"))
					Expect(stat.Message).To(Equal("Task configured successfully"))
					Expect(stat.Details.Running).To(BeFalse())
					Expect(tsk.Stop()).To(Succeed())
				},
			)

			It("Should configure and auto-start a task", func() {
				cfg := MustSucceed(pd.AlertTaskConfig{
					RoutingKey: strings.Repeat("a", 32),
					AutoStart:  true,
					Alerts: []pd.AlertConfig{
						{Status: "test-status", Enabled: true},
					},
				}.MsgpackEncodedJSON())
				t := task.Task{
					Key: 1, Name: "PagerDuty Test",
					Type: pd.AlertTaskType, Config: cfg,
				}
				tsk := MustSucceed(factory.ConfigureTask(ctx, t))
				Expect(tsk).ToNot(BeNil())
				var stat task.Status
				Expect(status.NewRetrieve[task.StatusDetails](statusSvc).
					WhereKeys(task.OntologyID(t.Key).String()).
					Entry(&stat).Exec(ctx, nil)).To(Succeed())
				Expect(stat.Variant).To(BeEquivalentTo("success"))
				Expect(stat.Message).To(Equal("Task started successfully"))
				Expect(stat.Details.Running).To(BeTrue())
				Expect(tsk.Stop()).To(Succeed())
			})
		})

		Describe("Name", func() {
			It("Should return pagerduty", func() {
				Expect(factory.Name()).To(Equal("pagerduty"))
			})
		})
	})
})
