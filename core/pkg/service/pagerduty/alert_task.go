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
	"uuid"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/secret"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

// AlertTaskType is the type identifier for PagerDuty alert tasks.
const AlertTaskType = "pagerduty_alert"

// validateConfig checks the deploy-time constraints on a task config: a routing key
// and at least one enabled alert.
func validateConfig(c TaskConfig) error {
	v := validate.New("pagerduty.task_config")
	v.Ternary("routing_key", c.RoutingKey == nil, "required")
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

// resolveRoutingKey returns the value of the secret with the given key, read as the
// rack of t. It returns a validation error when the value is not 32 characters.
func resolveRoutingKey(
	ctx context.Context,
	secrets *secret.Service,
	t task.Task,
	key secret.Key,
) (string, error) {
	values, err := secrets.Resolve(ctx, nil, t.Rack.OntologyID(), uuid.Nil(), key)
	if err != nil {
		return "", err
	}
	value := string(values[key])
	v := validate.New("pagerduty.task_config")
	v.Ternary("routing_key", len(value) != 32, "must be exactly 32 characters")
	if err = v.Error(); err != nil {
		return "", err
	}
	return value, nil
}

// alertTask sends a PagerDuty event for each change of a watched status.
type alertTask struct {
	*driver.Runner
	factoryCfg FactoryConfig
	task       task.Task
	cfg        TaskConfig
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
		DB:              factoryCfg.DB,
		Status:          factoryCfg.Status,
		Instrumentation: factoryCfg.Instrumentation,
		Task:            t,
		Open:            at.open,
	})
	if err != nil {
		return nil, err
	}
	return at, nil
}

// open reads the routing key once, so a run keeps the key it started with when the
// saved config or the secret's value changes.
func (t *alertTask) open(ctx context.Context) (func(context.Context) error, error) {
	routingKey, err := resolveRoutingKey(
		ctx,
		t.factoryCfg.Secret,
		t.task,
		*t.cfg.RoutingKey,
	)
	if err != nil {
		return nil, err
	}
	disconnect := t.factoryCfg.Status.Observe().OnChange(func(
		ctx context.Context,
		reader gorp.TxReader[status.Key, status.Status[any]],
	) {
		t.handleStatusChange(ctx, reader, routingKey)
	})
	return func(ctx context.Context) error {
		<-ctx.Done()
		disconnect()
		return nil
	}, nil
}

func (t *alertTask) handleStatusChange(
	ctx context.Context,
	reader gorp.TxReader[status.Key, status.Status[any]],
	routingKey string,
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
			t.sendEvent(ctx, event, routingKey)
		case status.VariantSuccess:
			event := t.buildResolveEvent(s.Key)
			t.sendEvent(ctx, event, routingKey)
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
	// PagerDuty requires a source, which an unnamed status lacks.
	source := s.Name
	if source == "" {
		source = s.Key
	}
	return pagerduty.V2Event{
		Action:   "trigger",
		DedupKey: s.Key,
		Client:   "Synnax",
		Payload: &pagerduty.V2Payload{
			Summary:   summary,
			Source:    source,
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
	return pagerduty.V2Event{Action: "resolve", DedupKey: statusKey}
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

// sendEvent sends event with routingKey. A failed send sets an error status on the
// task that stays until the next start or stop.
func (t *alertTask) sendEvent(
	ctx context.Context,
	event pagerduty.V2Event,
	routingKey string,
) {
	// The logs below carry event, which holds no routing key.
	sent := event
	sent.RoutingKey = routingKey
	resp, err := t.factoryCfg.Sender.SendEvent(ctx, sent)
	if err != nil {
		t.factoryCfg.L.Error(
			"failed to send PagerDuty event",
			zap.Stringer("task", t.task),
			zap.Any("event", event),
			zap.Error(err),
		)
		t.ReportError(ctx, "Failed to send PagerDuty event", err.Error())
		return
	}
	t.factoryCfg.L.Debug(
		"PagerDuty event sent successfully",
		zap.Any("event", event),
		zap.Any("response", resp),
		zap.Stringer("task", t.task),
	)
}
