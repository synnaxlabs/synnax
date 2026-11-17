// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package directory

import (
	"uuid"

	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

// ResourceID returns the resource ID of the directory with key.
func ResourceID(key Key) resource.ID {
	return resource.ID{Type: resource.TypeArcDirectory, Key: key.String()}
}

// KeyFromResourceID returns the directory key in id, or a validation error.
func KeyFromResourceID(id resource.ID) (Key, error) {
	if id.Type != resource.TypeArcDirectory {
		return uuid.Nil(), errors.Wrapf(
			validate.ErrValidation, "must be an Arc directory, got %q", id.Type,
		)
	}
	key, err := uuid.Parse(id.Key)
	if err != nil {
		return uuid.Nil(), errors.Wrapf(
			validate.ErrValidation, "invalid Arc directory key %q: %v", id.Key, err,
		)
	}
	return key, nil
}
