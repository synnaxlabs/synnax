// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package rtsp

import (
	"context"
	"time"

	"github.com/bluenviron/gortsplib/v5"
	"github.com/bluenviron/gortsplib/v5/pkg/base"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/synnax/pkg/service/video/ingest"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// FactoryConfig is the configuration for the RTSP task factory.
type FactoryConfig struct {
	// DB opens the transactions that status writes run in.
	//
	// [REQUIRED]
	DB *gorp.DB
	// Channel retrieves the video channels of tasks.
	//
	// [REQUIRED]
	Channel *channel.Service
	// Framer opens the writers of tasks.
	//
	// [REQUIRED]
	Framer *framer.Service
	// Status writes task statuses.
	//
	// [REQUIRED]
	Status *status.Service
	alamos.Instrumentation
	// KeyframeLimit is the longest time that a stream may go without a keyframe
	// before its task warns that no viewer can start to show it.
	//
	// [OPTIONAL] - Defaults to 10s.
	KeyframeLimit time.Duration
}

var (
	_ config.Config[FactoryConfig] = FactoryConfig{}
	// DefaultFactoryConfig is the default configuration for the RTSP task factory.
	DefaultFactoryConfig = FactoryConfig{KeyframeLimit: 10 * time.Second}
)

// Override implements config.Config.
func (c FactoryConfig) Override(other FactoryConfig) FactoryConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.DB = override.Nil(c.DB, other.DB)
	c.Channel = override.Nil(c.Channel, other.Channel)
	c.Framer = override.Nil(c.Framer, other.Framer)
	c.Status = override.Nil(c.Status, other.Status)
	c.KeyframeLimit = override.Numeric(c.KeyframeLimit, other.KeyframeLimit)
	return c
}

// Validate implements config.Config.
func (c FactoryConfig) Validate() error {
	v := validate.New("rtsp.factory")
	v.NotNil("db", c.DB)
	v.NotNil("channel", c.Channel)
	v.NotNil("framer", c.Framer)
	v.NotNil("status", c.Status)
	v.Positive("keyframe_limit", c.KeyframeLimit)
	return v.Error()
}

// Factory configures RTSP read tasks. Safe for concurrent use.
type Factory struct {
	cfg      FactoryConfig
	reporter driver.ConfigReporter
}

var _ driver.Factory = (*Factory)(nil)

// NewFactory validates the configuration and returns a Factory.
func NewFactory(cfgs ...FactoryConfig) (*Factory, error) {
	cfg, err := config.New(DefaultFactoryConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Factory{
		cfg: cfg,
		reporter: driver.ConfigReporter{
			DB:              cfg.DB,
			Status:          cfg.Status,
			Instrumentation: cfg.Instrumentation,
		},
	}, nil
}

// ConfigureTask implements driver.Factory.
func (f *Factory) ConfigureTask(
	ctx context.Context,
	t task.Task,
	cmdKey string,
) (driver.Task, error) {
	if t.Type != ReadTaskType {
		return nil, driver.ErrTaskNotHandled
	}
	var cfg ReadConfig
	if err := t.Config.Unmarshal(&cfg); err != nil {
		f.reporter.Report(ctx, t, cmdKey, false, err)
		return nil, err
	}
	rt, err := f.configureRead(ctx, t, cfg)
	if err != nil {
		f.reporter.Report(ctx, t, cmdKey, cfg.AutoStart, err)
		return nil, err
	}
	// A successful configure writes no status: the start that follows it answers the
	// command, and a "configured" status would answer it first with running false.
	if cfg.AutoStart {
		if err = rt.Start(ctx, driver.NoCommand); err != nil {
			return nil, err
		}
	}
	return rt, nil
}

// InitialTasks implements driver.Factory.
func (f *Factory) InitialTasks() []task.Task { return nil }

// Name implements driver.Factory.
func (f *Factory) Name() string { return "rtsp" }

func (f *Factory) configureRead(
	ctx context.Context,
	t task.Task,
	cfg ReadConfig,
) (*driver.ReadTask, error) {
	u, err := base.ParseURL(cfg.URL)
	if err != nil {
		return nil, errors.Wrapf(validate.ErrValidation, "url: %s", err)
	}
	keys, err := ingest.Channels(ctx, f.cfg.Channel, cfg.Channel)
	if err != nil {
		return nil, err
	}
	src := &source{
		url:      u,
		protocol: transport(cfg.Protocol),
		keys:     keys,
		offset:   cfg.Offset,
		capture:  cfg.TimeSource == TimeSourceCapture,
		limit:    f.cfg.KeyframeLimit,
		ins:      f.cfg.Child(t.Key.String()),
	}
	mode := framer.WriterModePersistStream
	if cfg.DataSavingDisabled {
		mode = framer.WriterModeStreamOnly
	}
	return driver.NewReadTask(driver.ReadTaskConfig{
		Instrumentation: f.cfg.Child(t.Key.String()),
		Source:          src,
		DB:              f.cfg.DB,
		Status:          f.cfg.Status,
		Framer:          f.cfg.Framer,
		Channels:        keys,
		Task:            t,
		Mode:            mode,
	})
}

// transport returns the gortsplib value of p. Automatic returns nil, which tries UDP
// first, then TCP.
func transport(p Protocol) *gortsplib.Protocol {
	switch p {
	case ProtocolAutomatic:
		return nil
	case ProtocolTCP:
		return new(gortsplib.ProtocolTCP)
	case ProtocolUDP:
		return new(gortsplib.ProtocolUDP)
	}
	panic(errors.Newf("rtsp: unknown protocol %q", p))
}
