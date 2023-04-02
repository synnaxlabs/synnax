// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package alamos

import (
	"context"
	"go.uber.org/zap"
)

// |||||| EXPERIMENT ||||||

// Instrumentation is alamos' core data type. It represents a hierarchical collection of application metrics.
// Instrumentation is a tree-like structure where each node is either a metric or a Sub-instrumentation.
//
// Creating Experiments:
//
// SinkTarget create an instrumentation, use alamos.New().
//
// Metrics:
//
// SinkTarget add a metric, use one of the metric constructors. Available metrics are:
//
//   - alamos.NewGauge
//   - alamos.NewSeries
//   - alamos.NewGaugeDuration
//   - alamos.NewSeriesDuration
//
// Each metric in an Instrumentation is uniquely identified by a string key. The instrumentation-key combination is used to
// identify the metric in generated reports.
//
// Empty Experiments:
//
// Alamos is designed to be used alongside production code. This means that it is possible to pass nil Experiments
// throughout an application. If a Metric is registered with an empty (nil) Instrumentation, all of its behavior will appear
// to remain the same, but the Metric will not allocate any memory or record any values. For example:
//
//		var exp alamos.Instrumentation
//	    // This gauge will appear to behave normally, but will not allocate memory or record values.
//		g := exp.NewGauge(exp, "bar")
//		g.Frame(1)
//
// The same principle applies for Sub-experiments. If a parent Instrumentation is empty and Sub is called, the returned
// Sub-instrumentation will be empty as well.
//
// When approaching empty experiments, we considered taking a route similar to zap.NewNop(), but because alamos
// makes extensive use of generics, and methods can't have type parameters, we decided to try tolerating nil
// experiments instead.
//
// Organizing Experiments:
//
// Only one top-level instrumentation should be created per application. Sub-experiments should be created to separate
// individual application concerns.
type Instrumentation interface {
	Report() Report
	L() *zap.Logger
	S() *zap.SugaredLogger
	filterTest(level Level) bool
	sub(string) Instrumentation
	getMetric(string) baseMetric
	addMetric(metric baseMetric)
	attachReporter(string, Level, Reporter)
	tracer
}

// WithContext returns new Instrumentation with the given key and options and a context
// with the Instrumentation attached.
func WithContext(ctx context.Context, key string, opts ...Option) (Instrumentation, context.Context) {
	inst := FromContext(ctx)
	inst.sub(key)
	return inst, ctx
}

// Attach attaches the given Instrumentation to the given context.
func Attach(ctx context.Context, ins Instrumentation) context.Context {
	return context.WithValue(ctx, contextKey, ins)
}

// Transfer retrieves the Instrumentation from the source context, and attaches it to
// a new context with the target context as its parent.
func Transfer(source, target context.Context) context.Context {
	return Attach(target, FromContext(source))
}

func Nop() (context.Context, Instrumentation) {
	return &instrumentation{
		options: &options{
			instrumentation: zap.NewNop().Sugar(),
		},
	}
}

func New(key string, opts ...Option) (Instrumentation, context.Context) {
	o := newOptions(opts...)
	return &instrumentation{
		key:          key,
		children:     make(map[string]Instrumentation),
		measurements: make(map[string]baseMetric),
		reporters:    make(map[string]Reporter),
		options:      o,
	}
}

const contextKey = "alamos-instrumentation"

func FromContext(ctx context.Context) Instrumentation {
	v := ctx.Value(contextKey)
	if v == nil {
		return nil
	}
	return v.(Instrumentation)
}

func fromContext(ctx context.Context) (*instrumentation, bool) {
	v := ctx.Value(contextKey)
	if v == nil {
		return nil, false
	}
	return v.(*instrumentation), true
}

func RetrieveMetric[T any](exp Instrumentation, key string) Metric[T] {
	if exp == nil {
		return nil
	}
	return exp.getMetric(key).(Metric[T])
}

type instrumentation struct {
	logger       *Logger
	options      *options
	key          string
	children     map[string]Instrumentation
	measurements map[string]baseMetric
	reports      map[string]Report
	reporters    map[string]Reporter
}

func (e *instrumentation) Key() string {
	return e.key
}

func (e *instrumentation) sub(key string) Instrumentation {
	exp := New(key)
	e.addSub(key, exp)
	return exp
}

func (e *instrumentation) getMetric(key string) baseMetric {
	return e.measurements[key]
}

func (e *instrumentation) addMetric(m baseMetric) {
	e.measurements[m.Key()] = m
}

func (e *instrumentation) addSub(key string, exp Instrumentation) Instrumentation {
	e.children[key] = exp
	return exp
}

func (e *instrumentation) filterTest(level Level) bool {
	for _, filter := range e.options.filters {
		if filter.Test(level) {
			return true
		}
	}
	return false
}
