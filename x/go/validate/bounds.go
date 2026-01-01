// Copyright 2026 Synnax Labs, Inc.
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

// NewInclusiveBoundsChecker returns a function that returns an error if a value
// is not within a set of inclusive, numeric bounds.
func NewInclusiveBoundsChecker[T types.Numeric](min T, max T) func(v T) error {
	name := types.Name[T]()
	return func(v T) error {
		if v >= min && v <= max {
			return nil
		}
		return errors.Wrapf(Error, "%s must be between %v and %v", name, min, max)
	}
}
