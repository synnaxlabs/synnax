// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package pb

import (
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

// Enum converts a protobuf string to the string enum E. It returns an error wrapping
// validate.ErrValidation when no member of E has the value s.
func Enum[E interface {
	~string
	IsValid() bool
}](s string) (E, error) {
	if e := E(s); e.IsValid() {
		return e, nil
	}
	return "", errors.Wrapf(validate.ErrValidation, "%q is not a valid key", s)
}
