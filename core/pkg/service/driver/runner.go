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
	"sync"
	"uuid"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

// RunnerConfig is the configuration for a Runner.
type RunnerConfig struct {
	// DB opens the transactions that status writes run in.
	//
	// [REQUIRED]
	DB *gorp.DB
	// Status writes the statuses of the task.
	//
	// [REQUIRED]
	Status *status.Service
	alamos.Instrumentation
	// Task is the stored task this instance runs.
	//
	// [REQUIRED]
	Task task.Task
	// Open acquires what a run needs and returns the run, which owns it. The run blocks
	// until ctx is cancelled or the run fails, and releases what Open acquired before
	// it returns. The error of Open answers the start command.
	//
	// [REQUIRED]
	Open func(ctx context.Context) (run func(ctx context.Context) error, err error)
}

var _ config.Config[RunnerConfig] = RunnerConfig{}

// Override implements config.Config.
func (c RunnerConfig) Override(other RunnerConfig) RunnerConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.DB = override.Nil(c.DB, other.DB)
	c.Status = override.Nil(c.Status, other.Status)
	c.Open = override.Nil(c.Open, other.Open)
	if other.Task.Key != uuid.Nil() {
		c.Task = other.Task
	}
	return c
}

// Validate implements config.Config.
func (c RunnerConfig) Validate() error {
	v := validate.New("driver.runner")
	v.NotNil("db", c.DB)
	v.NotNil("status", c.Status)
	v.NotNil("open", c.Open)
	v.Ternary("task", c.Task.Key == uuid.Nil(), "must have a key")
	return v.Error()
}

// Runner is the start and stop lifecycle of a task. It runs one goroutine between a
// start and the stop that follows it, answers both commands, and reports a run that
// ends on its own. ReadTask, WriteTask, and ScanTask are built on it; a task that
// streams no channels uses it directly. Safe for concurrent use.
type Runner struct {
	status *StatusHandler
	cfg    RunnerConfig
	// lifecycle serializes start and stop, so Open never runs while the previous run
	// still releases its resources.
	lifecycle sync.Mutex
	// mu guards active. A run that ends on its own holds mu until its status is
	// written, so a start that follows reports running after it.
	mu struct {
		// active is the current run. It is nil when the task is stopped.
		active *activeRun
		sync.Mutex
	}
}

var _ Task = (*Runner)(nil)

type activeRun struct {
	sCtx   signal.Context
	cancel context.CancelFunc
}

// NewRunner validates the configuration and returns a stopped Runner.
func NewRunner(cfgs ...RunnerConfig) (*Runner, error) {
	cfg, err := config.New(RunnerConfig{}, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Runner{
		cfg:    cfg,
		status: NewStatusHandler(cfg.DB, cfg.Status, cfg.Task),
	}, nil
}

// Exec implements Task.
func (r *Runner) Exec(ctx context.Context, cmd task.Command) error {
	switch cmd.Type {
	case startCommandType:
		return r.Start(ctx, cmd.Key)
	case stopCommandType:
		return r.stop(ctx, cmd.Key, true)
	}
	return ErrUnsupportedCommand
}

// Stop implements Task.
func (r *Runner) Stop(sendStatus bool) error {
	return r.stop(context.TODO(), NoCommand, sendStatus)
}

// release forgets active and reports whether it was still the current run. The
// caller that gets true owns the terminal status of the run.
func (r *Runner) release(active *activeRun) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if active == nil || r.mu.active != active {
		return false
	}
	r.mu.active = nil
	return true
}

// end releases active, which ended on its own with err, and writes its error status.
// It returns false when a stop already released active.
func (r *Runner) end(active *activeRun, err error) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.mu.active != active {
		return false
	}
	r.mu.active = nil
	active.cancel()
	message, description := "Task stopped", ""
	if err != nil {
		message, description = "Failed to run task", err.Error()
	}
	if sErr := r.status.Send(
		context.TODO(), NoCommand, status.VariantError, false, message, description,
	); sErr != nil {
		r.cfg.L.Error("failed to write error status", zap.Error(sErr))
	}
	return true
}

