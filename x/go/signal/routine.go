// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package signal

import (
	"context"
	"fmt"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/errors"
	"go.uber.org/zap"
	"runtime/pprof"
	"strconv"
)

type RoutineOption func(r *routineOptions)

type RoutineInfo struct {
	// Key is a unique identifier for the routine within the parent context.
	Key string
	// ContextKey is the key of the context that the routine is running in.
	ContextKey string
	// State is the current state of the routine.
	State RoutineState
	// FailureReason is the error that caused the routine to exit.
	FailureReason error
}

func (r RoutineInfo) PrettyString() string {
	return fmt.Sprintf(
		`
Keys: %s
ContextKey: %s
State: %s
FailureReason: %s,
`, r.Key, r.ContextKey, r.State, r.FailureReason)
}

type (
	RoutineState uint8
	PanicPolicy  uint8
	// contextPolicy is a private enum to keep enabling us to use the CancelOnExit()
	// and CancelOnFail() options pattern.
	contextPolicy uint8
)

//go:generate stringer -type=RoutineState
const (
	Starting RoutineState = iota
	Running
	Stopping
	Exited
	Failed
	ContextCanceled
	Panicked
)

//go:generate stringer -type=PanicPolicy
const (
	PropagatePanic PanicPolicy = iota
	RecoverNoErr
	RecoverErr
)

const (
	// CancelOnExit defines if the routine should cancel the context upon exiting.
	cancelOnExit contextPolicy = iota + 1
	// CancelOnFail defines if the routine should cancel the context upon exiting
	cancelOnFail
)

// Defer attaches the provided function f to the routine. The function will be
// executed on routine exit in LIFO style.
func Defer(f func(), opts ...RoutineOption) RoutineOption {
	o := newRoutineOptions(opts)
	return func(r *routineOptions) {
		r.deferrals = append(r.deferrals, deferral{key: o.key, f: func() error { f(); return nil }})
	}
}

// DeferErr attaches the provided function f to the routine. The function will
// be executed on routine exit in LIFO style. If the function returns a non-nil
// error, the routine will fail.
func DeferErr(f func() error, opts ...RoutineOption) RoutineOption {
	o := newRoutineOptions(opts)
	return func(r *routineOptions) {
		r.deferrals = append(r.deferrals, deferral{key: o.key, f: f})
	}
}

func WithKey(key string) RoutineOption { return func(r *routineOptions) { r.key = key } }

func WithKeyf(format string, args ...interface{}) RoutineOption {
	return func(r *routineOptions) {
		r.key = fmt.Sprintf(format, args...)
	}
}

func AddCallerSkip(skip int) RoutineOption {
	return func(r *routineOptions) {
		r.callerSkip = skip
	}
}

type deferral struct {
	key string
	f   func() error
}

func WithPanicPolicy(p PanicPolicy) RoutineOption {
	return func(r *routineOptions) {
		r.panicPolicy = p
	}
}

// WithMaxRestart sets the maximum number of attempted restarts after panicking.
// It also sets the panicPolicy to RecoverErr. If you wish to use a different panicPolicy,
// use the WithMaxRestartAndPanicPolicy function.
func WithMaxRestart(maxRestart int) RoutineOption {
	return func(r *routineOptions) {
		r.maxRestart = maxRestart
		r.panicPolicy = RecoverErr
	}
}

// WithMaxRestartAndPanicPolicy sets the maximum number of attempted restarts and the
// panic policy.
func WithMaxRestartAndPanicPolicy(maxRestart int, p PanicPolicy) RoutineOption {
	return func(r *routineOptions) {
		r.maxRestart = maxRestart
		r.panicPolicy = p
	}
}

// CancelOnExit instructs the goroutine to cancel upon exiting (error or no error)
// If CancelOnFail or CancelOnExit is already called, this overrides the previous
// configuration.
func CancelOnExit() RoutineOption {
	return func(r *routineOptions) {
		r.contextPolicy = cancelOnExit
	}
}

// CancelOnFail instructs the goroutine to cancel upon failing (error)
// If CancelOnFail or CancelOnExit is already called, this overrides the previous
// configuration.
func CancelOnFail() RoutineOption {
	return func(r *routineOptions) {
		r.contextPolicy = cancelOnFail
	}
}

type routineOptions struct {
	// key is a unique identifier for the routine. signal will panic if more
	// than one routine is started with the same key. If no key is provided
	// signal will automatically generate a unique key.
	key string
	// deferrals is a list of functions to be called on routine exit in reverse
	// order.
	deferrals []deferral
	// contextPolicy defines if the routine should cancel the context after
	// exiting.
	contextPolicy contextPolicy
	// panicPolicy defines what the routine should do if it panics.
	panicPolicy PanicPolicy
	// maxRestart defines the maximum number of times a panicking goroutine attempts to
	// restart before its panicPolicy kicks into place.
	maxRestart int
	// callerSkip is the number of stack frames to skip when logging.
	callerSkip int
}

type routine struct {
	ctx *core
	routineOptions
	L *alamos.Logger
	// span traces the goroutine's execution.
	span alamos.Span
	// state represents the current state of the routine
	state struct {
		state RoutineState
		// err is the error that cause the routine to exit.
		err error
	}
}

func (r *routine) info() RoutineInfo {
	return RoutineInfo{
		Key:           r.key,
		State:         r.state.state,
		FailureReason: r.state.err,
	}
}

