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

var _ gorp.Entry[Key] = Directory{}

// GorpKey implements gorp.Entry.
func (d Directory) GorpKey() Key { return d.Key }

// SetOptions implements gorp.Entry.
func (Directory) SetOptions() []any { return nil }

// ResourceID returns the ID of the directory in the ontology.
func (d Directory) ResourceID() resource.ID {
	return resource.ID{Type: resource.TypeArcDirectory, Key: d.Key.String()}
}
