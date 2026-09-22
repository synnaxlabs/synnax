// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package pagerduty

import (
	"context"
	"fmt"
	"time"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

// AlertTaskType is the type identifier for PagerDuty alert tasks.
const AlertTaskType = "pagerduty_alert"

// validateConfig checks the deploy-time constraints on a task config: a real
// routing key and at least one enabled alert.
func validateConfig(c TaskConfig) error {
	v := validate.New("pagerduty.task_config")
	v.Ternary("routing_key", len(c.RoutingKey) != 32, "must be exactly 32 characters")
	var hasEnabled bool
	for _, a := range c.Alerts {
		if !a.Disabled {
			hasEnabled = true
			break
		}
	}
	v.Ternary("alerts", !hasEnabled, "at least one alert must be enabled")
	return v.Error()
}

// alertTask sends a PagerDuty event for each change of a watched status.
type alertTask struct {
	*driver.Runner
	factoryCfg FactoryConfig
	task       task.Task
	cfg        TaskConfig
	// disconnect ends the status observation of the current run.
	disconnect observe.Disconnect
	// alertsByStatus holds the enabled alerts by the status they watch.
	alertsByStatus map[status.Key]Alert
}

var _ driver.Task = (*alertTask)(nil)

func newAlertTask(
	factoryCfg FactoryConfig,
	t task.Task,
	cfg TaskConfig,
) (*alertTask, error) {
	at := &alertTask{
		factoryCfg:     factoryCfg,
		task:           t,
		cfg:            cfg,
		alertsByStatus: make(map[status.Key]Alert, len(cfg.Alerts)),
	}
	for _, a := range cfg.Alerts {
		if !a.Disabled {
			at.alertsByStatus[a.Status] = a
		}
	}
	var err error
	at.Runner, err = driver.NewRunner(driver.RunnerConfig{
		Status:          factoryCfg.Status,
		Instrumentation: factoryCfg.Instrumentation,
		Task:            t,
		Open:            at.open,
		Run:             at.run,
	})
	return at, err
}

func (t *alertTask) open(context.Context) error {
	t.disconnect = t.factoryCfg.Status.Observe().OnChange(t.handleStatusChange)
	return nil
}

func (t *alertTask) run(ctx context.Context) error {
	<-ctx.Done()
	t.disconnect()
	return nil
}

func (t *alertTask) handleStatusChange(
	ctx context.Context,
	reader gorp.TxReader[status.Key, status.Status[any]],
) {
	for ch := range reader {
		if ch.Variant == change.VariantDelete {
			continue
		}
		alertCfg, ok := t.alertsByStatus[ch.Key]
		if !ok {
			continue
		}
		s := ch.Value
		switch s.Variant {
		case status.VariantError, status.VariantWarning, status.VariantInfo:
			event := t.buildTriggerEvent(s, alertCfg)
			t.sendEvent(ctx, event)
		case status.VariantSuccess:
			event := t.buildResolveEvent(s.Key)
			t.sendEvent(ctx, event)
		default:
			// loading, disabled — skip
		}
	}
}

func (t *alertTask) buildTriggerEvent(
	s status.Status[any],
	alertCfg Alert,
) pagerduty.V2Event {
	summary := s.Message
	if s.Description != "" {
		summary += fmt.Sprintf(": %s", s.Description)
	}
	return pagerduty.V2Event{
		RoutingKey: t.cfg.RoutingKey,
		Action:     "trigger",
		DedupKey:   s.Key,
		Client:     "Synnax",
		Payload: &pagerduty.V2Payload{
			Summary:   summary,
			Source:    s.Name,
			Severity:  mapSeverity(s.Variant, alertCfg.ErrorsCritical),
			Timestamp: s.Time.Time().Format(time.RFC3339),
			Component: alertCfg.Component,
			Group:     alertCfg.Group,
			Class:     alertCfg.Class,
			Details:   s.Details,
		},
	}
}

func (t *alertTask) buildResolveEvent(statusKey status.Key) pagerduty.V2Event {
	return pagerduty.V2Event{
		RoutingKey: t.cfg.RoutingKey,
		Action:     "resolve",
		DedupKey:   statusKey,
	}
}

func mapSeverity(variant status.Variant, errorsCritical bool) string {
	switch variant {
	case status.VariantError:
		if errorsCritical {
			return "critical"
		}
		return "error"
	case status.VariantWarning:
		return "warning"
	case status.VariantInfo:
		return "info"
	default:
		return "info"
	}
}

// sendEvent sends event and reports the result as the health of the task: a failed
// send warns until the next send succeeds.
func (t *alertTask) sendEvent(ctx context.Context, event pagerduty.V2Event) {
	resp, err := t.factoryCfg.Sender.SendEvent(ctx, event)
	if err != nil {
		t.factoryCfg.L.Error(
			"failed to send PagerDuty event",
			zap.Stringer("task", t.task),
			zap.Any("event", event),
			zap.Error(err),
		)
		t.Report(ctx, errors.Wrapf(
			driver.ErrTemporary, "failed to send PagerDuty event: %s", err.Error(),
		))
		return
	}
	t.Report(ctx, nil)
	t.factoryCfg.L.Debug(
		"PagerDuty event sent successfully",
		zap.Any("event", event),
		zap.Any("response", resp),
		zap.Stringer("task", t.task),
	)
}
