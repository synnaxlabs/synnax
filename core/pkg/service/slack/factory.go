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
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/telem"
	"go.uber.org/zap"
)

type factory struct{ cfg FactoryConfig }

var _ driver.Factory = (*factory)(nil)

// NewFactory creates a new Slack factory.
func NewFactory(cfgs ...FactoryConfig) (driver.Factory, error) {
	cfg, err := config.New(FactoryConfig{Sender: defaultSender}, cfgs...)
	if err != nil {
		return nil, err
	}
	return &factory{cfg: cfg}, nil
}

func (f *factory) ConfigureTask(ctx context.Context, t task.Task) (driver.Task, error) {
	switch t.Type {
	case ScanTaskType:
		return &scanTask{factoryCfg: f.cfg, task: t}, nil
	case AlertTaskType:
		return f.configureAlert(ctx, t)
	default:
		return nil, driver.ErrTaskNotHandled
	}
}

func (f *factory) configureAlert(ctx context.Context, t task.Task) (driver.Task, error) {
	var cfg AlertTaskConfig
	if err := t.Config.Unmarshal(&cfg); err != nil {
		f.setConfigStatus(ctx, t, status.VariantError, err.Error())
		return nil, err
	}
	if err := cfg.Validate(); err != nil {
		f.setConfigStatus(ctx, t, status.VariantError, err.Error())
		return nil, err
	}
	slackTask := &alertTask{factoryCfg: f.cfg, task: t, cfg: cfg}
	if cfg.AutoStart {
		if err := slackTask.start(ctx); err != nil {
			return nil, err
		}
	} else {
		f.setConfigStatus(ctx, t, status.VariantSuccess, "Task configured successfully")
	}
	return slackTask, nil
}

func (f *factory) setConfigStatus(
	ctx context.Context,
	t task.Task,
	variant status.Variant,
	message string,
) {
	stat := task.Status{
		Key:     task.OntologyID(t.Key).String(),
		Name:    t.Name,
		Variant: variant,
		Message: message,
		Time:    telem.Now(),
		Details: task.StatusDetails{Task: t.Key, Running: false},
	}
	if err := status.NewWriter[task.StatusDetails](f.cfg.Status, nil).
		Set(ctx, &stat); err != nil {
		f.cfg.L.Error(
			"failed to set configuration status",
			zap.Stringer("task", t),
			zap.Stringer("status", stat),
			zap.Error(err),
		)
	}
}

func (f *factory) Name() string { return "slack" }
