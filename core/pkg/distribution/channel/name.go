// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel

import (
	"fmt"
	"math/rand"
	"regexp"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

var ErrInvalidName = errors.Wrap(validate.Error, "invalid channel name")

// validNamePattern matches valid channel names: letters, digits, and underscores only
var validNamePattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// ValidateName validates a channel name according to the following rules:
// 1. Only letters, digits, and underscores are allowed
// 2. Cannot start with a digit
// 3. Cannot be a reserved Arc keyword
// 4. Cannot be empty
func ValidateName(name string) error {
	if name == "" {
		return errors.Wrap(ErrInvalidName, "name cannot be empty")
	}
	if !validNamePattern.MatchString(name) {
		return errors.Wrapf(
			ErrInvalidName,
			"channel name '%s' contains invalid characters. Only letters, digits, and underscores are allowed, and it cannot start with a digit",
			name,
		)
	}
	return nil
}

// NewRandomName generates a random channel name that should be unique.
func NewRandomName() string {
	randomSuffix := rand.Intn(999999999)
	return fmt.Sprintf("test_ch_%09d", randomSuffix)
}
