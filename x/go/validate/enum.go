// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package validate

import (
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/types"
)

// NewEnumBoundsChecker returns a function
func NewEnumBoundsChecker[T types.Numeric](min T, max T) func(v T) error {
	name := types.Name[T]()
	return func(v T) error {
		if v >= min && v <= max {
			return nil
		}
		return errors.Wrapf(Error, "%s must be between %v and %v", name, min, max)
	}
}
