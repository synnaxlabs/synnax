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

var _ gorp.Entry[Key] = User{}

// ResourceID returns a unique identifier for the user for use within a resource
// ontology.
func (u User) ResourceID() resource.ID {
	return resource.ID{Type: resource.TypeUser, Key: u.Key.String()}
}

// GorpKey implements gorp.Entry.
func (u User) GorpKey() Key { return u.Key }

// SetOptions implements gorp.Entry.
func (User) SetOptions() []any { return nil }
