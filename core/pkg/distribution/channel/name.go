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
	"strings"
	"unicode"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
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

// TransformName converts an invalid channel name into a valid one for migration purposes.
// It replaces invalid characters with underscores and handles reserved keywords by
// appending a suffix.
func TransformName(name string) string {
	if name == "" {
		return "channel"
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return "channel"
	}

	var (
		result           strings.Builder
		hasLetterOrDigit bool
	)
	for i, r := range name {
		switch {
		case unicode.IsLetter(r):
			result.WriteRune(r)
			hasLetterOrDigit = true
		case unicode.IsDigit(r):
			if i == 0 {
				// If starts with digit, prepend underscore
				result.WriteRune('_')
			}
			result.WriteRune(r)
			hasLetterOrDigit = true
		default:
			result.WriteRune('_')
		}
	}
	if !hasLetterOrDigit {
		return "channel"
	}

	return result.String()
}

// NewUniqueName generates a unique channel name by appending a numeric suffix if the
// name already exists in the provided name set.
func NewUniqueName(baseName string, existingNames set.Set[string]) string {
	name := baseName
	counter := 1
	for existingNames.Contains(name) {
		name = fmt.Sprintf("%s_%d", baseName, counter)
		counter++
	}
	return name
}

// NewRandomName generates a random channel name that should be unique.
func NewRandomName() string {
	randomSuffix := rand.Intn(999999999)
	return fmt.Sprintf("test_ch_%09d", randomSuffix)
}
