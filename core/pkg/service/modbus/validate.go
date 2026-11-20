// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package modbus

import (
	"strconv"

	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// registerDataTypes are the data types the Driver reads from and writes to registers.
var registerDataTypes = set.New(
	telem.Int8T,
	telem.Uint8T,
	telem.Int16T,
	telem.Uint16T,
	telem.Int32T,
	telem.Uint32T,
	telem.Int64T,
	telem.Uint64T,
	telem.Float32T,
	telem.Float64T,
)

func validateRegisterValue(r RegisterValue, read bool) error {
	v := validate.New("modbus.RegisterValue")
	msg := "must be a numeric data type, not " + string(r.DataType)
	if read && r.DataType == telem.StringT {
		msg += "; read a string with a register string channel"
	}
	v.Ternary("data_type", !registerDataTypes.Contains(r.DataType), msg)
	return v.Error()
}

func validateRegisterValueAt(v *validate.Validator, i int, r RegisterValue, read bool) {
	v.Exec(func() error {
		err := validateRegisterValue(r, read)
		return validate.PathedError(err, "channels", strconv.Itoa(i))
	})
}

func validateReadConfig(c *ReadConfig) error {
	v := validate.New("modbus.ReadConfig")
	v.Exec(c.Validate)
	v.GreaterThan("stream_rate", c.StreamRate, 0)
	for i, ch := range c.Channels {
		switch variant := ch.Variant.(type) {
		case HoldingRegisterReadChannel:
			validateRegisterValueAt(v, i, variant.RegisterValue, true)
		case InputRegisterReadChannel:
			validateRegisterValueAt(v, i, variant.RegisterValue, true)
		}
	}
	return v.Error()
}

func validateWriteConfig(c *WriteConfig) error {
	v := validate.New("modbus.WriteConfig")
	v.Exec(c.Validate)
	for i, ch := range c.Channels {
		if variant, ok := ch.Variant.(HoldingRegisterWriteChannel); ok {
			validateRegisterValueAt(v, i, variant.RegisterValue, false)
		}
	}
	return v.Error()
}
