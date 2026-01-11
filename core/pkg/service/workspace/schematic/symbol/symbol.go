// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package symbol

import (
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/uuid"
)

// Symbol represents a schematic symbol specification with its visual states and regions.
type Symbol struct {
	// Key is a unique identifier for the symbol.
	Key uuid.UUID `json:"key" msgpack:"key"`
	// Name is a human-readable name for the symbol.
	Name string `json:"name" msgpack:"name"`
	// Data is JSON-encoded data containing SVG, states and regions for the symbol.
	Data map[string]any `json:"data" msgpack:"data"`
}

var _ gorp.Entry[uuid.UUID] = Symbol{}

// GorpKey implements gorp.Entry.
func (s Symbol) GorpKey() uuid.UUID { return s.Key }

// SetOptions implements gorp.Entry.
func (s Symbol) SetOptions() []any { return nil }
