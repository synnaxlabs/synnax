// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1

import (
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
)

var _ gorp.Entry[Key] = Policy{}

// GorpKey implements the gorp.Entry interface.
func (p Policy) GorpKey() Key { return p.Key }

// SetOptions implements the gorp.Entry interface.
func (Policy) SetOptions() []any { return nil }

// ResourceID returns the unique resource identifier for the policy.
func (p Policy) ResourceID() resource.ID {
	return resource.ID{Type: resource.TypePolicy, Key: p.Key.String()}
}
