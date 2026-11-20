// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v5

import (
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
)

var _ gorp.Entry[Key] = Table{}

// GorpKey implements gorp.Entry.
func (t Table) GorpKey() Key { return t.Key }

// ResourceID returns the unique resource identifier for the table.
func (t Table) ResourceID() resource.ID {
	return resource.ID{Type: resource.TypeTable, Key: t.Key.String()}
}
