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

	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/telem"
	"go.uber.org/zap"
)

const (
	// ScanTaskType is the type of the internal Slack scan task, which validates bot
	// tokens on behalf of the device connect flow.
	ScanTaskType = "slack_scan"
	// TestConnectionCommandType validates a bot token via auth.test.
	TestConnectionCommandType = "test_connection"
	// scanTaskName is the human-readable name of the internal scan task.
	scanTaskName = "Slack Scanner"
)

// testConnectionArgs are the arguments of a test_connection command.
type testConnectionArgs struct {
	// Token is the bot token to validate.
	Token string `json:"token" msgpack:"token"`
}

type scanTask struct {
	factoryCfg FactoryConfig
	task       task.Task
}

var _ driver.Task = (*scanTask)(nil)

func (t *scanTask) Exec(ctx context.Context, cmd task.Command) error {
	if cmd.Type != TestConnectionCommandType {
		return driver.ErrUnsupportedCommand
	}
	var args testConnectionArgs
	if err := cmd.Args.Unmarshal(&args); err != nil {
		t.report(ctx, cmd.Key, status.VariantError, err.Error())
		return nil
	}
	if err := t.factoryCfg.Sender.AuthTest(ctx, args.Token); err != nil {
		t.report(ctx, cmd.Key, status.VariantError, err.Error())
		return nil
	}
	t.report(ctx, cmd.Key, status.VariantSuccess, "Workspace connected")
	return nil
}

func (t *scanTask) Stop() error { return nil }

// report publishes a command's outcome as a task status keyed by the command, so the
// synchronous caller (the device connect modal) can observe it.
func (t *scanTask) report(
	ctx context.Context,
	cmdKey string,
	variant status.Variant,
	message string,
) {
	stat := task.Status{
		Key:     task.OntologyID(t.task.Key).String(),
		Name:    t.task.Name,
		Variant: variant,
		Message: message,
		Time:    telem.Now(),
		Details: task.StatusDetails{Task: t.task.Key, Running: true, Cmd: cmdKey},
	}
	if err := status.NewWriter[task.StatusDetails](t.factoryCfg.Status, nil).
		Set(ctx, &stat); err != nil {
		t.factoryCfg.L.Error("failed to set scan task status", zap.Error(err))
	}
}

// EnsureScanTask creates the internal Slack scan task on the given rack if one does not
// already exist. It is idempotent across restarts.
func EnsureScanTask(ctx context.Context, tasks *task.Service, rackKey rack.Key) error {
	var existing []task.Task
	if err := tasks.NewRetrieve().
		Where(task.And(task.MatchRacks(rackKey), task.MatchTypes(ScanTaskType))).
		Entries(&existing).
		Exec(ctx, nil); err != nil {
		return err
	}
	if len(existing) > 0 {
		return nil
	}
	scan := task.Task{
		Key:      task.NewKey(rackKey, 0),
		Name:     scanTaskName,
		Type:     ScanTaskType,
		Internal: true,
	}
	return tasks.NewWriter(nil).Create(ctx, &scan)
}
