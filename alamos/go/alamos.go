// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package alamos

import "fmt"

type sub[L any] interface {
	sub(meta InstrumentationMeta) L
}

type Instrumentation struct {
	meta InstrumentationMeta
	// L is the Logger used by this instrumentation.
	L *Logger
	// T is the Tracer used by this instrumentation.
	T *Tracer
	// R is the Reporter used by this instrumentation.
	R *Reporter
}

func (i Instrumentation) IsZero() bool { return i.meta.IsZero() }

func (i Instrumentation) Sub(key string) Instrumentation {
	meta := i.meta.sub(key)
	return Instrumentation{
		L: i.L.sub(meta),
		T: i.T.sub(meta),
		R: i.R.sub(meta),
	}
}

type InstrumentationMeta struct {
	// Key is the key used to identify this instrumentation.
	Key string
	// ServiceName is the name of the service.
	ServiceName string
	// Filter is the filter used by this instrumentation.
	Filter Filter
}

func (m InstrumentationMeta) sub(key string) InstrumentationMeta {
	return InstrumentationMeta{
		Key:         fmt.Sprintf("%s.%s", m.Key, key),
		ServiceName: m.ServiceName,
		Filter:      m.Filter,
	}
}

func (m InstrumentationMeta) IsZero() bool {
	return m.Key != ""
}

type Option func(*Instrumentation)

func WithTracer(tracer *Tracer) Option {
	return func(ins *Instrumentation) {
		ins.T = tracer
	}
}

func WithLogger(logger *Logger) Option {
	return func(ins *Instrumentation) {
		ins.L = logger
	}
}

func WithReports(reports *Reporter) Option {
	return func(ins *Instrumentation) {
		ins.R = reports
	}
}

func WithFilter(filter Filter) Option {
	return func(ins *Instrumentation) {
		ins.meta.Filter = filter
	}
}

func WithServiceName(serviceName string) Option {
	return func(ins *Instrumentation) {
		ins.meta.ServiceName = serviceName
	}
}

func New(key string, options ...Option) Instrumentation {
	ins := Instrumentation{meta: InstrumentationMeta{Key: key}}
	for _, option := range options {
		option(&ins)
	}
	return ins
}
