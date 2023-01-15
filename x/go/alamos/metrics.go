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
	"github.com/synnaxlabs/x/types"
	"sync"
)

// Metric is a container for storing measured values.
type Metric[T any] interface {
	// Record records a value.
	Record(T)
	// Values returns the recorded values as a slice.
	Values() []T
	// Count returns the number of times Record was called.
	Count() int
	baseMetric
}

// |||||| BASE ||||||

type baseMetric interface {
	// Key returns the key of the metric.
	Key() string
	// Report returns a report of the recorded values.
	Report() map[string]interface{}
}

func newBaseMetric(key string) baseMetric { return defaultBaseMetric{k: key} }

type defaultBaseMetric struct {
	k string
}

func (b defaultBaseMetric) Key() string { return b.k }

func (b defaultBaseMetric) Report() map[string]interface{} { return map[string]interface{}{"key": b.k} }

// |||||| GAUGE ||||||

type gauge[T types.Numeric] struct {
	baseMetric
	mu    sync.Mutex
	count int
	value T
	min   T
	max   T
}

// NewGauge creates a new gauge metric. A gauge records the sum of all recorded values as well as
// the number of times Frame was called.
func NewGauge[T types.Numeric](exp Experiment, level Level, key string) Metric[T] {
	if m := emptyMetric[T](exp, level, key); m != nil {
		return m
	}
	m := &gauge[T]{baseMetric: newBaseMetric(key)}
	exp.addMetric(m)
	return m
}

// Count implements Metric.
func (g *gauge[T]) Count() int { return g.count }

// Values implements Metric.
//
//	The first value returned represents the average value.
//	The second value represents the sum of all recorded value
//	The third value represents the number of times Record was called.
func (g *gauge[T]) Values() []T { return []T{g.average(), g.value, T(g.count)} }

func (g *gauge[T]) average() T {
	if g.count == 0 {
		return 0
	}
	return g.value / T(g.count)
}

func (g *gauge[T]) Record(v T) {
	g.mu.Lock()
	defer g.mu.Unlock()
	// Check for min/max
	if g.count == 0 {
		g.min = v
		g.max = v
	} else {
		if v < g.min {
			g.min = v
		}
		if v > g.max {
			g.max = v
		}
	}
	g.value += v
	g.count++
}

func (g *gauge[T]) Report() map[string]interface{} {
	return map[string]interface{}{
		"key":   g.Key(),
		"count": g.count,
		"value": g.value,
		"avg":   g.average(),
		"min":   g.min,
		"max":   g.max,
	}
}

// |||||| SERIES ||||||

type series[T any] struct {
	baseMetric
	values []T
}

func (s *series[T]) Value() interface{} { return s.values }

func (s *series[T]) Values() []T { return s.values }

func (s *series[T]) Record(v T) { s.values = append(s.values, v) }

func (s *series[T]) Count() int { return len(s.values) }

func (s *series[T]) Report() map[string]interface{} {
	base := s.baseMetric.Report()
	base["values"] = s.values
	return base
}

// NewSeries creates a new series metric. A series stores all recorded values in a slice.
func NewSeries[T any](exp Experiment, level Level, key string) Metric[T] {
	if m := emptyMetric[T](exp, level, key); m != nil {
		return m
	}
	m := &series[T]{baseMetric: newBaseMetric(key)}
	exp.addMetric(m)
	return m
}

// |||||| EMPTY ||||||

type empty[T any] struct{}

func (e empty[T]) Values() []T { return nil }

func (e empty[T]) Record(T) {}

func (e empty[T]) Count() int { return 0 }

func (e empty[T]) Key() string { return "" }

func (e empty[T]) Report() map[string]interface{} { return nil }

func emptyMetric[T any](exp Experiment, level Level, key string) Metric[T] {
	if exp != nil && exp.filterTest(level) {
		return nil
	}
	m := empty[T]{}
	return m
}
