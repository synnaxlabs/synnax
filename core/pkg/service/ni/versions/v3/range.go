// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v3

import (
	"context"
	"reflect"
	"uuid"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp/migrate"
	"go.uber.org/zap"
)

const rangeMigrationKey = "v63_order_ranges"

// AnalogReadRangeMigration swaps each reversed channel range of the stored analog read
// configs, and gives each range whose minimum equals its maximum the default range of
// the channel's type. Earlier Cores stored such ranges.
var AnalogReadRangeMigration = migrate.NewInstrumentedEntry(
	rangeMigrationKey,
	func(
		_ context.Context,
		c AnalogReadConfig,
		ins alamos.Instrumentation,
	) (AnalogReadConfig, error) {
		logOrdered(ins, c.GorpKey(), orderAIRanges(c.Channels))
		return c, nil
	},
)

// AnalogWriteRangeMigration is AnalogReadRangeMigration for analog write configs.
var AnalogWriteRangeMigration = migrate.NewInstrumentedEntry(
	rangeMigrationKey,
	func(
		_ context.Context,
		c AnalogWriteConfig,
		ins alamos.Instrumentation,
	) (AnalogWriteConfig, error) {
		logOrdered(ins, c.GorpKey(), orderAORanges(c.Channels))
		return c, nil
	},
)

// CounterReadRangeMigration is AnalogReadRangeMigration for counter read configs.
var CounterReadRangeMigration = migrate.NewInstrumentedEntry(
	rangeMigrationKey,
	func(
		_ context.Context,
		c CounterReadConfig,
		ins alamos.Instrumentation,
	) (CounterReadConfig, error) {
		logOrdered(ins, c.GorpKey(), orderCIRanges(c.Channels))
		return c, nil
	},
)

// orderAIRanges orders the channel ranges as AnalogReadRangeMigration does, and returns
// the indices of the channels it changes.
func orderAIRanges(channels []AIChannel) []int {
	var changed []int
	for i := range channels {
		var ok bool
		if channels[i].Variant, ok = orderRange(channels[i].Variant); ok {
			changed = append(changed, i)
		}
	}
	return changed
}

// orderAORanges is orderAIRanges for analog output channels.
func orderAORanges(channels []AOChannel) []int {
	var changed []int
	for i := range channels {
		var ok bool
		if channels[i].Variant, ok = orderRange(channels[i].Variant); ok {
			changed = append(changed, i)
		}
	}
	return changed
}

// orderCIRanges is orderAIRanges for counter input channels.
func orderCIRanges(channels []CIChannel) []int {
	var changed []int
	for i := range channels {
		var ok bool
		if channels[i].Variant, ok = orderRange(channels[i].Variant); ok {
			changed = append(changed, i)
		}
	}
	return changed
}

// orderRange returns variant with its range in order, and whether it changed. A variant
// without a range returns unchanged.
func orderRange[V any](variant V) (V, bool) {
	val := reflect.ValueOf(variant)
	if !val.IsValid() {
		return variant, false
	}
	ptr := reflect.New(val.Type())
	ptr.Elem().Set(val)
	field := ptr.Elem().FieldByName("MinMaxVal")
	if !field.IsValid() || field.Type() != reflect.TypeFor[MinMaxVal]() {
		return variant, false
	}
	r := as[*MinMaxVal](field.Addr().Interface())
	switch {
	case r.MinVal > r.MaxVal:
		r.MinVal, r.MaxVal = r.MaxVal, r.MinVal
	case r.MinVal == r.MaxVal:
		*r = MinMaxVal{}
		as[interface{ DefaultGroups() }](ptr.Interface()).DefaultGroups()
	default:
		return variant, false
	}
	return as[V](ptr.Elem().Interface()), true
}

func logOrdered(ins alamos.Instrumentation, config uuid.UUID, channels []int) {
	for _, i := range channels {
		ins.L.Warn(
			"ordered the range of an NI channel",
			zap.Stringer("config", config),
			zap.Int("channel", i),
		)
	}
}
