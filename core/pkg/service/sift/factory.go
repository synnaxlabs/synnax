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

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/secret"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// ScanTaskType is the type of the internal task that checks Sift devices and answers
// their commands.
const ScanTaskType = "sift_scan"

const scanTaskName = "Sift Scanner"

// FactoryConfig is the configuration for the Sift driver factory.
type FactoryConfig struct {
	alamos.Instrumentation
	// DB opens the transactions that status writes run in.
	//
	// [REQUIRED]
	DB *gorp.DB
	// Device retrieves Sift devices.
	//
	// [REQUIRED]
	Device *device.Service
	// Status writes the statuses of tasks and devices.
	//
	// [REQUIRED]
	Status *status.Service
	// Secret reads the API keys of Sift devices.
	//
	// [REQUIRED]
	Secret *secret.Service
	// Timeout bounds one call to Sift from a check or a command.
	//
	// [OPTIONAL] - Defaults to 10 s.
	Timeout telem.TimeSpan
}

var _ config.Config[FactoryConfig] = FactoryConfig{}

// DefaultFactoryConfig is the default configuration for the Sift driver factory.
var DefaultFactoryConfig = FactoryConfig{Timeout: 10 * telem.Second}

// Override implements config.Config.
func (c FactoryConfig) Override(other FactoryConfig) FactoryConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.DB = override.Nil(c.DB, other.DB)
	c.Device = override.Nil(c.Device, other.Device)
	c.Status = override.Nil(c.Status, other.Status)
	c.Secret = override.Nil(c.Secret, other.Secret)
	c.Timeout = override.Numeric(c.Timeout, other.Timeout)
	return c
}

// Validate implements config.Config.
func (c FactoryConfig) Validate() error {
	v := validate.New("sift.factory")
	v.NotNil("db", c.DB)
	v.NotNil("device", c.Device)
	v.NotNil("status", c.Status)
	v.NotNil("secret", c.Secret)
	v.Positive("timeout", c.Timeout)
	return v.Error()
}

type factory struct{ cfg FactoryConfig }

var _ driver.Factory = (*factory)(nil)

// NewFactory returns a driver factory for Sift tasks.
func NewFactory(cfgs ...FactoryConfig) (driver.Factory, error) {
	cfg, err := config.New(DefaultFactoryConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &factory{cfg: cfg}, nil
}

// Name implements driver.Factory.
func (f *factory) Name() string { return Make }

// InitialTasks implements driver.Factory.
func (f *factory) InitialTasks() []task.Task {
	return []task.Task{{
		Name:   scanTaskName,
		Type:   ScanTaskType,
		Config: msgpack.EncodedJSON{},
	}}
}

// ConfigureTask implements driver.Factory.
func (f *factory) ConfigureTask(
	ctx context.Context,
	t task.Task,
	cmdKey string,
) (driver.Task, error) {
	if t.Type != ScanTaskType {
		return nil, driver.ErrTaskNotHandled
	}
	st, autoStart, err := f.configureScan(t)
	if err != nil {
		driver.ConfigReporter{
			DB:              f.cfg.DB,
			Status:          f.cfg.Status,
			Instrumentation: f.cfg.Instrumentation,
		}.Report(ctx, t, cmdKey, false, err)
		return nil, err
	}
	if autoStart {
		if err = st.Start(ctx, driver.NoCommand); err != nil {
			return nil, err
		}
	}
	return st, nil
}

func validateScanConfig(cfg *ScanConfig) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	v := validate.New("sift.scan")
	v.Ternaryf("rate", cfg.Rate.Period() <= 0, "%s is too high", cfg.Rate)
	return v.Error()
}

// configureScan returns the scan task of t, and whether the factory starts it.
func (f *factory) configureScan(t task.Task) (*driver.ScanTask, bool, error) {
	var cfg ScanConfig
	if err := t.Config.Unmarshal(&cfg); err != nil {
		return nil, false, err
	}
	if err := validateScanConfig(&cfg); err != nil {
		return nil, false, err
	}
	st, err := driver.NewScanTask(driver.ScanTaskConfig{
		Instrumentation:    f.cfg.Child("scan"),
		Scanner:            scanner{cfg: f.cfg, rack: t.Rack},
		DB:                 f.cfg.DB,
		Status:             f.cfg.Status,
		Device:             f.cfg.Device,
		Make:               Make,
		ReachableMessage:   "Connected to Sift",
		UnreachableMessage: "Failed to connect to Sift",
		Task:               t,
		Interval:           cfg.Rate.Period().Duration(),
		CheckTimeout:       f.cfg.Timeout.Duration(),
	})
	if err != nil {
		return nil, false, err
	}
	return st, !cfg.Disabled, nil
}
