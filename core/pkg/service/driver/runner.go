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

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/signal"
	"go.uber.org/zap"
)

// runner is the start and stop lifecycle that ReadTask and WriteTask share. It runs
// one goroutine between a start and the stop that follows it, answers both commands,
// and reports a run that ends on its own. Safe for concurrent use.
type runner struct {
	status *StatusHandler
	// open acquires what a run needs. Its error answers the start command.
	open func(ctx context.Context) error
	// run blocks until ctx is cancelled or the run fails. It releases what open
	// acquired before it returns.
	run func(ctx context.Context) error
	ins alamos.Instrumentation
	// lifecycle serializes start and stop, so open never runs while the previous run
	// still releases its resources.
	lifecycle sync.Mutex
	mu        struct {
		// active is the current run. It is nil when the task is stopped.
		active *activeRun
		sync.Mutex
	}
}

type activeRun struct {
	sCtx   signal.Context
	cancel context.CancelFunc
}

func (r *runner) exec(ctx context.Context, cmd task.Command) error {
	switch cmd.Type {
	case startCommandType:
		return r.start(ctx, cmd.Key)
	case stopCommandType:
		return r.stop(ctx, cmd.Key, true)
	}
	return ErrUnsupportedCommand
}

// release forgets active and reports whether it was still the current run. The
// caller that gets true owns the terminal status of the run.
func (r *runner) release(active *activeRun) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if active == nil || r.mu.active != active {
		return false
	}
	r.mu.active = nil
	return true
}

func (r *runner) current() *activeRun {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.mu.active
}

func (r *runner) start(ctx context.Context, cmdKey string) error {
	r.lifecycle.Lock()
	defer r.lifecycle.Unlock()
	if r.current() != nil {
		return r.status.Ack(ctx, cmdKey, true)
	}
	if err := r.open(ctx); err != nil {
		return errors.Combine(
			err,
			r.status.Send(ctx, cmdKey, status.VariantError, false, err.Error()),
		)
	}
	active := &activeRun{}
	active.sCtx, active.cancel = signal.Isolated(signal.WithInstrumentation(r.ins))
	r.mu.Lock()
	r.mu.active = active
	r.mu.Unlock()
	// The running status goes out before the run starts, so a run that fails at once
	// cannot have its error status overwritten.
	err := r.status.Send(
		ctx, cmdKey, status.VariantSuccess, true, "Task started successfully",
	)
	active.sCtx.Go(func(ctx context.Context) error {
		err := r.run(ctx)
		// A stop that already released the run waits on this error and reports it.
		if !r.release(active) {
			return err
		}
		active.cancel()
		message := "Task stopped"
		if err != nil {
			message = err.Error()
		}
		if sErr := r.status.Send(
			context.TODO(), NoCommand, status.VariantError, false, message,
		); sErr != nil {
			r.ins.L.Error("failed to write error status", zap.Error(sErr))
		}
		return nil
	}, signal.RecoverWithErrOnPanic())
	return err
}

func (r *runner) stop(ctx context.Context, cmdKey string, sendStatus bool) error {
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
		return errors.Combine(
			err,
			r.status.Send(ctx, cmdKey, status.VariantError, false, err.Error()),
		)
	}
	return r.status.Send(
		ctx, cmdKey, status.VariantSuccess, false, "Task stopped successfully",
	)
}

// report writes the health of a run to the task status: a warning for an error that
// matches ErrTemporary, and the status that the warning replaced for a nil error.
func (r *runner) report(ctx context.Context, err error) {
	var sErr error
	if err == nil {
		sErr = r.status.ClearWarning(ctx)
	} else {
		sErr = r.status.Warn(ctx, err.Error(), "")
	}
	if sErr != nil {
		r.ins.L.Error("failed to write task health status", zap.Error(sErr))
	}
}
