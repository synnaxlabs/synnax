// Copyright 2025 Synnax Labs, Inc.
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
	"runtime/pprof"
	"strconv"
	"time"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/breaker"
	"github.com/synnaxlabs/x/errors"
	"go.uber.org/zap"
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
	panicPolicy  uint8
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

//go:generate stringer -type=panicPolicy
const (
	propagatePanic panicPolicy = iota
	recoverNoErr
	recoverErr
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

// WithKey attaches a key to identify the routine.
func WithKey(key string) RoutineOption { return func(r *routineOptions) { r.key = key } }

// WithKeyf attaches a formatted string to identify the routine.
func WithKeyf(format string, args ...any) RoutineOption {
	return func(r *routineOptions) {
		r.key = fmt.Sprintf(format, args...)
	}
}

type deferral struct {
	key string
	f   func() error
}

// RecoverWithErrOnPanic instructs the goroutine to recover if it panics, and fail with
// the panic as an error.
// Setting RecoverWithErrOnPanic will override a previously set panic policy.
func RecoverWithErrOnPanic() RoutineOption {
	return func(r *routineOptions) {
		r.panicPolicy = recoverErr
	}
}

// RecoverWithoutErrOnPanic instructs the goroutine to recover if it panics, and exit
// without error.
// Setting RecoverWithoutErrOnPanic will override a previously set panic policy.
func RecoverWithoutErrOnPanic() RoutineOption {
	return func(r *routineOptions) {
		r.panicPolicy = recoverNoErr
	}
}

// WithBreaker sets the breaker to use to enable the goroutine to attempt to rerun
// despite a panic. The breaker controls the interval between two reruns as well as the
// coefficient by which the interval grows.
// When WithBreaker is used, the PanicPolicy of recoverErr is automatically used.
func WithBreaker(breakerCfg breaker.Config) RoutineOption {
	return func(r *routineOptions) {
		r.breakerCfg = breakerCfg
		r.useBreaker = true
		r.panicPolicy = recoverErr
	}
}

// WithBaseRetryInterval sets the base interval for the breaker used to restart the
// goroutine. The base retry interval is how much time the breaker waits before trying
// to restart for the first time. (Default: 1 second)
func WithBaseRetryInterval(retryInterval time.Duration) RoutineOption {
	return func(r *routineOptions) {
		r.breakerCfg.BaseInterval = retryInterval
		r.useBreaker = true
	}
}

// WithRetryOnPanic attempts to recover from a panicking goroutine and restarts it.
// If an argument is passed into it, it retries for the specified amount of time and
// exits with an error if it panics on its last attempt.
// If at any retry the goroutine exits with or without error, the goroutine exits and
// no longer attempts to restart.
func WithRetryOnPanic(maxRetries ...int) RoutineOption {
	return func(r *routineOptions) {
		if len(maxRetries) == 0 {
			r.breakerCfg.MaxRetries = breaker.InfiniteRetries
		} else {
			r.breakerCfg.MaxRetries = maxRetries[0]
		}
		r.useBreaker = true
		r.panicPolicy = recoverErr
	}
}

// WithRetryScale sets the scale on the breaker used to restart the goroutine. The scale
// defines the rate by which the interval between two retries grow. (Default: 1)
func WithRetryScale(scale float32) RoutineOption {
	return func(r *routineOptions) {
		r.breakerCfg.Scale = scale
		r.useBreaker = true
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
	panicPolicy panicPolicy
	// useBreaker determines whether a breaker is used in this routine to attempt to
	// restart on panic.
	useBreaker bool
	// breakerCfg is used to direct control flow in the case where the routine panics.
	breakerCfg breaker.Config
	// callerSkip is the number of stack frames to skip when logging.
	callerSkip int
}

type routine struct {
	ctx *core
	routineOptions
	L *alamos.Logger
	// span traces the goroutine's execution.
	span alamos.Span
	// breaker is the circuit breaker used in the goroutine
	breaker breaker.Breaker
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
		r.key = "anonymous_" + strconv.Itoa(len(r.ctx.mu.routines))
	}

	if r.useBreaker {
		r.breaker, r.state.err = breaker.NewBreaker(r.ctx, r.breakerCfg)
		if r.state.err != nil {
			return r.ctx, false
		}
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
	r.ctx.mu.Lock()
	r.ctx.L.Debug("stopping routine", r.zapFields()...)
	r.state.state = Stopping
	r.ctx.mu.Unlock()

	for i := range r.deferrals {
		if dErr := r.deferrals[len(r.deferrals)-i-1].f(); dErr != nil {
			err = errors.Combine(err, dErr)
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

func (r *routine) maybeRecover(panicReason any) error {
	r.ctx.mu.Lock()
	defer r.ctx.mu.Unlock()

	zapFields := []zap.Field{
		zap.Stringer("recovery_strategy", r.panicPolicy),
	}
	if err, ok := panicReason.(error); ok {
		zapFields = append(zapFields, zap.Error(err))
	}
	r.ctx.L.Error("routine panicked", zapFields...)
	r.ctx.L.Debugf(routineFailedFormat, r.key, panicReason, r.ctx.routineDiagnostics())

	switch r.panicPolicy {
	case propagatePanic:
		r.state.state = Panicked
		if err, ok := panicReason.(error); ok {
			r.state.err = err
			_ = r.span.Error(err)
		}
		r.span.End()
		panic(panicReason)
	case recoverErr:
		r.state.state = Failed
		if err, ok := panicReason.(error); ok {
			return errors.Wrapf(err, "routine %s recovered", r.key)
		}
		return errors.Newf("%s", panicReason)
	case recoverNoErr:
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
		zap.Strings("running", r.ctx.unsafeRunningKeys()),
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
	insP := r.ctx.Meta.Path
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
				for {
					recovered := false
					func() {
						defer func() {
							if e := recover(); e != nil {
								recovered = true
								err = r.maybeRecover(e)
							}
						}()
						err = f(ctx)
					}()
					if !recovered || !r.useBreaker || !r.breaker.Wait() {
						break
					}
				}
				err = r.runPostlude(err)
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
