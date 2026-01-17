// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package zyn

import (
	"reflect"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/types"
	"github.com/synnaxlabs/x/validate"
)

var ErrInvalidDestinationType = errors.Wrap(
	validate.ErrValidation,
	"invalid destination type",
)

func NewInvalidDestinationTypeError(expected string, received reflect.Value) error {
	return errors.Wrapf(
		ErrInvalidDestinationType,
		"must be a non-nil pointer to a %s, but received %s",
		expected,
		types.ValueName(received),
	)
}

func isNumericType(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	}
	return false
}

func validateNilData(destVal reflect.Value, data any, base baseZ) (bool, error) {
	if data != nil {
		return true, nil
	}
	if base.optional {
		if destVal.Elem().Kind() == reflect.Pointer {
			destVal.Elem().Set(reflect.Zero(destVal.Elem().Type()))
		}
		return false, nil
	}
	return false, errors.WithStack(validate.ErrRequired)
}
