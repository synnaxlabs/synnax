// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package alamos provides a collection of tools for instrumenting distributed services.
// These tools cover all three pillars of observability: logs, traces, and metrics.
// The intent of alamos is not to provide implementations for these pillars, but instead
// provided an opinionated interface for integrating with existing providers.
//
// The core data type of alamos is the Instrumentation type, which is an aggregation
// of the Logger (L), Tracer (T), and Reporter (R) services.
//
// The Logger (L) is an enhanced version of zap's logger that provides no-op logging
// when nil.
//
// The Tracer (T) is uses both open-telemetry and go's internal tracing APIs to provide
// tracing. It can also propagate tracing information across RPC boundaries.
//
// The Reporter (R) allows for attaching metadata to the instrumentation, and is ideal for
// recording the configuration of the application.
package alamos

// Instrumentation is the alamos core data type, and represents a collection of
// instrumentation tools: a logger, a devTracer, and a reporter.
//
// The zero-value represents a no-op instrumentation that does no logging, tracing,
// or reporting. We recommend embedding Instrumentation within the configs
// of your services, like the following:
//
//	type MyServiceConfig struct {
//		alamos.Instrumentation
//	     // ...other fields
//	}
//
// This provides access to instrumentation tools directly in the config:
//
//	cfg := MyServiceConfig{}
//	// cfg.L is a no-op logger
//	cfg.L.Debug("hello world")
//
// To instantiate the config with instrumentation, use the alamos.New function along with
// the With* options:
//
//	// define a tracing config
//	tracingConfig := tracing.Properties{}
//	devTracer := tracing.NewTracer(tracingConfig)
//	ins := alamos.Name(alamos.WithTracer(devTracer))
//
// Use the same approach to configure the Logger and Reporter.
//
// Instrumentation is organized in a hierarchy, where the Child method can be used to
// create child instrumentation of its parent. This allows for instrumentation to match
// the architecture of the application. For example, instrumentation that tracks
// low-level db requests could be created with ins.Child("db"), and instrumentation
// tracking api requests could be created with ins.Child("api"). See the Child method
// for more details.
type Instrumentation struct {
	// L is the Logger used by this instrumentation. This field should be considered
	// read-only.
	L *Logger
	// T is the Tracer used by this instrumentation. This field should be considered
	// read-only.
	T *Tracer
	// R is the Reporter used by this instrumentation. This field should be considered
	// read-only.
	R *Reporter
	// Meta is the Metadata associated with this instrumentation. This field should be
	// considered read-only.
	Meta InstrumentationMeta
	// children stores all child instrumentation created by this instrumentation.
	children map[string]Instrumentation
}

// New instantiates new Instrumentation with the given key and options. The returned
// instrumentation is considered the 'root' instrumentation, and can be used to create
// child instrumentation with the Child method.
func New(key string, options ...Option) Instrumentation {
	ins := Instrumentation{Meta: InstrumentationMeta{Key: key, Path: key}}
	for _, option := range options {
		option(&ins)
	}
	if ins.T != nil {
		ins.T.meta = ins.Meta
	}
	if ins.R != nil {
		ins.R.meta = ins.Meta
	}
	return ins
}

// IsZero returns true if the instrumentation is the zero value for its type.
func (i Instrumentation) IsZero() bool { return i.Meta.IsZero() }

// Child creates a child of this instrumentation with the given key. The child
// instrumentation will have a path of 'parent.key'. All traces and logs created
// by the child instrumentation will be tagged with the path.
func (i Instrumentation) Child(key string) Instrumentation {
	if i.Meta.IsZero() {
		return Instrumentation{}
	}
	if i.children == nil {
		i.children = map[string]Instrumentation{}
	}
	meta := i.Meta.child(key)
	ins := Instrumentation{
		Meta: meta,
		L:    i.L.child(meta),
		T:    i.T.child(meta),
		R:    i.R.sub(meta),
	}
	i.children[key] = ins
	return ins
}

// InstrumentationMeta is general metadata about the given instrumentation.
type InstrumentationMeta struct {
	// Key is the key used to identify this instrumentation. This key should be
	// unique within the context of its parent instrumentation (in a similar manner
	// to a file in a directory).
	Key string
	// Path is a keychain representing the parents of this instrumentation. For
	// example, an instrumentation created from 'distribution' with a key of
	// 'storage' would have a path of 'distribution.storage'.
	Path string
	// ServiceName is the name of the service that this instrumentation belongs to,
	// and is typically identified by the host-name or container-name.
	ServiceName string
}

// Report implements the ReportProvider interface.
func (im InstrumentationMeta) Report() Report {
	return Report{
		"key":          im.Key,
		"path":         im.Path,
		"service_name": im.ServiceName,
	}
}

func (im InstrumentationMeta) child(key string) InstrumentationMeta {
	return InstrumentationMeta{
		Key:         key,
		Path:        im.extendPath(key),
		ServiceName: im.ServiceName,
	}
}

func (im InstrumentationMeta) extendPath(v string) string {
	return im.Path + "." + v
}

// IsZero returns true if the instrumentation is the zero value for its type.
func (im InstrumentationMeta) IsZero() bool { return im.Key == "" }

// Option is an options pattern used with New.
type Option func(*Instrumentation)

// WithTracer configures the instrumentation to use the given Tracer.
func WithTracer(t *Tracer) Option { return func(ins *Instrumentation) { ins.T = t } }

// WithReporter configures the instrumentation to use the given Reporter.
func WithReporter(r *Reporter) Option { return func(ins *Instrumentation) { ins.R = r } }

// WithLogger configures the instrumentation to use the given Logger.
func WithLogger(l *Logger) Option { return func(ins *Instrumentation) { ins.L = l } }
