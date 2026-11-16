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
	"github.com/synnaxlabs/x/types"
	"github.com/synnaxlabs/x/validate"
)

// Narrow converts a protobuf integer to the narrower integer type T. It returns an
// error wrapping validate.ErrValidation when v does not fit in T.
func Narrow[T, S types.Integer](v S) (T, error) {
	n := T(v)
	if S(n) != v || (n < 0) != (v < 0) {
		return 0, errors.Wrapf(validate.ErrValidation, "%d is out of range", v)
	}
	return n, nil
}
