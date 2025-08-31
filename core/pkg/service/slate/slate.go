// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package slate

import (
	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/service/slate/spec"
	"github.com/synnaxlabs/x/gorp"
)

type Slate struct {
	Key   uuid.UUID  `json:"key" msgpack:"key"`
	Graph spec.Graph `json:"graph" msgpack:"graph"`
}

var _ gorp.Entry[uuid.UUID] = Slate{}

// GorpKey implements gorp.Entry.
func (s Slate) GorpKey() uuid.UUID { return s.Key }

// SetOptions implements gorp.Entry.
func (s Slate) SetOptions() []any { return nil }
