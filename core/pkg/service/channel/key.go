// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel

import (
	"strconv"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

// ParseKey attempts to parse the string representation of a Key into a Key.
func ParseKey(s string) (Key, error) {
	k, err := strconv.Atoi(s)
	if err != nil {
		return Key(0), errors.Wrapf(
			validate.ErrValidation, "%s is not a valid channel key", s,
		)
	}
	return Key(k), nil
}
