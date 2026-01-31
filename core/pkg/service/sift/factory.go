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

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/service/device"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
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
	//
	// [REQUIRED]
	Device *device.Service
	// Ranger is used for retrieving range information for uploads.
	//
	// [REQUIRED]
	Ranger *ranger.Service
	// Framer is used for reading/writing telemetry.
	//
	// [REQUIRED]
	Framer *framer.Service
	// Channel is used for retrieving channel metadata.
	//
	// [REQUIRED]
	Channel *channel.Service
	// Status is used for task status updates.
	//
	// [REQUIRED]
	Status *status.Service
	// Task is used for deleting tasks after upload completion.
	//
	// [REQUIRED]
	Task *task.Service
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
	c.Task = override.Nil(c.Task, other.Task)
	return c
}

func (c FactoryConfig) Validate() error {
	v := validate.New("sift.factory")
	validate.NotNil(v, "device", c.Device)
	validate.NotNil(v, "ranger", c.Ranger)
	validate.NotNil(v, "framer", c.Framer)
	validate.NotNil(v, "channel", c.Channel)
	validate.NotNil(v, "status", c.Status)
	validate.NotNil(v, "task", c.Task)
	return v.Error()
}

// Factory creates Sift upload tasks.
type Factory struct {
	cfg  FactoryConfig
	pool *ClientPool
}

var _ driver.Factory = (*Factory)(nil)

// OpenFactory creates a new Sift factory.
func OpenFactory(cfgs ...FactoryConfig) (*Factory, error) {
	cfg, err := config.New(DefaultFactoryConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Factory{cfg: cfg, pool: NewClientPool()}, nil
}

// ConfigureTask creates a Sift upload task.
func (f *Factory) ConfigureTask(ctx driver.Context, t task.Task) (driver.Task, error) {
	if t.Type != TaskType {
		return nil, driver.ErrTaskNotHandled
	}

	var cfg TaskConfig
	if err := json.Unmarshal([]byte(t.Config), &cfg); err != nil {
		f.setStatus(ctx, t, xstatus.VariantError, err.Error())
		return nil, err
	}

	// Retrieve device for connection properties
	var dev device.Device
	if err := f.cfg.Device.NewRetrieve().
		WhereKeys(cfg.DeviceKey).
		Entry(&dev).
		Exec(ctx, nil); err != nil {
		f.setStatus(ctx, t, xstatus.VariantError, err.Error())
		return nil, err
	}

	props, err := ParseDeviceProperties(dev.Properties)
	if err != nil {
		f.setStatus(ctx, t, xstatus.VariantError, err.Error())
		return nil, err
	}

	task := newUploadTask(t, cfg, props, f.cfg, f.pool)
	f.setStatus(ctx, t, xstatus.VariantSuccess, "Task configured")
	return task, nil
}

func (f *Factory) setStatus(ctx driver.Context, t task.Task, variant xstatus.Variant, message string) {
	stat := task.Status{
		Key:     task.OntologyID(t.Key).String(),
		Name:    t.Name,
		Variant: variant,
		Message: message,
		Time:    telem.Now(),
		Details: task.StatusDetails{Task: t.Key, Running: false},
	}
	if err := ctx.SetStatus(stat); err != nil {
		f.cfg.L.Error("failed to set status", zap.Uint64("key", uint64(t.Key)), zap.Error(err))
	}
}

// Name returns the factory name.
func (f *Factory) Name() string { return "Sift" }

// Close closes the connection pool.
func (f *Factory) Close() error { return f.pool.Close() }
