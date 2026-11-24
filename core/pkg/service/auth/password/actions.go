// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package password

import (
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

// Handle sets the username of the password. It returns a validation error when the
// username is empty.
func (s SetUsernamePayload) Handle(p Password) (Password, error) {
	if s.Username == "" {
		return Password{}, errors.Wrap(validate.ErrValidation, "username: required")
	}
	p.Username = s.Username
	return p, nil
}
