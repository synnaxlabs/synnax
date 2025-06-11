// Copyright 2025 Synnax Labs, Inc.
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

var InvalidDestinationTypeError = errors.Wrap(validate.Error, "invalid destination type")

func NewInvalidDestinationTypeError(expected string, received reflect.Value) error {
	return errors.Wrapf(
		InvalidDestinationTypeError,
		"must be a non-nil pointer to a %s, but received %s",
		expected,
		types.ValueName(received),
	)
}

func validateDestinationValue(dest reflect.Value, expected string) error {
	if dest.Kind() != reflect.Ptr || dest.IsNil() {
		return NewInvalidDestinationTypeError(expected, dest)
	}
	return nil
}

func validateNilData(destVal reflect.Value, data any, base baseZ) (bool, error) {
	if data != nil {
		return true, nil

	}
	if base.optional {
		if destVal.Elem().Kind() == reflect.Ptr {
			destVal.Elem().Set(reflect.Zero(destVal.Elem().Type()))
		}
		return false, nil
	}
	return false, validate.FieldError{Message: "value is required but was nil"}
}
