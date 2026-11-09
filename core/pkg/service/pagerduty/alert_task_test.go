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
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	pd "github.com/synnaxlabs/synnax/pkg/service/pagerduty"
	"github.com/synnaxlabs/synnax/pkg/service/secret"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("AlertTask", func() {
	var (
		sender     *mockEventSender
		factory    driver.Factory
		routingKey secret.Key
	)

	validConfig := func(alerts ...pd.Alert) pd.TaskConfig {
		return pd.TaskConfig{RoutingKey: &routingKey, Alerts: alerts}
	}

	configureAndStart := func(
		ctx context.Context,
		cfg pd.TaskConfig,
	) driver.Task {
		GinkgoHelper()
		t := storeTask(ctx, "PagerDuty Test", cfg)
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
		GinkgoHelper()
		tx := db.OpenTx()
		defer func() { Expect(tx.Close()).To(Succeed()) }()
		w := statusSvc.NewWriter(tx)
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

	BeforeEach(func(ctx SpecContext) {
		sender = newMockSender()
		factory = MustSucceed(pd.NewFactory(pd.FactoryConfig{
			DB:     db,
			Status: statusSvc,
			Secret: secretSvc,
			Sender: sender,
		}))
		routingKey = createRoutingKey(ctx, strings.Repeat("b", 32))
	})

	Describe("Exec", func() {
		It("Should return ErrUnsupportedCommand for unknown commands",
			func(ctx context.Context) {
				t := storeTask(ctx, "test", validConfig(pd.Alert{Status: "s1"}))
				tsk := MustSucceed(factory.ConfigureTask(ctx, t, "cmd-1"))
				defer func() { Expect(tsk.Stop(true)).To(Succeed()) }()
				Expect(
					tsk.Exec(ctx, task.Command{Type: "restart"}),
				).To(MatchError(driver.ErrUnsupportedCommand))
			},
		)

		DescribeTable("Should acknowledge a command that needs no work",
			func(ctx context.Context, cmdType string, running bool) {
				t := storeTask(
					ctx,
					"PagerDuty Test",
					validConfig(pd.Alert{Status: "s1"}),
				)
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
					Where(status.MatchOwners[task.StatusDetails](t.ResourceID())).
					Entry(&stat).Exec(ctx, db)).To(Succeed())
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

		It("Should send one event for each of two quick changes, in order",
			func(ctx context.Context) {
				tsk := configureAndStart(ctx, validConfig(
					pd.Alert{Status: "quick-blocker"},
					pd.Alert{Status: "quick-flap"},
				))
				defer func() { Expect(tsk.Stop(true)).To(Succeed()) }()
				release := sender.hold()
				setStatus(ctx, "quick-blocker", status.VariantError, "Busy", nil)
				setStatus(ctx, "quick-flap", status.VariantError, "Broke", nil)
				setStatus(ctx, "quick-flap", status.VariantSuccess, "Fixed", nil)
				release()
				Eventually(func() int { return len(sender.getEvents()) }).
					WithTimeout(2 * time.Second).
					Should(Equal(3))
				Expect(sender.getEvents()).To(HaveExactElements(
					HaveField("DedupKey", "quick-blocker"),
					And(
						HaveField("DedupKey", "quick-flap"),
						HaveField("Action", "trigger"),
					),
					And(
						HaveField("DedupKey", "quick-flap"),
						HaveField("Action", "resolve"),
					),
				))
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

	Describe("Routing key", func() {
		It("Should keep the value it started with until the next start",
			func(ctx context.Context) {
				tsk := configureAndStart(ctx, validConfig(
					pd.Alert{Status: "replaced-key"},
				))
				defer func() { Expect(tsk.Stop(true)).To(Succeed()) }()
				replaceRoutingKey(ctx, routingKey, strings.Repeat("c", 32))

				setStatus(ctx, "replaced-key", status.VariantError, "First", nil)
				Eventually(func() int { return len(sender.getEvents()) }).
					WithTimeout(2 * time.Second).
					Should(Equal(1))

				Expect(tsk.Exec(ctx, task.Command{Type: "stop"})).To(Succeed())
				Expect(tsk.Exec(ctx, task.Command{Type: "start"})).To(Succeed())
				setStatus(ctx, "replaced-key", status.VariantSuccess, "Second", nil)
				Eventually(func() int { return len(sender.getEvents()) }).
					WithTimeout(2 * time.Second).
					Should(Equal(2))

				events := sender.getEvents()
				Expect(events[0].RoutingKey).To(Equal(strings.Repeat("b", 32)))
				Expect(events[1].RoutingKey).To(Equal(strings.Repeat("c", 32)))
			},
		)

		It("Should keep sending after the saved config picks another secret",
			func(ctx context.Context) {
				t := storeTask(ctx, "PagerDuty Test", validConfig(
					pd.Alert{Status: "saved-key"},
				))
				tsk := MustSucceed(factory.ConfigureTask(ctx, t, "cmd-1"))
				defer func() { Expect(tsk.Stop(true)).To(Succeed()) }()
				Expect(tsk.Exec(ctx, task.Command{Type: "start"})).To(Succeed())
				other := createRoutingKey(ctx, strings.Repeat("c", 32))
				cfg := validConfig(pd.Alert{Status: "saved-key"})
				cfg.RoutingKey = &other
				rec := MustSucceed(alertSvc.Alert.Prepare(
					ctx,
					db,
					t.Key,
					t.Name,
					encodeConfig(cfg),
				))
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return rec.Write(ctx, tx)
				})).To(Succeed())

				setStatus(ctx, "saved-key", status.VariantError, "Trigger", nil)
				Eventually(func() int { return len(sender.getEvents()) }).
					WithTimeout(2 * time.Second).
					Should(Equal(1))
				Expect(sender.getEvents()[0].RoutingKey).
					To(Equal(strings.Repeat("b", 32)))
			},
		)

		It("Should fail to start when the routing key is not 32 characters",
			func(ctx context.Context) {
				t := storeTask(
					ctx,
					"PagerDuty Test",
					validConfig(pd.Alert{Status: "bad-key"}),
				)
				tsk := MustSucceed(factory.ConfigureTask(ctx, t, "cmd-1"))
				defer func() { Expect(tsk.Stop(true)).To(Succeed()) }()
				replaceRoutingKey(ctx, routingKey, "tooshort")
				Expect(tsk.Exec(ctx, task.Command{Type: "start"})).To(Succeed())
				Eventually(func(g Gomega) {
					var stat task.Status
					g.Expect(statusSvc.NewRetrieve[task.StatusDetails]().
						Where(status.MatchOwners[task.StatusDetails](t.ResourceID())).
						Entry(&stat).
						Exec(ctx, db)).To(Succeed())
					g.Expect(stat.Variant).To(Equal(status.VariantError))
					g.Expect(stat.Message).To(Equal("Failed to start task"))
					g.Expect(stat.Description).
						To(ContainSubstring("must be exactly 32 characters"))
				}).WithTimeout(2 * time.Second).Should(Succeed())
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
				w := statusSvc.NewWriter(tx)
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

		It("Should use the status key as the source when the status has no name",
			func(ctx context.Context) {
				tsk := configureAndStart(ctx, validConfig(
					pd.Alert{Status: "unnamed"},
				))
				defer func() { Expect(tsk.Stop(true)).To(Succeed()) }()

				tx := db.OpenTx()
				defer func() { Expect(tx.Close()).To(Succeed()) }()
				Expect(statusSvc.NewWriter(tx).Set(ctx, &status.Status[any]{
					Key:     "unnamed",
					Variant: status.VariantError,
					Message: "Something broke",
					Time:    telem.Now(),
				})).To(Succeed())
				Expect(tx.Commit(ctx)).To(Succeed())

				Eventually(func() int { return len(sender.getEvents()) }).
					WithTimeout(2 * time.Second).
					Should(BeNumerically(">=", 1))
				Expect(sender.getEvents()[0].Payload.Source).To(Equal("unnamed"))
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
				sender.setError(errors.New("simulated PagerDuty outage"))
				t := storeTask(
					ctx,
					"PagerDuty Test",
					validConfig(pd.Alert{Status: "send-failure"}),
				)
				tsk := MustSucceed(factory.ConfigureTask(ctx, t, "cmd-1"))
				defer func() { Expect(tsk.Stop(true)).To(Succeed()) }()
				Expect(tsk.Exec(ctx, task.Command{Type: "start"})).To(Succeed())
				retrieve := func(g Gomega) task.Status {
					var stat task.Status
					g.Expect(statusSvc.NewRetrieve[task.StatusDetails]().
						Where(status.MatchOwners[task.StatusDetails](t.ResourceID())).
						Entry(&stat).
						Exec(ctx, db)).To(Succeed())
					return stat
				}
				const (
					message     = "Failed to send PagerDuty event"
					description = "simulated PagerDuty outage"
				)

				setStatus(ctx, "send-failure", status.VariantError,
					"Trigger send", nil)
				Eventually(func(g Gomega) {
					stat := retrieve(g)
					g.Expect(stat.Variant).To(Equal(status.VariantError))
					g.Expect(stat.Message).To(Equal(message))
					g.Expect(stat.Description).To(Equal(description))
					g.Expect(stat.Details.Running).To(BeTrue())
				}).WithTimeout(2 * time.Second).Should(Succeed())

				By("Keeping the error status after a send that succeeds")
				sender.setError(nil)
				setStatus(ctx, "send-failure", status.VariantSuccess, "Resolved", nil)
				Eventually(func() int { return len(sender.getEvents()) }).
					WithTimeout(2 * time.Second).
					Should(Equal(1))
				stat := retrieve(Default)
				Expect(stat.Variant).To(Equal(status.VariantError))
				Expect(stat.Message).To(Equal(message))
				Expect(stat.Description).To(Equal(description))
			},
		)
	})
})
