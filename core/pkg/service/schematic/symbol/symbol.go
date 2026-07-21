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
	"github.com/google/uuid"
	"github.com/synnaxlabs/x/gorp"
)

// Key is a unique identifier for a symbol, represented as a UUID.
type Key = uuid.UUID

// Symbol represents a schematic symbol specification with its visual states and regions.
type Symbol struct {
	// Data is JSON-encoded data containing SVG, states and regions for the symbol.
	Data map[string]any `json:"data" msgpack:"data"`
	// Name is a human-readable name for the symbol.
	Name string `json:"name" msgpack:"name"`
	// Key is a unique identifier for the symbol.
	Key Key `json:"key" msgpack:"key"`
}

var _ gorp.Entry[Key] = Symbol{}

// GorpKey implements gorp.Entry.
func (s Symbol) GorpKey() Key { return s.Key }

// SetOptions implements gorp.Entry.
func (Symbol) SetOptions() []any { return nil }
