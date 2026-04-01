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
	"context"
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	pd "github.com/synnaxlabs/synnax/pkg/service/pagerduty"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	xstatus "github.com/synnaxlabs/x/status"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("AlertTask", func() {
	Describe("Config", func() {
		Describe("MsgpackEncodedJSON", func() {
			It("Should round-trip all fields correctly", func() {
				cfg := pd.AlertTaskConfig{
					RoutingKey: strings.Repeat("x", 32),
					AutoStart:  true,
					Alerts: []pd.AlertConfig{
						{
							Status:               "my-status",
							TreatErrorAsCritical: true,
							Component:            "sensor",
							Group:                "hw",
							Class:                "temp",
							Enabled:              true,
						},
					},
				}
				m := MustSucceed(cfg.MsgpackEncodedJSON())
				var decoded pd.AlertTaskConfig
				Expect(m.Unmarshal(&decoded)).To(Succeed())
				Expect(decoded).To(Equal(cfg))
			})

			It("Should produce a valid map with expected keys", func() {
				cfg := pd.AlertTaskConfig{
					RoutingKey: strings.Repeat("a", 32),
					Alerts:     []pd.AlertConfig{{Status: "s1", Enabled: true}},
				}
				m := MustSucceed(cfg.MsgpackEncodedJSON())
				Expect(m).To(HaveKey("routing_key"))
				Expect(m).To(HaveKey("auto_start"))
				Expect(m).To(HaveKey("alerts"))
			})
		})
	})
	var (
		sender  *mockEventSender
		factory driver.Factory
		ctx     driver.Context
	)

	validConfig := func(alerts ...pd.AlertConfig) pd.AlertTaskConfig {
		return pd.AlertTaskConfig{
			RoutingKey: strings.Repeat("b", 32),
			AutoStart:  false,
			Alerts:     alerts,
		}
	}

	configureAndStart := func(
		specCtx SpecContext,
		cfg pd.AlertTaskConfig,
	) driver.Task {
		t := task.Task{
			Key:    task.NewKey(1, 1),
			Name:   "PagerDuty Test",
			Type:   pd.AlertTaskType,
			Config: MustSucceed(cfg.MsgpackEncodedJSON()),
		}
		tsk := MustSucceed(factory.ConfigureTask(ctx, t))
		Expect(tsk.Exec(specCtx, task.Command{Type: "start"})).To(Succeed())
		return tsk
	}

	setStatus := func(
		specCtx SpecContext,
		key string,
		variant xstatus.Variant,
		message string,
		details any,
	) {
		tx := db.OpenTx()
		defer func() { Expect(tx.Close()).To(Succeed()) }()
		w := status.NewWriter[any](statusSvc, tx)
		Expect(w.Set(specCtx, &status.Status[any]{
			Key:     key,
			Name:    "Test Source",
			Variant: variant,
			Message: message,
			Time:    telem.Now(),
			Details: details,
		})).To(Succeed())
		Expect(tx.Commit(specCtx)).To(Succeed())
	}

	BeforeEach(func(specCtx SpecContext) {
		sender = newMockSender()
		factory = MustSucceed(pd.NewFactory(pd.FactoryConfig{
			Status: statusSvc,
			Sender: sender,
		}))
		ctx = driver.NewContext(specCtx, statusSvc)
	})

	Describe("Exec", func() {
		It("Should return ErrUnsupportedCommand for unknown commands",
			func(specCtx SpecContext) {
				cfg := validConfig(pd.AlertConfig{Status: "s1", Enabled: true})
				t := task.Task{
					Key:    task.NewKey(1, 1),
					Name:   "test",
					Type:   pd.AlertTaskType,
					Config: MustSucceed(cfg.MsgpackEncodedJSON()),
				}
				tsk := MustSucceed(factory.ConfigureTask(ctx, t))
				defer func() { Expect(tsk.Stop()).To(Succeed()) }()
				err := tsk.Exec(specCtx, task.Command{Type: "restart"})
				Expect(err).To(MatchError(driver.ErrUnsupportedCommand))
			},
		)
	})

	Describe("Status Observation", func() {
		It("Should send a trigger event when a watched status changes to error",
			func(specCtx SpecContext) {
				tsk := configureAndStart(specCtx, validConfig(
					pd.AlertConfig{Status: "watched-error", Enabled: true},
				))
				defer func() { Expect(tsk.Stop()).To(Succeed()) }()

				setStatus(specCtx, "watched-error", xstatus.VariantError,
					"Something broke", nil)

				Eventually(func() int { return len(sender.Events()) }).
					WithTimeout(2 * time.Second).
					Should(BeNumerically(">=", 1))

				events := sender.Events()
				Expect(events[0].Action).To(Equal("trigger"))
				Expect(events[0].DedupKey).To(Equal("watched-error"))
				Expect(events[0].RoutingKey).To(Equal(strings.Repeat("b", 32)))
				Expect(events[0].Payload).ToNot(BeNil())
				Expect(events[0].Payload.Summary).To(Equal("Something broke"))
				Expect(events[0].Payload.Source).To(Equal("Test Source"))
				Expect(events[0].Payload.Severity).To(Equal("error"))
			},
		)

		It("Should send a resolve event when a watched status changes to success",
			func(specCtx SpecContext) {
				tsk := configureAndStart(specCtx, validConfig(
					pd.AlertConfig{Status: "watched-resolve", Enabled: true},
				))
				defer func() { Expect(tsk.Stop()).To(Succeed()) }()

				setStatus(specCtx, "watched-resolve", xstatus.VariantSuccess,
					"All good", nil)

				Eventually(func() int { return len(sender.Events()) }).
					WithTimeout(2 * time.Second).
					Should(BeNumerically(">=", 1))

				events := sender.Events()
				Expect(events[0].Action).To(Equal("resolve"))
				Expect(events[0].DedupKey).To(Equal("watched-resolve"))
			},
		)

		It("Should ignore status changes for unwatched keys",
			func(specCtx SpecContext) {
				tsk := configureAndStart(specCtx, validConfig(
					pd.AlertConfig{Status: "watched-only", Enabled: true},
				))
				defer func() { Expect(tsk.Stop()).To(Succeed()) }()

				setStatus(specCtx, "unwatched-key", xstatus.VariantError,
					"Should be ignored", nil)

				Consistently(func() int { return len(sender.Events()) }).
					WithTimeout(500 * time.Millisecond).
					Should(Equal(0))
			},
		)

		It("Should ignore disabled alerts", func(specCtx SpecContext) {
			tsk := configureAndStart(specCtx, validConfig(
				pd.AlertConfig{Status: "disabled-alert", Enabled: false},
				pd.AlertConfig{Status: "enabled-alert", Enabled: true},
			))
			defer func() { Expect(tsk.Stop()).To(Succeed()) }()

			setStatus(specCtx, "disabled-alert", xstatus.VariantError,
				"Should be ignored", nil)

			Consistently(func() int { return len(sender.Events()) }).
				WithTimeout(500 * time.Millisecond).
				Should(Equal(0))
		})

		It("Should skip loading and disabled status variants",
			func(specCtx SpecContext) {
				tsk := configureAndStart(specCtx, validConfig(
					pd.AlertConfig{Status: "variant-skip", Enabled: true},
				))
				defer func() { Expect(tsk.Stop()).To(Succeed()) }()

				setStatus(specCtx, "variant-skip", xstatus.VariantLoading,
					"Loading...", nil)

				Consistently(func() int { return len(sender.Events()) }).
					WithTimeout(500 * time.Millisecond).
					Should(Equal(0))
			},
		)

		It("Should send a trigger event for warning status",
			func(specCtx SpecContext) {
				tsk := configureAndStart(specCtx, validConfig(
					pd.AlertConfig{Status: "watched-warning", Enabled: true},
				))
				defer func() { Expect(tsk.Stop()).To(Succeed()) }()

				setStatus(specCtx, "watched-warning", xstatus.VariantWarning,
					"Watch out", nil)

				Eventually(func() int { return len(sender.Events()) }).
					WithTimeout(2 * time.Second).
					Should(BeNumerically(">=", 1))

				Expect(sender.Events()[0].Payload.Severity).To(Equal("warning"))
			},
		)

		It("Should send a trigger event for info status",
			func(specCtx SpecContext) {
				tsk := configureAndStart(specCtx, validConfig(
					pd.AlertConfig{Status: "watched-info", Enabled: true},
				))
				defer func() { Expect(tsk.Stop()).To(Succeed()) }()

				setStatus(specCtx, "watched-info", xstatus.VariantInfo, "FYI", nil)

				Eventually(func() int { return len(sender.Events()) }).
					WithTimeout(2 * time.Second).
					Should(BeNumerically(">=", 1))

				Expect(sender.Events()[0].Payload.Severity).To(Equal("info"))
			},
		)
	})

	Describe("Severity Mapping", func() {
		It("Should map error to critical when TreatErrorAsCritical is true",
			func(specCtx SpecContext) {
				tsk := configureAndStart(specCtx, validConfig(
					pd.AlertConfig{
						Status:               "critical-error",
						Enabled:              true,
						TreatErrorAsCritical: true,
					},
				))
				defer func() { Expect(tsk.Stop()).To(Succeed()) }()

				setStatus(specCtx, "critical-error", xstatus.VariantError,
					"Critical failure", nil)

				Eventually(func() int { return len(sender.Events()) }).
					WithTimeout(2 * time.Second).
					Should(BeNumerically(">=", 1))

				Expect(sender.Events()[0].Payload.Severity).To(Equal("critical"))
			},
		)

		It("Should map error to error when TreatErrorAsCritical is false",
			func(specCtx SpecContext) {
				tsk := configureAndStart(specCtx, validConfig(
					pd.AlertConfig{
						Status:               "normal-error",
						Enabled:              true,
						TreatErrorAsCritical: false,
					},
				))
				defer func() { Expect(tsk.Stop()).To(Succeed()) }()

				setStatus(specCtx, "normal-error", xstatus.VariantError,
					"Normal failure", nil)

				Eventually(func() int { return len(sender.Events()) }).
					WithTimeout(2 * time.Second).
					Should(BeNumerically(">=", 1))

				Expect(sender.Events()[0].Payload.Severity).To(Equal("error"))
			},
		)
	})

	Describe("Event Payload Mapping", func() {
		It("Should map status fields to PagerDuty event fields correctly",
			func(specCtx SpecContext) {
				tsk := configureAndStart(specCtx, validConfig(
					pd.AlertConfig{
						Status:    "payload-test",
						Enabled:   true,
						Component: "sensor-array",
						Group:     "hardware",
						Class:     "temperature-warning",
					},
				))
				defer func() { Expect(tsk.Stop()).To(Succeed()) }()

				tx := db.OpenTx()
				defer func() { Expect(tx.Close()).To(Succeed()) }()
				w := status.NewWriter[any](statusSvc, tx)
				Expect(w.Set(specCtx, &status.Status[any]{
					Key:         "payload-test",
					Name:        "Temperature Sensor",
					Variant:     xstatus.VariantWarning,
					Message:     "High temperature",
					Description: "Exceeded 80C threshold",
					Time:        telem.Now(),
					Details:     map[string]any{"temp": 85.2},
				})).To(Succeed())
				Expect(tx.Commit(specCtx)).To(Succeed())

				Eventually(func() int { return len(sender.Events()) }).
					WithTimeout(2 * time.Second).
					Should(BeNumerically(">=", 1))

				event := sender.Events()[0]
				Expect(event.DedupKey).To(Equal("payload-test"))
				Expect(event.Payload.Source).To(Equal("Temperature Sensor"))
				Expect(event.Payload.Summary).To(
					ContainSubstring("High temperature"),
				)
				Expect(event.Payload.Summary).To(
					ContainSubstring("Exceeded 80C threshold"),
				)
				Expect(event.Payload.Severity).To(Equal("warning"))
				Expect(event.Payload.Component).To(Equal("sensor-array"))
				Expect(event.Payload.Group).To(Equal("hardware"))
				Expect(event.Payload.Class).To(Equal("temperature-warning"))
				Expect(event.Payload.Details).ToNot(BeNil())
			},
		)

		It("Should use only message as summary when description is empty",
			func(specCtx SpecContext) {
				tsk := configureAndStart(specCtx, validConfig(
					pd.AlertConfig{Status: "no-desc", Enabled: true},
				))
				defer func() { Expect(tsk.Stop()).To(Succeed()) }()

				setStatus(specCtx, "no-desc", xstatus.VariantError, "Simple error", nil)

				Eventually(func() int { return len(sender.Events()) }).
					WithTimeout(2 * time.Second).
					Should(BeNumerically(">=", 1))

				Expect(sender.Events()[0].Payload.Summary).To(
					Equal("Simple error"),
				)
			},
		)
	})

	Describe("Stop", func() {
		It("Should stop observing status changes after stop",
			func(specCtx SpecContext) {
				tsk := configureAndStart(specCtx, validConfig(
					pd.AlertConfig{Status: "stop-test", Enabled: true},
				))
				Expect(tsk.Stop()).To(Succeed())

				setStatus(specCtx, "stop-test", xstatus.VariantError, "After stop", nil)

				Consistently(func() int { return len(sender.Events()) }).
					WithTimeout(500 * time.Millisecond).
					Should(Equal(0))
			},
		)

		It("Should cancel in-flight sendEvent goroutines when the parent context is cancelled",
			func(specCtx SpecContext) {
				sender.SetError(fmt.Errorf("simulated PagerDuty outage"))
				cancelCtx, cancel := context.WithCancel(specCtx)
				defer cancel()
				t := task.Task{
					Key:  task.NewKey(1, 1),
					Name: "PagerDuty Test",
					Type: pd.AlertTaskType,
					Config: MustSucceed(validConfig(
						pd.AlertConfig{Status: "cancel-mid-retry", Enabled: true},
					).MsgpackEncodedJSON()),
				}
				tsk := MustSucceed(factory.ConfigureTask(ctx, t))
				Expect(tsk.Exec(cancelCtx, task.Command{Type: "start"})).To(Succeed())
				defer func() { Expect(tsk.Stop()).To(Succeed()) }()

				setStatus(specCtx, "cancel-mid-retry", xstatus.VariantError,
					"Trigger retry loop", nil)

				Eventually(func() int32 { return sender.SendCallCount() }).
					WithTimeout(2 * time.Second).
					Should(BeNumerically(">=", 1))

				cancel()
				callsBefore := sender.SendCallCount()

				Consistently(func() int32 { return sender.SendCallCount() }).
					WithTimeout(500 * time.Millisecond).
					Should(BeNumerically("<=", callsBefore+1))
			},
		)
	})
})
