// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package sift

import (
	"context"

	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/sift/client"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	xstatus "github.com/synnaxlabs/x/status"
	"github.com/synnaxlabs/x/telem"
	"go.uber.org/zap"
)

// uploadTask handles a single upload to Sift.
type uploadTask struct {
	task       task.Task
	cfg        TaskConfig
	siftClient client.Client
	fCfg       FactoryConfig

	uploader *Uploader
}

var _ driver.Task = (*uploadTask)(nil)

func newUploadTask(
	t task.Task,
	cfg TaskConfig,
	siftClient client.Client,
	fCfg FactoryConfig,
) *uploadTask {
	return &uploadTask{
		task:       t,
		cfg:        cfg,
		siftClient: siftClient,
		fCfg:       fCfg,
	}
}

func (u *uploadTask) run(ctx context.Context) {
	u.uploader = &Uploader{
		Client:     u.siftClient,
		Framer:     u.fCfg.Framer,
		ChannelSvc: u.fCfg.Channel,
	}

	u.setStatus(xstatus.VariantInfo, "Starting upload", true)

	params := UploadParams{
		ClientKey: clientKey(u.task.Key),
		AssetName: u.cfg.AssetName,
		FlowName:  u.cfg.FlowName,
		RunName:   u.cfg.RunName,
		Channels:  u.cfg.Channels,
		TimeRange: u.cfg.TimeRange,
	}

	err := u.uploader.Upload(ctx, params)
	if err != nil {
		if ctx.Err() != nil {
			u.setStatus(xstatus.VariantWarning, "Upload cancelled", false)
		} else {
			u.setStatus(xstatus.VariantError, err.Error(), false)
		}
	} else {
		u.setStatus(xstatus.VariantSuccess, "Upload completed", false)
		u.deleteTask()
	}
}

func (u *uploadTask) Exec(ctx context.Context, cmd task.Command) error {
	return driver.ErrUnsupportedCommand
}

func (u *uploadTask) Stop() error {
	if u.uploader != nil {
		u.uploader.Stop()
	}
	return nil
}

func (u *uploadTask) setStatus(variant xstatus.Variant, message string, running bool) {
	stat := task.Status{
		Key:     task.OntologyID(u.task.Key).String(),
		Name:    u.task.Name,
		Variant: variant,
		Message: message,
		Time:    telem.Now(),
		Details: task.StatusDetails{Task: u.task.Key, Running: running},
	}
	if err := status.NewWriter[task.StatusDetails](
		u.fCfg.Status, nil,
	).Set(context.Background(), &stat); err != nil {
		u.fCfg.L.Error("failed to set status", zap.Error(err))
	}
}

func (u *uploadTask) deleteTask() {
	if err := u.fCfg.Task.NewWriter(nil).Delete(
		context.Background(), u.task.Key, false,
	); err != nil {
		u.fCfg.L.Error("failed to delete task",
			zap.Uint64("task", uint64(u.task.Key)),
			zap.Error(err))
	}
}
