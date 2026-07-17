// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package slack

import (
	"context"
	"encoding/json"

	"github.com/synnaxlabs/synnax/pkg/service/device"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

// AlertTaskType is the type identifier for Slack alert tasks.
const AlertTaskType = "slack_alert"

// AlertTaskConfig is the configuration for a Slack alert task: which device to post
// through, which channel to post to, and which statuses to watch.
type AlertTaskConfig struct {
	// Device is the key of the Slack device holding the workspace bot token.
	Device string `json:"device" msgpack:"device"`
	// Channel is the Slack channel name or ID to post to.
	Channel string `json:"channel" msgpack:"channel"`
	// Statuses is the list of status keys to watch; a change to any one posts a message.
	Statuses []string `json:"statuses" msgpack:"statuses"`
	// AutoStart controls whether the task starts automatically when configured.
	AutoStart bool `json:"auto_start" msgpack:"auto_start"`
}

// Validate validates the alert task configuration.
func (c AlertTaskConfig) Validate() error {
	v := validate.New("slack.alert_task_config")
	validate.NotEmptyString(v, "device", c.Device)
	validate.NotEmptyString(v, "channel", c.Channel)
	validate.NotEmptySlice(v, "statuses", c.Statuses)
	return v.Error()
}

// MsgpackEncodedJSON converts the config into a msgpack.EncodedJSON suitable for use as
// a task.Task.Config value.
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

// deviceProperties is the subset of a Slack device's properties this task reads.
type deviceProperties struct {
	// BotToken authenticates the app to the Slack workspace.
	BotToken string `json:"bot_token" msgpack:"bot_token"`
}

type alertTask struct {
	factoryCfg FactoryConfig
	task       task.Task
	cfg        AlertTaskConfig
	// token is the workspace bot token, resolved from the device at start.
	token string
	// watched holds the status keys this task posts on.
	watched    set.Set[string]
	disconnect observe.Disconnect
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
	if err := t.resolveToken(ctx); err != nil {
		t.updateStatus(ctx, status.VariantError, false, err.Error())
		return err
	}
	t.watched = set.New(t.cfg.Statuses...)
	t.disconnect = t.factoryCfg.Status.Observe().OnChange(t.handleStatusChange)
	t.updateStatus(ctx, status.VariantSuccess, true, "Task started successfully")
	return nil
}

// resolveToken retrieves the configured device and reads its bot token.
func (t *alertTask) resolveToken(ctx context.Context) error {
	var dev device.Device
	if err := t.factoryCfg.Device.NewRetrieve().
		Where(device.MatchKeys(t.cfg.Device)).
		Entry(&dev).
		Exec(ctx, nil); err != nil {
		return errors.Wrapf(err, "failed to retrieve slack device %s", t.cfg.Device)
	}
	var props deviceProperties
	if err := dev.Properties.Unmarshal(&props); err != nil {
		return errors.Wrap(err, "failed to parse slack device properties")
	}
	if props.BotToken == "" {
		return errors.Newf("slack device %s has no bot token", t.cfg.Device)
	}
	t.token = props.BotToken
	return nil
}

func (t *alertTask) Stop() error { return t.stop(context.TODO()) }

func (t *alertTask) stop(ctx context.Context) error {
	if t.disconnect != nil {
		t.disconnect()
		t.disconnect = nil
	}
	t.updateStatus(ctx, status.VariantSuccess, false, "Task stopped successfully")
	return nil
}

func (t *alertTask) handleStatusChange(
	ctx context.Context,
	reader gorp.TxReader[string, status.Status[any]],
) {
	for ch := range reader {
		if ch.Variant == change.VariantDelete || !t.watched.Contains(ch.Key) {
			continue
		}
		if err := t.factoryCfg.Sender.Post(
			ctx, t.token, buildMessage(t.cfg.Channel, ch.Value),
		); err != nil {
			t.factoryCfg.L.Error(
				"failed to post slack message",
				zap.Stringer("task", t.task),
				zap.String("status", ch.Key),
				zap.Error(err),
			)
			t.updateStatus(ctx, status.VariantError, true,
				"Failed to post Slack message: "+err.Error())
		}
	}
}

func (t *alertTask) updateStatus(
	ctx context.Context,
	variant status.Variant,
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
	if err := status.NewWriter[task.StatusDetails](t.factoryCfg.Status, nil).
		Set(ctx, &stat); err != nil {
		t.factoryCfg.L.Error("failed to set task status", zap.Error(err))
	}
}
