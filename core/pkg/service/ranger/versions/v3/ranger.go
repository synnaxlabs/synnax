// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v3

import (
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
)

var _ gorp.Entry[Key] = Range{}

// GorpKey implements gorp.Entry.
func (r Range) GorpKey() Key { return r.Key }

// ResourceID returns the ID of the range in the ontology.
func (r Range) ResourceID() resource.ID {
	return resource.ID{Type: resource.TypeRange, Key: r.Key.String()}
}
