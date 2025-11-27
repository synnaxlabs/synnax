// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package example

import (
	"github.com/google/uuid"
	"github.com/synnaxlabs/x/gorp"
)

//go:generate jerky
type Group struct {
	// Key is the unique identifier for the group.
	Key uuid.UUID `json:"key"`
	// Name is the group's display name.
	Name string `json:"name"`
	// Description is the group's description.
	Description string `json:"description"`
	// MemberCount is the number of members in the group.
	MemberCount int32 `json:"member_count"`
}

var _ gorp.Entry[uuid.UUID] = Group{}

func (g Group) GorpKey() uuid.UUID { return g.Key }

func (g Group) SetOptions() []any { return nil }
