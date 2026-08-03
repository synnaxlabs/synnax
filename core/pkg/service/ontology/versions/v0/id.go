// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0

import (
	"strings"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

// String returns a string representation of the ID in the format "Type:Key".
func (i ID) String() string { return string(i.Type) + ":" + i.Key }

// IsZero returns true if the ID is the zero value (both Key and Type are empty).
func (i ID) IsZero() bool { return i.Key == "" && i.Type == "" }

// IsType returns true if the ID represents a type identifier (has a Type but no Key).
// Type IDs are used to identify resource types rather than specific resource instances.
func (i ID) IsType() bool { return i.Type != "" && i.Key == "" }

// ParseID parses the given key into an ID.
func ParseID(key string) (ID, error) {
	// We explicitly allow IDs that have multiple colons, i.e. "foo:bar:baz" will be
	// parsed as ID{Type: "foo", Key: "bar:baz"}.
	split := strings.SplitN(key, ":", 2)
	if len(split) != 2 {
		return ID{}, errors.Wrapf(
			validate.ErrValidation,
			"[ontology.resource] - failed to parse id: %s",
			key,
		)
	}
	if split[0] == "" {
		return ID{}, errors.Wrapf(
			validate.ErrValidation,
			"[ontology.resource] - failed to parse id: %s (empty type)",
			key,
		)
	}
	return ID{Type: ResourceType(split[0]), Key: split[1]}, nil
}
