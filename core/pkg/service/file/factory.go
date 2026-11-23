// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package file

import (
	"context"
	"os"
	"path/filepath"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

const (
	browserTaskType = "file_browser"
	browserTaskName = "File Browser"
	// listCommand lists a folder, or the top level when its path is empty.
	listCommand = "list"
)

// FactoryConfig is the configuration for creating a Factory.
type FactoryConfig struct {
	alamos.Instrumentation
	// DB is the database that the task statuses are stored in.
	// [REQUIRED]
	DB *gorp.DB
	// Status stores the answers of the task.
	// [REQUIRED]
	Status *status.Service
	// SecretsDir is the secrets folder of the Core's rack. A listing never reads the
	// files in it.
	// [OPTIONAL]
	SecretsDir string
}

var _ config.Config[FactoryConfig] = FactoryConfig{}

// Override implements config.Config.
func (c FactoryConfig) Override(other FactoryConfig) FactoryConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.DB = override.Nil(c.DB, other.DB)
	c.Status = override.Nil(c.Status, other.Status)
	c.SecretsDir = override.String(c.SecretsDir, other.SecretsDir)
	return c
}

// Validate implements config.Config.
func (c FactoryConfig) Validate() error {
	v := validate.New("file.factory")
	v.NotNil("db", c.DB)
	v.NotNil("status", c.Status)
	return v.Error()
}

// Factory gives the Core's rack the file browser task, which lists the folders of the
// Core's computer.
type Factory struct {
	cfg     FactoryConfig
	browser browser
}

var _ driver.Factory = (*Factory)(nil)

// NewFactory creates a Factory from the given configurations.
func NewFactory(cfgs ...FactoryConfig) (*Factory, error) {
	cfg, err := config.New(FactoryConfig{}, cfgs...)
	if err != nil {
		return nil, err
	}
	f := &Factory{cfg: cfg}
	if cfg.SecretsDir != "" {
		if f.browser.secretsDir, err = filepath.Abs(cfg.SecretsDir); err != nil {
			return nil, err
		}
	}
	if home, err := os.UserHomeDir(); err == nil {
		f.browser.home = filepath.Clean(home)
	}
	return f, nil
}

// Name implements driver.Factory.
func (*Factory) Name() string { return "file" }

// InitialTasks implements driver.Factory.
func (*Factory) InitialTasks() []task.Task {
	return []task.Task{{
		Name:   browserTaskName,
		Type:   browserTaskType,
		Config: msgpack.EncodedJSON{},
	}}
}

// ConfigureTask implements driver.Factory. The task runs at once.
func (f *Factory) ConfigureTask(
	ctx context.Context,
	t task.Task,
	cmdKey string,
) (driver.Task, error) {
	if t.Type != browserTaskType {
		return nil, driver.ErrTaskNotHandled
	}
	h := driver.NewStatusHandler(f.cfg.DB, f.cfg.Status, t)
	if err := h.Ack(ctx, cmdKey, true); err != nil {
		return nil, err
	}
	c, err := driver.NewCommander(driver.CommanderConfig{
		Instrumentation: f.cfg.Instrumentation,
		Status:          h,
		Exec:            f.browser.exec,
		Message:         "Listed files",
	})
	if err != nil {
		return nil, err
	}
	return c, nil
}
