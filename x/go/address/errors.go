// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package address

import "github.com/synnaxlabs/x/errors"

// ErrNotFound is a general error used when an address is not found.
var ErrNotFound = errors.New("[address] - not found")

// NewErrTargetNotFound returns an error indicating that the target at the given
// address could not be located.
func NewErrTargetNotFound(target Address) error {
	return errors.Wrapf(ErrNotFound, "target %s not found", target)
}