func (r *routine) runPrelude() (ctx context.Context, proceed bool) {
	r.ctx.mu.Lock()
	defer r.ctx.mu.Unlock()

	if r.key == "" {
		r.key = "anonymous-" + strconv.Itoa(len(r.ctx.mu.routines))
	}

	r.ctx.mu.routines = append(r.ctx.mu.routines, r)

	// If the context has already been canceled, don't even start the routine.
	if r.ctx.Err() != nil {
		r.state.state = Failed
		r.state.err = r.ctx.Err()
		return r.ctx, false
	}
	r.state.state = Starting

	r.ctx.L.Debug("starting routine", r.zapFields()...)
	ctx, r.span = r.ctx.T.Prod(r.ctx, r.path())

	return ctx, true
}

// runPostlude decides the state of the goroutine upon exiting and combines err with
// any errors from deferred functions.
func (r *routine) runPostlude(err error) error {
	r.ctx.L.Debug("stopping routine", r.zapFields()...)

	r.ctx.mu.Lock()
	r.state.state = Stopping
	r.ctx.mu.Unlock()

	for i := range r.deferrals {
		if dErr := r.deferrals[len(r.deferrals)-i-1].f(); dErr != nil {
			err = errors.CombineErrors(err, dErr)
		}
	}

	r.ctx.mu.Lock()
	defer r.ctx.mu.Unlock()

	if err != nil {
		_ = r.span.Error(err, context.Canceled)
		// Only non-context errors are considered failures.
		if errors.IsAny(err, context.Canceled, context.DeadlineExceeded) {
			r.state.state = ContextCanceled
		} else {
			r.state.state = Failed
			r.state.err = err
			r.ctx.L.Error("routine failed", r.zapFields()...)
			r.ctx.L.Debugf(routineFailedFormat, r.key, r.state.err, r.ctx.routineDiagnostics())
		}
		if r.contextPolicy == cancelOnFail {
			r.ctx.cancel()
		}
	} else {
		r.state.state = Exited
	}

	if r.contextPolicy == cancelOnExit {
		r.ctx.cancel()
	}

	r.span.End()

	r.ctx.maybeStop()

	r.ctx.L.Debug("routine stopped", r.zapFields()...)

	return err
}

func (r *routine) maybeRecover(err any) error {
	r.ctx.mu.Lock()
	defer r.ctx.mu.Unlock()

	r.ctx.L.Debugf("routine panicked, handling with strategy %v", r.panicPolicy)
	r.ctx.L.Debugf(routineFailedFormat, r.key, err, r.ctx.routineDiagnostics())

	switch r.panicPolicy {
	case PropagatePanic:
		r.state.state = Panicked
		if err, ok := err.(error); ok {
			r.state.err = err
			_ = r.span.Error(err)
		}
		r.span.End()
		panic(err)
	case RecoverErr:
		r.state.state = Failed
		if err, ok := err.(error); ok {
			return errors.Wrap(err, "routine recovered")
		}
		return errors.Newf("%s", err)
	case RecoverNoErr:
		r.state.state = Exited
		return nil
	default:
		msg := fmt.Sprintf("unknown panic policy %v", r.panicPolicy)
		r.ctx.L.DPanic(msg)
		return errors.Newf(msg)
	}
}

func (r *routine) zapFields() []zap.Field {
	opts := []zap.Field{
		zap.String("key", r.path()),
		zap.Stringer("state", r.state.state),
		zap.Error(r.state.err),
	}
	deferralKeys := make([]string, len(r.deferrals))
	for i, def := range r.deferrals {
		if def.key != "" {
			deferralKeys[i] = def.key
		}
	}
	if len(deferralKeys) > 0 {
		opts = append(opts, zap.Strings("deferrals", deferralKeys))
	}
	return opts
}

func (r *routine) path() string {
	insP := r.ctx.Instrumentation.Meta.Path
	if len(insP) > 0 {
		return insP + "." + r.key
	}
	return r.key
}

func (r *routine) goRun(f func(context.Context) error) {
	if ctx, proceed := r.runPrelude(); proceed {
		pprof.Do(ctx, pprof.Labels("routine", r.path()), func(ctx context.Context) {
			r.ctx.mu.Lock()
			r.state.state = Running
			r.ctx.mu.Unlock()
			r.ctx.internal.Go(func() (err error) {
				for i := 0; i < r.maxRestart+1; i++ {
					restart := false
					if r.maxRestart != 0 && i != r.maxRestart {
						// Before reaching the "last straw", restart the goroutine if
						// recovered from a panic.
						restart = true
					}
					func() {
						defer func() {
							if e := recover(); e != nil {
								if restart {
									return
								}
								err = r.maybeRecover(e)
							}
						}()
						err = f(ctx)
					}()

					if !restart {
						err = r.runPostlude(err)
						return
					}
				}
				return
			})
		})
	}
}

func newRoutine(c *core, opts []RoutineOption) *routine {
	r := &routine{
		ctx:            c,
		L:              c.L,
		routineOptions: newRoutineOptions(opts),
	}
	if r.callerSkip > 0 {
		r.L = c.L.WithOptions(zap.AddCallerSkip(r.callerSkip))
	}
	return r
}

func newRoutineOptions(opts []RoutineOption) routineOptions {
	o := routineOptions{}
	for _, opt := range opts {
		opt(&o)
	}
	return o
}

const routineFailedFormat = `
----------------------------- FAILURE DIAGNOSTICS ------------------------------

key: %s

--------------------------------- ERROR ----------------------------------------

%+v

-------------------------------- ROUTINES --------------------------------------
%s

----------------------------- END DIAGNOSTICS ----------------------------------
`
