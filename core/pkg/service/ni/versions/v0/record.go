// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0

// CustomTypeName gives AnalogReadConfig records a table prefix that cannot collide with
// the
// identically named config types of other integrations.
func (AnalogReadConfig) CustomTypeName() string { return "ni_analog_read_config" }

// CustomTypeName gives AnalogWriteConfig records a table prefix that cannot collide
// with the
// identically named config types of other integrations.
func (AnalogWriteConfig) CustomTypeName() string { return "ni_analog_write_config" }

// CustomTypeName gives CounterReadConfig records a table prefix that cannot collide
// with the
// identically named config types of other integrations.
func (CounterReadConfig) CustomTypeName() string { return "ni_counter_read_config" }

// CustomTypeName gives DigitalReadConfig records a table prefix that cannot collide
// with the
// identically named config types of other integrations.
func (DigitalReadConfig) CustomTypeName() string { return "ni_digital_read_config" }

// CustomTypeName gives DigitalWriteConfig records a table prefix that cannot collide
// with the
// identically named config types of other integrations.
func (DigitalWriteConfig) CustomTypeName() string { return "ni_digital_write_config" }

// CustomTypeName gives ScannerConfig records a table prefix that cannot collide with
// the
// identically named config types of other integrations.
func (ScannerConfig) CustomTypeName() string { return "ni_scanner_config" }
