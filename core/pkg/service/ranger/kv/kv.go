// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package kv implements a service for managing key-value pairs on ranges.
package kv

import (
	"github.com/google/uuid"
	"github.com/synnaxlabs/x/gorp"
)

const keySeparator = "<--->"

// Pair is a unique, key-value pair tied directly to a specific range.
type Pair struct {
	// Key is the key of the key-value pair.
	Key string `json:"key" msgpack:"key"`
	// Value is the value of the key-value pair.
	Value string `json:"value" msgpack:"value"`
	// Range is the range that the key-value pair belongs to.
	Range uuid.UUID `json:"range" msgpack:"range"`
}

var _ gorp.Entry[string] = Pair{}

// GorpKey implements gorp.Entry.
func (k Pair) GorpKey() string { return k.Range.String() + keySeparator + k.Key }

// SetOptions implements gorp.Entry.
func (k Pair) SetOptions() []any { return nil }
