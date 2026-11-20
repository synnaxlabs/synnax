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

var _ gorp.Entry[Key] = Named{}

// GorpKey implements gorp.Entry.
func (n Named) GorpKey() Key { return n.Key }

// ResourceID returns the resource ID of the named color.
func (n Named) ResourceID() resource.ID {
	return resource.ID{Type: resource.TypeColor, Key: n.Key.String()}
}

// Inline returns the fixed color that takes the place of a reference to n when n is
// deleted. The fixed color keeps the light value of n.
func (n Named) Inline() FixedRef { return FixedRef{Value: n.Light} }