func (r *Runner) current() *activeRun {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.mu.active
}

// Start opens and starts a run, and answers cmdKey. A factory calls it with NoCommand
// for a task that starts automatically.
func (r *Runner) Start(ctx context.Context, cmdKey string) error {
	r.lifecycle.Lock()
	defer r.lifecycle.Unlock()
	if r.current() != nil {
		return r.status.Ack(ctx, cmdKey, true)
	}
	run, err := r.cfg.Open(ctx)
	if err != nil {
		return errors.Combine(err, r.status.Send(
			ctx,
			cmdKey,
			status.VariantError,
			false,
			"Failed to start task",
			err.Error(),
		))
	}
	active := &activeRun{}
	active.sCtx, active.cancel = signal.Isolated(
		signal.WithInstrumentation(r.cfg.Instrumentation),
	)
	r.mu.Lock()
	r.mu.active = active
	r.mu.Unlock()
	// The running status goes out before the run starts, so a run that fails at once
	// cannot have its error status overwritten.
	err = r.status.Send(ctx, cmdKey, status.VariantSuccess, true, "Started task", "")
	active.sCtx.Go(func(ctx context.Context) error {
		err := r.runRecovered(ctx, run)
		// A stop that already released the run waits on this error and reports it.
		if !r.end(active, err) {
			return err
		}
		return nil
	})
	if err == nil {
		return nil
	}
	// A caller that gets an error drops the task, so the run must not outlive it.
	if r.release(active) {
		active.cancel()
	}
	if wErr := active.sCtx.Wait(); !errors.Is(wErr, context.Canceled) {
		err = errors.Combine(err, wErr)
	}
	return err
}

// runRecovered returns the error of run, or an error for its panic.
func (r *Runner) runRecovered(
	ctx context.Context,
	run func(context.Context) error,
) (err error) {
	defer func() {
		p := recover()
		if p == nil {
			return
		}
		r.cfg.L.Error("task panicked", zap.Any("panic", p), zap.Stack("stack"))
		if pErr, ok := p.(error); ok {
			err = errors.Wrap(pErr, "task panicked")
		} else {
			err = errors.Newf("task panicked: %v", p)
		}
	}()
	return run(ctx)
}

func (r *Runner) stop(ctx context.Context, cmdKey string, sendStatus bool) error {
	r.lifecycle.Lock()
	defer r.lifecycle.Unlock()
	active := r.current()
	if !r.release(active) {
		if sendStatus {
			return r.status.Ack(ctx, cmdKey, false)
		}
		return nil
	}
	active.cancel()
	err := active.sCtx.Wait()
	if errors.Is(err, context.Canceled) {
		err = nil
	}
	if !sendStatus {
		return err
	}
	if err != nil {
		return errors.Combine(err, r.status.Send(
			ctx, cmdKey, status.VariantError, false, "Failed to stop task", err.Error(),
		))
	}
	return r.status.Send(ctx, cmdKey, status.VariantSuccess, false, "Stopped task", "")
}

// Report writes the health of a run to the task status: a warning with message and the
// text of a non-nil err, and the status that the warning replaced for a nil err.
func (r *Runner) Report(ctx context.Context, message string, err error) {
	var sErr error
	if err == nil {
		sErr = r.status.ClearWarning(ctx)
	} else {
		sErr = r.status.Warn(ctx, message, err.Error())
	}
	if sErr != nil {
		r.cfg.L.Error("failed to write task health status", zap.Error(sErr))
	}
}

// ReportError writes an error status for a run that keeps running. The status stays
// until the next start or stop: Report does not clear it.
func (r *Runner) ReportError(ctx context.Context, message, description string) {
	if err := r.status.Send(
		ctx, NoCommand, status.VariantError, true, message, description,
	); err != nil {
		r.cfg.L.Error("failed to write task error status", zap.Error(err))
	}
}
