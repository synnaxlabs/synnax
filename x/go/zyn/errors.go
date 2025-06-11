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
	"github.com/synnaxlabs/x/validate"
)

func InvalidDestError(t string) error {
	return errors.Wrapf(validate.Error, "dest must be a non-nil pointer to a %s", t)
}

func checkDestVal(dest reflect.Value, t string) error {
	if dest.Kind() != reflect.Ptr || dest.IsNil() {
		return InvalidDestError(t)
	}
	return nil

}
