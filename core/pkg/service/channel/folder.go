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
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

// CheckFolderName returns name in NFC, the form in which a channel folder stores its
// name, or a validation error when a path cannot hold it: it holds a '.', an invisible
// character, or bytes that are not UTF-8. It returns an empty name, which the group
// reports as required.
func CheckFolderName(name string) (string, error) {
	name = validate.NormalizeName(name)
	if name == "" || validate.IsPathSegment(name) {
		return name, nil
	}
	return "", validate.PathedError(errors.Wrapf(
		validate.ErrValidation,
		"group name '%s' must hold a visible character, and no '.' or invisible "+
			"character",
		name,
	), "name")
}
