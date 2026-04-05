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
	"encoding/json"
	"fmt"
	"time"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/observe"
	xstatus "github.com/synnaxlabs/x/status"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

// AlertTaskType is the type identifier for PagerDuty alert tasks.
const AlertTaskType = "pagerduty_alert"

// AlertConfig is the configuration for a single alert, mapping a Synnax status to a
// PagerDuty alert.
type AlertConfig struct {
	// Status is the Synnax status key to alert on.
	Status string `json:"status" msgpack:"status"`
	// TreatErrorAsCritical controls whether error status variant maps to "critical"
	// (true) or "error" (false) severity in PagerDuty.
	TreatErrorAsCritical bool `json:"treat_error_as_critical" msgpack:"treat_error_as_critical"`
	// Component of the source machine responsible for the event, for example "mysql" or
	// "eth0".
	Component string `json:"component" msgpack:"component"`
	// Group is a logical grouping of components of a service, for example "app-stack".
	Group string `json:"group" msgpack:"group"`
	// Class is the class/type of the event, for example "ping failure" or "cpu load".
	Class string `json:"class" msgpack:"class"`
	// Enabled controls whether this specific alert is active.
	Enabled bool `json:"enabled" msgpack:"enabled"`
}

// AlertTaskConfig is the configuration for a PagerDuty alert task.
type AlertTaskConfig struct {
	// RoutingKey is the 32-character Integration Key for an integration on a service
	// or on a global ruleset.
	RoutingKey string `json:"routing_key" msgpack:"routing_key"`
	// AutoStart controls whether the task starts automatically when configured.
	AutoStart bool `json:"auto_start" msgpack:"auto_start"`
	// Alerts is the list of alert configurations to send.
	Alerts []AlertConfig `json:"alerts" msgpack:"alerts"`
}

// Validate validates the alert task configuration.
func (c AlertTaskConfig) Validate() error {
	v := validate.New("pagerduty.alert_task_config")
	v.Ternary("routing_key", len(c.RoutingKey) != 32, "must be exactly 32 characters")
	var hasEnabled bool
	for _, a := range c.Alerts {
		if a.Enabled {
			hasEnabled = true
			break
		}
	}
	v.Ternary("alerts", !hasEnabled, "at least one alert must be enabled")
	return v.Error()
}

// MsgpackEncodedJSON converts the config into a binary.MsgpackEncodedJSON suitable
// for use as a task.Task.Config value.
func (c AlertTaskConfig) MsgpackEncodedJSON() (msgpack.EncodedJSON, error) {
	b, err := json.Marshal(c)
	if err != nil {
		return nil, err
	}
	var m msgpack.EncodedJSON
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return m, nil
}

type alertTask struct {
	factoryCfg FactoryConfig
	task       task.Task
	cfg        AlertTaskConfig
	disconnect observe.Disconnect
	// alertsByStatus maps status keys to their AlertConfig for O(1) lookup.
	alertsByStatus map[string]AlertConfig
}

var _ driver.Task = (*alertTask)(nil)

func (t *alertTask) Exec(ctx context.Context, cmd task.Command) error {
	switch cmd.Type {
	case "start":
		return t.start(ctx)
	case "stop":
		return t.stop(ctx)
	default:
		return driver.ErrUnsupportedCommand
	}
}

func (t *alertTask) start(ctx context.Context) error {
	if t.disconnect != nil {
		return nil
	}
	t.alertsByStatus = make(map[string]AlertConfig, len(t.cfg.Alerts))
	for _, a := range t.cfg.Alerts {
		if a.Enabled {
			t.alertsByStatus[a.Status] = a
		}
	}
	t.disconnect = t.factoryCfg.Status.Observe().OnChange(t.handleStatusChange)
	t.updateStatus(ctx, xstatus.VariantSuccess, true, "Task started successfully")
	return nil
}

func (t *alertTask) Stop() error { return t.stop(context.TODO()) }

func (t *alertTask) stop(ctx context.Context) error {
	if t.disconnect != nil {
		t.disconnect()
		t.disconnect = nil
	}
	t.updateStatus(ctx, xstatus.VariantSuccess, false, "Task stopped successfully")
	return nil
}

func (t *alertTask) handleStatusChange(
	ctx context.Context,
	reader gorp.TxReader[string, status.Status[any]],
) {
	for ch := range reader {
		if ch.Variant == change.VariantDelete {
			continue
		}
		alertCfg, ok := t.alertsByStatus[ch.Key]
		if !ok || !alertCfg.Enabled {
			continue
		}
		s := ch.Value
		switch s.Variant {
		case xstatus.VariantError, xstatus.VariantWarning, xstatus.VariantInfo:
			event := t.buildTriggerEvent(s, alertCfg)
			t.sendEvent(ctx, event)
		case xstatus.VariantSuccess:
			event := t.buildResolveEvent(s.Key)
			t.sendEvent(ctx, event)
		default:
			// loading, disabled — skip
		}
	}
}

func (t *alertTask) buildTriggerEvent(
	s status.Status[any],
	alertCfg AlertConfig,
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
			Severity:  t.mapSeverity(s.Variant, alertCfg.TreatErrorAsCritical),
			Timestamp: s.Time.Time().Format(time.RFC3339),
			Component: alertCfg.Component,
			Group:     alertCfg.Group,
			Class:     alertCfg.Class,
			Details:   s.Details,
		},
	}
}

func (t *alertTask) buildResolveEvent(statusKey string) pagerduty.V2Event {
	return pagerduty.V2Event{
		RoutingKey: t.cfg.RoutingKey,
		Action:     "resolve",
		DedupKey:   statusKey,
	}
}

func (t *alertTask) mapSeverity(
	variant xstatus.Variant,
	treatErrorAsCritical bool,
) string {
	switch variant {
	case xstatus.VariantError:
		if treatErrorAsCritical {
			return "critical"
		}
		return "error"
	case xstatus.VariantWarning:
		return "warning"
	case xstatus.VariantInfo:
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
		t.updateStatus(ctx, xstatus.VariantError, true,
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
	variant xstatus.Variant,
	running bool,
	message string,
) {
	stat := task.Status{
		Key:     task.OntologyID(t.task.Key).String(),
		Name:    t.task.Name,
		Variant: variant,
		Message: message,
		Time:    telem.Now(),
		Details: task.StatusDetails{Task: t.task.Key, Running: running},
	}
	err := status.NewWriter[task.StatusDetails](t.factoryCfg.Status, nil).
		Set(ctx, &stat)
	if err != nil {
		t.factoryCfg.L.Error("failed to set task status", zap.Error(err))
	}
}
