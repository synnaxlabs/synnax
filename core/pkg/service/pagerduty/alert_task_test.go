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

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	pd "github.com/synnaxlabs/synnax/pkg/service/pagerduty"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("AlertTask", func() {
	var (
		sender  *mockEventSender
		factory driver.Factory
	)

	validConfig := func(alerts ...pd.Alert) pd.TaskConfig {
		return pd.TaskConfig{
			RoutingKey: strings.Repeat("b", 32),
			Alerts:     alerts,
		}
	}

	configureAndStart := func(
		ctx context.Context,
		cfg pd.TaskConfig,
	) driver.Task {
		t := task.Task{
			Key:    uuid.New(),
			Name:   "PagerDuty Test",
			Type:   pd.AlertTaskType,
			Config: encodeConfig(cfg),
		}
		tsk := MustSucceed(factory.ConfigureTask(ctx, t, "cmd-1"))
		Expect(tsk.Exec(ctx, task.Command{Type: "start"})).To(Succeed())
		return tsk
	}

	setStatus := func(
		ctx context.Context,
		key status.Key,
		variant status.Variant,
		message string,
		details any,
	) {
		tx := db.OpenTx()
		defer func() { Expect(tx.Close()).To(Succeed()) }()
		w := statusSvc.NewWriter[any](tx)
		Expect(w.Set(ctx, &status.Status[any]{
			Key:     key,
			Name:    "Test Source",
			Variant: variant,
			Message: message,
			Time:    telem.Now(),
			Details: details,
		})).To(Succeed())
		Expect(tx.Commit(ctx)).To(Succeed())
	}

	BeforeEach(func() {
		sender = newMockSender()
		factory = MustSucceed(pd.NewFactory(pd.FactoryConfig{
			Status: statusSvc,
			Sender: sender,
		}))
	})

	Describe("Exec", func() {
		It("Should return ErrUnsupportedCommand for unknown commands",
			func(ctx context.Context) {
				cfg := validConfig(pd.Alert{Status: "s1"})
				t := task.Task{
					Key:    uuid.New(),
					Name:   "test",
					Type:   pd.AlertTaskType,
					Config: encodeConfig(cfg),
				}
				tsk := MustSucceed(factory.ConfigureTask(ctx, t, "cmd-1"))
				defer func() { Expect(tsk.Stop(true)).To(Succeed()) }()
				err := tsk.Exec(ctx, task.Command{Type: "restart"})
				Expect(err).To(MatchError(driver.ErrUnsupportedCommand))
			},
		)

		DescribeTable("Should acknowledge a command that needs no work",
			func(ctx context.Context, cmdType string, running bool) {
				t := task.Task{
					Key:    uuid.New(),
					Name:   "PagerDuty Test",
					Type:   pd.AlertTaskType,
					Config: encodeConfig(validConfig(pd.Alert{Status: "s1"})),
				}
				tsk := MustSucceed(factory.ConfigureTask(ctx, t, "cmd-1"))
				defer func() { Expect(tsk.Stop(false)).To(Succeed()) }()
				if running {
					Expect(tsk.Exec(ctx, task.Command{
						Type: "start",
						Key:  "cmd-first",
					})).To(Succeed())
				}
				Expect(tsk.Exec(ctx, task.Command{
					Type: cmdType,
					Key:  "cmd-again",
				})).To(Succeed())
				var stat task.Status
				Expect(statusSvc.NewRetrieve[task.StatusDetails]().
					Where(status.MatchKeys[task.StatusDetails](t.OntologyID().String())).
					Entry(&stat).Exec(ctx, nil)).To(Succeed())
				Expect(stat.Details.Cmd).To(Equal("cmd-again"))
				Expect(stat.Details.Running).To(Equal(running))
			},
			Entry("start on a running task", "start", true),
			Entry("stop on a stopped task", "stop", false),
		)
	})

	Describe("Status Observation", func() {
		It("Should send a trigger event when a watched status changes to error",
			func(ctx context.Context) {
				tsk := configureAndStart(ctx, validConfig(
					pd.Alert{Status: "watched-error"},
				))
				defer func() { Expect(tsk.Stop(true)).To(Succeed()) }()

				setStatus(ctx, "watched-error", status.VariantError,
					"Something broke", nil)

				Eventually(func() int { return len(sender.getEvents()) }).
					WithTimeout(2 * time.Second).
					Should(BeNumerically(">=", 1))

				events := sender.getEvents()
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
			func(ctx context.Context) {
				tsk := configureAndStart(ctx, validConfig(
					pd.Alert{Status: "watched-resolve"},
				))
				defer func() { Expect(tsk.Stop(true)).To(Succeed()) }()

				setStatus(ctx, "watched-resolve", status.VariantSuccess,
					"All good", nil)

				Eventually(func() int { return len(sender.getEvents()) }).
					WithTimeout(2 * time.Second).
					Should(BeNumerically(">=", 1))

				events := sender.getEvents()
				Expect(events[0].Action).To(Equal("resolve"))
				Expect(events[0].DedupKey).To(Equal("watched-resolve"))
			},
		)

		It("Should ignore status changes for unwatched keys",
			func(ctx context.Context) {
				tsk := configureAndStart(ctx, validConfig(
					pd.Alert{Status: "watched-only"},
				))
				defer func() { Expect(tsk.Stop(true)).To(Succeed()) }()

				setStatus(ctx, "unwatched-key", status.VariantError,
					"Should be ignored", nil)

				Consistently(func() int { return len(sender.getEvents()) }).
					WithTimeout(500 * time.Millisecond).
					Should(Equal(0))
			},
		)

		It("Should ignore disabled alerts", func(ctx context.Context) {
			tsk := configureAndStart(ctx, validConfig(
				pd.Alert{Status: "disabled-alert", Disabled: true},
				pd.Alert{Status: "enabled-alert"},
			))
			defer func() { Expect(tsk.Stop(true)).To(Succeed()) }()

			setStatus(ctx, "disabled-alert", status.VariantError,
				"Should be ignored", nil)

			Consistently(func() int { return len(sender.getEvents()) }).
				WithTimeout(500 * time.Millisecond).
				Should(Equal(0))
		})

		It("Should skip loading and disabled status variants",
			func(ctx context.Context) {
				tsk := configureAndStart(ctx, validConfig(
					pd.Alert{Status: "variant-skip"},
				))
				defer func() { Expect(tsk.Stop(true)).To(Succeed()) }()

				setStatus(ctx, "variant-skip", status.VariantLoading,
					"Loading...", nil)

				Consistently(func() int { return len(sender.getEvents()) }).
					WithTimeout(500 * time.Millisecond).
					Should(Equal(0))
			},
		)

		It("Should send a trigger event for warning status",
			func(ctx context.Context) {
				tsk := configureAndStart(ctx, validConfig(
					pd.Alert{Status: "watched-warning"},
				))
				defer func() { Expect(tsk.Stop(true)).To(Succeed()) }()

				setStatus(ctx, "watched-warning", status.VariantWarning,
					"Watch out", nil)

				Eventually(func() int { return len(sender.getEvents()) }).
					WithTimeout(2 * time.Second).
					Should(BeNumerically(">=", 1))

				Expect(sender.getEvents()[0].Payload.Severity).To(Equal("warning"))
			},
		)

		It("Should send a trigger event for info status",
			func(ctx context.Context) {
				tsk := configureAndStart(ctx, validConfig(
					pd.Alert{Status: "watched-info"},
				))
				defer func() { Expect(tsk.Stop(true)).To(Succeed()) }()

				setStatus(ctx, "watched-info", status.VariantInfo, "FYI", nil)

				Eventually(func() int { return len(sender.getEvents()) }).
					WithTimeout(2 * time.Second).
					Should(BeNumerically(">=", 1))

				Expect(sender.getEvents()[0].Payload.Severity).To(Equal("info"))
			},
		)
	})

	Describe("Severity Mapping", func() {
		It("Should map error to critical when ErrorsCritical is true",
			func(ctx context.Context) {
				tsk := configureAndStart(ctx, validConfig(
					pd.Alert{
						Status:         "critical-error",
						ErrorsCritical: true,
					},
				))
				defer func() { Expect(tsk.Stop(true)).To(Succeed()) }()

				setStatus(ctx, "critical-error", status.VariantError,
					"Critical failure", nil)

				Eventually(func() int { return len(sender.getEvents()) }).
					WithTimeout(2 * time.Second).
					Should(BeNumerically(">=", 1))

				Expect(sender.getEvents()[0].Payload.Severity).To(Equal("critical"))
			},
		)

		It("Should map error to error when ErrorsCritical is false",
			func(ctx context.Context) {
				tsk := configureAndStart(ctx, validConfig(
					pd.Alert{
						Status:         "normal-error",
						ErrorsCritical: false,
					},
				))
				defer func() { Expect(tsk.Stop(true)).To(Succeed()) }()

				setStatus(ctx, "normal-error", status.VariantError,
					"Normal failure", nil)

				Eventually(func() int { return len(sender.getEvents()) }).
					WithTimeout(2 * time.Second).
					Should(BeNumerically(">=", 1))

				Expect(sender.getEvents()[0].Payload.Severity).To(Equal("error"))
			},
		)
	})

	Describe("Event Payload Mapping", func() {
		It("Should map status fields to PagerDuty event fields correctly",
			func(ctx context.Context) {
				tsk := configureAndStart(ctx, validConfig(
					pd.Alert{
						Status:    "payload-test",
						Component: "sensor-array",
						Group:     "hardware",
						Class:     "temperature-warning",
					},
				))
				defer func() { Expect(tsk.Stop(true)).To(Succeed()) }()

				tx := db.OpenTx()
				defer func() { Expect(tx.Close()).To(Succeed()) }()
				w := statusSvc.NewWriter[any](tx)
				Expect(w.Set(ctx, &status.Status[any]{
					Key:         "payload-test",
					Name:        "Temperature Sensor",
					Variant:     status.VariantWarning,
					Message:     "High temperature",
					Description: "Exceeded 80C threshold",
					Time:        telem.Now(),
					Details:     map[string]any{"temp": 85.2},
				})).To(Succeed())
				Expect(tx.Commit(ctx)).To(Succeed())

				Eventually(func() int { return len(sender.getEvents()) }).
					WithTimeout(2 * time.Second).
					Should(BeNumerically(">=", 1))

				event := sender.getEvents()[0]
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
			func(ctx context.Context) {
				tsk := configureAndStart(ctx, validConfig(
					pd.Alert{Status: "no-desc"},
				))
				defer func() { Expect(tsk.Stop(true)).To(Succeed()) }()

				setStatus(ctx, "no-desc", status.VariantError, "Simple error", nil)

				Eventually(func() int { return len(sender.getEvents()) }).
					WithTimeout(2 * time.Second).
					Should(BeNumerically(">=", 1))

				Expect(sender.getEvents()[0].Payload.Summary).To(
					Equal("Simple error"),
				)
			},
		)
	})

	Describe("Stop", func() {
		It("Should stop observing status changes after stop",
			func(ctx context.Context) {
				tsk := configureAndStart(ctx, validConfig(
					pd.Alert{Status: "stop-test"},
				))
				Expect(tsk.Stop(true)).To(Succeed())

				setStatus(ctx, "stop-test", status.VariantError, "After stop", nil)

				Consistently(func() int { return len(sender.getEvents()) }).
					WithTimeout(500 * time.Millisecond).
					Should(Equal(0))
			},
		)

		It("Should set error status when sendEvent fails",
			func(ctx context.Context) {
				sender.setError(fmt.Errorf("simulated PagerDuty outage"))
				tsk := configureAndStart(ctx, validConfig(
					pd.Alert{Status: "send-failure"},
				))
				defer func() { Expect(tsk.Stop(true)).To(Succeed()) }()

				setStatus(ctx, "send-failure", status.VariantError,
					"Trigger send", nil)

				Eventually(func() int32 { return sender.sendCallCount() }).
					WithTimeout(2 * time.Second).
					Should(BeNumerically(">=", 1))
			},
		)
	})
})
