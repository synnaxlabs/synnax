// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package driver

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"sync"
	"time"

	"github.com/synnaxlabs/alamos"
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

// CommanderConfig is the configuration for creating a Commander.
type CommanderConfig struct {
	alamos.Instrumentation
	// Status answers the commands.
	//
	// [REQUIRED]
	Status *StatusHandler
	// Exec runs a command and returns the data that answers it. The data answers the
	// command even when the error is not nil. A Failure gives the message and the
	// description of the answer. The text of any other error is the description. It
	// returns ErrUnsupportedCommand for a command it does not know. The Commander runs
	// one Exec at a time, and cancels ctx when it stops.
	//
	// [REQUIRED]
	Exec func(ctx context.Context, cmd task.Command) (msgpack.EncodedJSON, error)
	// Message is the message of an answer that succeeds.
	//
	// [REQUIRED]
	Message string
}

var _ config.Config[CommanderConfig] = CommanderConfig{}

// Override implements config.Config.
func (c CommanderConfig) Override(other CommanderConfig) CommanderConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.Status = override.Nil(c.Status, other.Status)
	c.Exec = override.Nil(c.Exec, other.Exec)
	c.Message = override.String(c.Message, other.Message)
	return c
}

// Validate implements config.Config.
func (c CommanderConfig) Validate() error {
	v := validate.New("driver.commander")
	v.NotNil("status", c.Status)
	v.NotNil("exec", c.Exec)
	v.NotEmptyString("message", c.Message)
	return v.Error()
}

// Commander answers the commands of a task apart from the command loop of the driver,
// so a slow command does not hold up the commands of other tasks. Safe for concurrent
// use.
type Commander struct {
	cfg CommanderConfig
	// turn holds a token while a command runs.
	turn     chan struct{}
	commands struct {
		// sCtx runs the commands. It is nil when no command ran since the last stop.
		sCtx   signal.Context
		cancel context.CancelFunc
		sync.Mutex
	}
}

var _ Task = (*Commander)(nil)

// NewCommander validates the configuration and returns a Commander.
func NewCommander(cfgs ...CommanderConfig) (*Commander, error) {
	cfg, err := config.New(CommanderConfig{}, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Commander{cfg: cfg, turn: make(chan struct{}, 1)}, nil
}

// Exec implements Task. The command goes to Exec of the configuration in the
// background, and Exec returns nil at once. Commands take turns in no fixed order. The
// result answers the command, within the time limit of ctx, which starts when the
// command gets its turn. A command that Exec does not support gets an error.
func (c *Commander) Exec(ctx context.Context, cmd task.Command) error {
	deadline, limited := ctx.Deadline()
	limit := time.Until(deadline)
	c.commands.Lock()
	defer c.commands.Unlock()
	if c.commands.sCtx == nil {
		c.commands.sCtx, c.commands.cancel = signal.Isolated(
			signal.WithInstrumentation(c.cfg.Instrumentation),
		)
	}
	c.commands.sCtx.Go(func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			return nil
		case c.turn <- struct{}{}:
		}
		defer func() { <-c.turn }()
		if limited {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, limit)
			defer cancel()
		}
		c.answer(ctx, cmd)
		return nil
	}, signal.RecoverWithErrOnPanic())
	return nil
}

// answer runs cmd and answers it. A command cancelled by a stop gets no answer, since
// the successor of the task owns its status.
func (c *Commander) answer(ctx context.Context, cmd task.Command) {
	data, err := c.cfg.Exec(ctx, cmd)
	if errors.Is(ctx.Err(), context.Canceled) {
		return
	}
	if sErr := checkSize(data); sErr != nil {
		data, err = nil, sErr
	}
	var (
		variant     = status.VariantSuccess
		message     = c.cfg.Message
		description string
		f           Failure
	)
	switch {
	case errors.Is(err, ErrUnsupportedCommand):
		c.cfg.L.Warn(
			"unsupported command",
			zap.Stringer("command", cmd),
			zap.Stringer("task", cmd.Task),
		)
		variant = status.VariantError
		message = failureMessage(cmd.Type)
		description = unsupportedDescription(cmd.Type)
	case errors.As(err, &f):
		variant, message, description = status.VariantError, f.Message, f.Description
	case err != nil:
		variant = status.VariantError
		message, description = failureMessage(cmd.Type), err.Error()
	}
	if rErr := c.cfg.Status.Reply(
		context.WithoutCancel(ctx), cmd.Key, variant, message, description, data,
	); rErr != nil {
		c.cfg.L.Error("failed to answer command", zap.Error(rErr))
	}
}

// checkSize returns a Failure when data holds more than replyLimit bytes of JSON.
func checkSize(data msgpack.EncodedJSON) error {
	encoded, err := json.Marshal(data)
	if err != nil || len(encoded) <= replyLimit {
		return err
	}
	return Failure{
		Message: "Answer too large",
		Description: fmt.Sprintf(
			"The answer holds %.1f MiB of JSON, and a command answers with at most "+
				"1 MiB. Ask for less data, such as fewer topics.",
			float64(len(encoded))/replyLimit,
		),
	}
}

// Stop implements Task. It cancels the commands and waits for them to return. It writes
// no status.
func (c *Commander) Stop(bool) error {
	c.commands.Lock()
	sCtx, cancel := c.commands.sCtx, c.commands.cancel
	c.commands.sCtx, c.commands.cancel = nil, nil
	c.commands.Unlock()
	if sCtx == nil {
		return nil
	}
	cancel()
	return errors.Skip(sCtx.Wait(), context.Canceled)
}
