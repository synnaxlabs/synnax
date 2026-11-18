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
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

const (
	// BrowserTaskType is the type of the file browser task.
	BrowserTaskType = "file_browser"
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
	// Status stores the replies of the task.
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
		Type:   BrowserTaskType,
		Config: msgpack.EncodedJSON{},
	}}
}

// ConfigureTask implements driver.Factory. The task runs at once.
func (f *Factory) ConfigureTask(
	ctx context.Context,
	t task.Task,
	cmdKey string,
) (driver.Task, error) {
	if t.Type != BrowserTaskType {
		return nil, driver.ErrTaskNotHandled
	}
	bt := &browserTask{
		Instrumentation: f.cfg.Instrumentation,
		browser:         f.browser,
		status:          driver.NewStatusHandler(f.cfg.DB, f.cfg.Status, t),
	}
	if err := bt.status.Ack(ctx, cmdKey, true); err != nil {
		return nil, err
	}
	bt.lists, bt.cancel = signal.Isolated(
		signal.WithInstrumentation(f.cfg.Instrumentation),
	)
	return bt, nil
}

type browserTask struct {
	alamos.Instrumentation
	status  *driver.StatusHandler
	lists   signal.Context
	cancel  context.CancelFunc
	browser browser
}

var _ driver.Task = (*browserTask)(nil)

// Exec implements driver.Task. A list runs in the background and Exec returns at once,
// so a slow folder does not hold up the commands of other tasks. The listing answers
// the command as the data of the task status, within the time limit of ctx.
func (t *browserTask) Exec(ctx context.Context, cmd task.Command) error {
	if cmd.Type != listCommand {
		return driver.ErrUnsupportedCommand
	}
	deadline, limited := ctx.Deadline()
	t.lists.Go(func(ctx context.Context) error {
		if limited {
			var cancel context.CancelFunc
			ctx, cancel = context.WithDeadline(ctx, deadline)
			defer cancel()
		}
		t.answer(ctx, cmd)
		return nil
	}, signal.RecoverWithErrOnPanic())
	return nil
}

// answer lists for cmd and answers it. A list that Stop cancels gets no answer, since
// the successor of the task owns its status.
func (t *browserTask) answer(ctx context.Context, cmd task.Command) {
	l, err := t.list(ctx, cmd.Args)
	if errors.Is(ctx.Err(), context.Canceled) {
		return
	}
	var data msgpack.EncodedJSON
	if err == nil {
		data, err = msgpack.NewEncodedJSON(l)
	}
	var (
		variant     = status.VariantSuccess
		message     = "Listed files"
		description string
		f           driver.Failure
	)
	switch {
	case errors.As(err, &f):
		variant, message, description = status.VariantError, f.Message, f.Description
	case err != nil:
		variant, message = status.VariantError, "Failed to list files"
		description = err.Error()
	}
	if err = t.status.Reply(
		context.WithoutCancel(ctx), cmd.Key, variant, message, description, data,
	); err != nil {
		t.L.Error(
			"failed to answer command",
			zap.Stringer("command", cmd),
			zap.Error(err),
		)
	}
}

func (t *browserTask) list(
	ctx context.Context,
	args msgpack.EncodedJSON,
) (listing, error) {
	var parsed struct {
		Path string `json:"path"`
	}
	if err := args.Unmarshal(&parsed); err != nil {
		return listing{}, failure("The path must be a string.")
	}
	return t.browser.list(ctx, parsed.Path)
}

// Stop implements driver.Task. It cancels the lists that run and writes no status.
func (t *browserTask) Stop(bool) error {
	t.cancel()
	return errors.Skip(t.lists.Wait(), context.Canceled)
}
