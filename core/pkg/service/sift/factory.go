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
	"encoding/json"
	"fmt"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/service/device"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	xstatus "github.com/synnaxlabs/x/status"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

// FactoryConfig is the configuration for creating a Sift factory.
type FactoryConfig struct {
	// Device is used for retrieving Sift device information.
	// [REQUIRED]
	Device *device.Service
	// Ranger is used for retrieving range information for uploads.
	// [REQUIRED]
	Ranger *ranger.Service
	// Framer is used for reading/writing telemetry.
	// [REQUIRED]
	Framer *framer.Service
	// Channel is used for retrieving channel metadata.
	// [REQUIRED]
	Channel *channel.Service
	// Status is used for task status updates.
	// [REQUIRED]
	Status *status.Service
	alamos.Instrumentation
}

var (
	_                    config.Config[FactoryConfig] = FactoryConfig{}
	DefaultFactoryConfig                              = FactoryConfig{}
)

func (c FactoryConfig) Override(other FactoryConfig) FactoryConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.Device = override.Nil(c.Device, other.Device)
	c.Ranger = override.Nil(c.Ranger, other.Ranger)
	c.Framer = override.Nil(c.Framer, other.Framer)
	c.Channel = override.Nil(c.Channel, other.Channel)
	c.Status = override.Nil(c.Status, other.Status)
	return c
}

func (c FactoryConfig) Validate() error {
	v := validate.New("sift.factory")
	validate.NotNil(v, "device", c.Device)
	validate.NotNil(v, "ranger", c.Ranger)
	validate.NotNil(v, "framer", c.Framer)
	validate.NotNil(v, "channel", c.Channel)
	validate.NotNil(v, "status", c.Status)
	return v.Error()
}

// Factory creates Sift tasks from task definitions.
type Factory struct {
	cfg  FactoryConfig
	pool *ConnectionPool
}

var _ driver.Factory = (*Factory)(nil)

