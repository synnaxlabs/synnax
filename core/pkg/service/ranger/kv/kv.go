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
	"github.com/synnaxlabs/x/gorp"
)

const keySeparator = "<--->"

var _ gorp.Entry[string] = Pair{}

// GorpKey implements gorp.Entry.
func (k Pair) GorpKey() string { return k.Range.String() + keySeparator + k.Key }

// SetOptions implements gorp.Entry.
func (k Pair) SetOptions() []any { return nil }

func (k Pair) CustomTypeName() string { return "KVPair" }
