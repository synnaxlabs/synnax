// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package project

import (
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

// ValidateOwner checks the owner of a visualization, which belongs to exactly one
// project or range. It returns a validation error on the project field when proj and
// rng are both set or both nil.
func ValidateOwner(proj *Key, rng *ranger.Key) error {
	if (proj == nil) != (rng == nil) {
		return nil
	}
	return validate.PathedError(errors.Wrap(
		validate.ErrValidation,
		"must belong to exactly one of a project and a range",
	), "project")
}