// OpenFactory creates a new Sift factory.
func OpenFactory(cfgs ...FactoryConfig) (*Factory, error) {
	cfg, err := config.New(DefaultFactoryConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Factory{
		cfg:  cfg,
		pool: NewConnectionPool(),
	}, nil
}

// ConfigureTask creates a Sift task if this factory handles the task type.
func (f *Factory) ConfigureTask(ctx driver.Context, t task.Task) (driver.Task, error) {
	switch t.Type {
	case TaskTypeUploader:
		return f.configureUploader(ctx, t)
	case TaskTypeWriter:
		return f.configureWriter(ctx, t)
	default:
		return nil, driver.ErrNotHandled
	}
}

func (f *Factory) configureUploader(
	ctx driver.Context,
	t task.Task,
) (driver.Task, error) {
	var cfg UploaderTaskConfig
	if err := json.Unmarshal([]byte(t.Config), &cfg); err != nil {
		f.setConfigStatus(ctx, t, xstatus.VariantError, err.Error())
		return nil, err
	}

	// Retrieve the device to get connection properties
	var dev device.Device
	if err := f.cfg.Device.NewRetrieve().
		WhereKeys(cfg.DeviceKey).
		Entry(&dev).
		Exec(ctx, nil); err != nil {
		f.setConfigStatus(ctx, t, xstatus.VariantError, err.Error())
		return nil, err
	}

	props, err := ParseDeviceProperties(dev.Properties)
	if err != nil {
		f.setConfigStatus(ctx, t, xstatus.VariantError, err.Error())
		return nil, err
	}

	uploader := newUploaderTask(t, props, f.cfg, f.pool)
	f.setConfigStatus(ctx, t, xstatus.VariantSuccess, "Uploader task configured successfully")
	return uploader, nil
}

func (f *Factory) configureWriter(
	ctx driver.Context,
	t task.Task,
) (driver.Task, error) {
	var cfg WriterTaskConfig
	if err := json.Unmarshal([]byte(t.Config), &cfg); err != nil {
		f.setConfigStatus(ctx, t, xstatus.VariantError, err.Error())
		return nil, err
	}

	// Retrieve the device to get connection properties
	var dev device.Device
	if err := f.cfg.Device.NewRetrieve().
		WhereKeys(cfg.DeviceKey).
		Entry(&dev).
		Exec(ctx, nil); err != nil {
		f.setConfigStatus(ctx, t, xstatus.VariantError, err.Error())
		return nil, err
	}

	props, err := ParseDeviceProperties(dev.Properties)
	if err != nil {
		f.setConfigStatus(ctx, t, xstatus.VariantError, err.Error())
		return nil, err
	}

	writer, err := newWriterTask(ctx, t, cfg, props, f.cfg, f.pool)
	if err != nil {
		f.setConfigStatus(ctx, t, xstatus.VariantError, err.Error())
		return nil, err
	}

	if cfg.AutoStart {
		if err := writer.Exec(ctx, task.Command{Type: "start"}); err != nil {
			return writer, err
		}
	} else {
		f.setConfigStatus(ctx, t, xstatus.VariantSuccess, "Writer task configured successfully")
	}

	return writer, nil
}

// ConfigureInitialTasks creates uploader tasks for each Sift device on the rack.
func (f *Factory) ConfigureInitialTasks(
	ctx driver.Context,
	rackKey rack.Key,
) ([]task.Task, error) {
	// Query all Sift devices on this rack
	var devices []device.Device
	if err := f.cfg.Device.NewRetrieve().
		WhereRacks(rackKey).
		Entries(&devices).
		Exec(ctx, nil); err != nil {
		return nil, err
	}

	var tasks []task.Task
	for _, dev := range devices {
		if dev.Make != DeviceMake {
			continue // Skip non-Sift devices
		}

		// Create uploader task config
		cfg := UploaderTaskConfig{DeviceKey: dev.Key}
		cfgJSON, err := json.Marshal(cfg)
		if err != nil {
			f.cfg.L.Error("failed to marshal uploader config",
				zap.String("device", dev.Key),
				zap.Error(err),
			)
			continue
		}

		// Generate a deterministic task key for this device
		taskKey := task.NewKey(rackKey, deviceKeyToLocalKey(dev.Key))

		tasks = append(tasks, task.Task{
			Key:      taskKey,
			Name:     fmt.Sprintf("Sift Uploader - %s", dev.Name),
			Type:     TaskTypeUploader,
			Config:   string(cfgJSON),
			Internal: true,
		})
	}

	return tasks, nil
}

func (f *Factory) setConfigStatus(
	ctx driver.Context,
	t task.Task,
	variant xstatus.Variant,
	message string,
) {
	stat := task.Status{
		Key:     task.OntologyID(t.Key).String(),
		Name:    t.Name,
		Variant: variant,
		Message: message,
		Time:    telem.Now(),
		Details: task.StatusDetails{
			Task:    t.Key,
			Running: false,
		},
	}
	if err := ctx.SetStatus(stat); err != nil {
		f.cfg.L.Error(
			"failed to set configuration status for task",
			zap.Uint64("key", uint64(t.Key)),
			zap.String("name", t.Name),
			zap.Error(err),
		)
	}
}

// Close closes the connection pool.
func (f *Factory) Close() error { return f.pool.Close() }

// deviceKeyToLocalKey converts a device key string to a local task key.
// Uses a simple hash to generate a deterministic uint32.
func deviceKeyToLocalKey(deviceKey string) uint32 {
	// FNV-1a hash for deterministic key generation
	var hash uint32 = 2166136261
	for i := 0; i < len(deviceKey); i++ {
		hash ^= uint32(deviceKey[i])
		hash *= 16777619
	}
	// Ensure non-zero (task keys must have non-zero local key)
	if hash == 0 {
		hash = 1
	}
	return hash
}
