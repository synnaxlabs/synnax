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

type alertTask struct {
	factoryCfg FactoryConfig
	task       task.Task
	cfg        TaskConfig
	// status is the authority on this instance's current status.
	status     *driver.StatusHandler
	disconnect observe.Disconnect
	// alertsByStatus maps status keys to their enabled Alert for O(1) lookup.
	alertsByStatus map[status.Key]Alert
}

var _ driver.Task = (*alertTask)(nil)

// Exec implements driver.Task. Every command it handles ends in a status carrying
// the command key, so a caller waiting on the command always resolves.
func (t *alertTask) Exec(ctx context.Context, cmd task.Command) error {
	switch cmd.Type {
	case "start":
		return t.start(ctx, cmd.Key)
	case "stop":
		return t.stop(ctx, cmd.Key, true)
	default:
		return driver.ErrUnsupportedCommand
	}
}

func (t *alertTask) start(ctx context.Context, cmdKey string) error {
	if t.disconnect != nil {
		t.ackCurrent(ctx, cmdKey, true)
		return nil
	}
	t.alertsByStatus = make(map[status.Key]Alert, len(t.cfg.Alerts))
	for _, a := range t.cfg.Alerts {
		if !a.Disabled {
			t.alertsByStatus[a.Status] = a
		}
	}
	t.disconnect = t.factoryCfg.Status.Observe().OnChange(t.handleStatusChange)
	t.updateStatus(
		ctx,
		cmdKey,
		status.VariantSuccess,
		true,
		"Task started successfully",
	)
	return nil
}

func (t *alertTask) Stop(sendStatus bool) error {
	return t.stop(context.TODO(), driver.NoCommand, sendStatus)
}

func (t *alertTask) stop(ctx context.Context, cmdKey string, sendStatus bool) error {
	if t.disconnect == nil {
		if sendStatus {
			t.ackCurrent(ctx, cmdKey, false)
		}
		return nil
	}
	t.disconnect()
	t.disconnect = nil
	if sendStatus {
		t.updateStatus(
			ctx,
			cmdKey,
			status.VariantSuccess,
			false,
			"Task stopped successfully",
		)
	}
	return nil
}

// ackCurrent answers cmdKey with the task's current status, for a command that
// needs no work.
func (t *alertTask) ackCurrent(ctx context.Context, cmdKey string, running bool) {
	if err := t.status.Ack(ctx, cmdKey, running); err != nil {
		t.factoryCfg.L.Error("failed to acknowledge command",
			zap.Stringer("task", t.task),
			zap.String("cmd", cmdKey),
			zap.Error(err),
		)
	}
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
			Severity:  mapSeverity(s.Variant, alertCfg.TreatErrorAsCritical),
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

func mapSeverity(variant status.Variant, treatErrorAsCritical bool) string {
	switch variant {
	case status.VariantError:
		if treatErrorAsCritical {
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

func (t *alertTask) sendEvent(ctx context.Context, event pagerduty.V2Event) {
	resp, err := t.factoryCfg.Sender.SendEvent(ctx, event)
	if err != nil {
		t.factoryCfg.L.Error(
			"failed to send PagerDuty event",
			zap.Stringer("task", t.task),
			zap.Any("event", event),
			zap.Error(err),
		)
		t.updateStatus(ctx, driver.NoCommand, status.VariantError, true,
			fmt.Sprintf("Failed to send PagerDuty event: %s", err.Error()))
		return
	}
	t.factoryCfg.L.Debug(
		"PagerDuty event sent successfully",
		zap.Any("event", event),
		zap.Any("response", resp),
		zap.Stringer("task", t.task),
	)
}

func (t *alertTask) updateStatus(
	ctx context.Context,
	cmdKey string,
	variant status.Variant,
	running bool,
	message string,
) {
	if err := t.status.Send(ctx, cmdKey, variant, running, message); err != nil {
		t.factoryCfg.L.Error("failed to set task status", zap.Error(err))
	}
}
