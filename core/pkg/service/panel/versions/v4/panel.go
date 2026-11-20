// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v4

import (
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
)

// Version is the portable schema version of the Panel shape this package defines. A
// panel has no single-resource file form, so it is hand-written rather than generated
// from a @go imex marker.
const Version imex.Version = 4

var _ gorp.Entry[Key] = Panel{}

// GorpKey implements gorp.Entry.
func (p Panel) GorpKey() Key { return p.Key }

// ResourceID returns the resource.ID of the resource.
func (p Panel) ResourceID() resource.ID {
	return resource.ID{Type: resource.TypePanel, Key: p.Key.String()}
}
