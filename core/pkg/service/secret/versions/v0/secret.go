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
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
)

var _ gorp.Entry[Key] = Secret{}

// GorpKey returns the secret's storage key.
func (s Secret) GorpKey() Key { return s.Base().Key }

// SetOptions returns no additional storage options.
func (Secret) SetOptions() []any { return nil }

// ResourceID returns the secret's resource identifier.
func (s Secret) ResourceID() resource.ID {
	return resource.ID{Type: resource.TypeSecret, Key: s.Base().Key.String()}
}
