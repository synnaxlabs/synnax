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
	"encoding/json"

	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/service/device"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/sift/client"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/errors"
	xstatus "github.com/synnaxlabs/x/status"
	"github.com/synnaxlabs/x/telem"
	"go.uber.org/zap"
)

// UploadTaskType is the task type for Sift uploads.
const UploadTaskType = "sift_upload"

type UploadTaskConfig struct {
	// DeviceKey references the Sift device containing connection config.
	DeviceKey string `json:"device_key"`
	// AssetName is the Sift asset name to upload to.
	AssetName string `json:"asset_name"`
	// FlowName is the Sift flow name for this upload.
	FlowName string `json:"flow_name"`
	// RunName is the Sift run name. A run will be created with this name.
	RunName string `json:"run_name"`
	// Channels are the Synnax channel keys to upload.
	Channels []channel.Key `json:"channels"`
	// TimeRange is the time range to upload.
	TimeRange telem.TimeRange `json:"time_range"`
}

func ParseUploadTaskConfig(s string) (UploadTaskConfig, error) {
	var c UploadTaskConfig
	if err := json.Unmarshal([]byte(s), &c); err != nil {
		return c, errors.Wrap(err, "failed to parse Sift upload task config")
	}
	return c, nil
}

// uploadTask handles a single upload to Sift.
type uploadTask struct {
	task       task.Task
	cfg        UploadTaskConfig
	siftClient client.Client
	fCfg       FactoryConfig

	uploader *Uploader
}

var _ driver.Task = (*uploadTask)(nil)

func newUploadTask(
	t task.Task,
	cfg UploadTaskConfig,
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
		ClientKey: string(u.task.Key),
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

func (u *uploadTask) Exec(context.Context, task.Command) error {
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

func (f *Factory) configureUploadTask(ctx driver.Context, t task.Task) (driver.Task, error) {
	cfg, err := ParseUploadTaskConfig(t.Config)
	if err != nil {
		f.setStatus(ctx, t, xstatus.VariantError, err.Error(), false)
		return nil, err
	}

	// Retrieve device for connection properties
	var dev device.Device
	if err := f.cfg.Device.NewRetrieve().
		WhereKeys(cfg.DeviceKey).
		Entry(&dev).
		Exec(ctx, nil); err != nil {
		f.setStatus(ctx, t, xstatus.VariantError, err.Error(), false)
		return nil, err
	}

	props, err := ParseDeviceProperties(dev.Properties)
	if err != nil {
		f.setStatus(ctx, t, xstatus.VariantError, err.Error(), false)
		return nil, err
	}

	// Get or create client
	client, err := f.pool.Get(ctx, props.URI, props.APIKey)
	if err != nil {
		f.setStatus(ctx, t, xstatus.VariantError, err.Error(), false)
		return nil, err
	}

	uploadTask := newUploadTask(t, cfg, client, f.cfg)

	f.setStatus(ctx, t, xstatus.VariantSuccess, "Task configured", true)

	// Auto-start the upload
	go uploadTask.run(ctx)

	return uploadTask, nil
}
