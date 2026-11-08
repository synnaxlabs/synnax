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
	"strings"

	"github.com/synnaxlabs/gorp"
)

const keySeparator = "<--->"

var _ gorp.Entry[string] = Pair{}

// GorpKey implements gorp.Entry.
func (p Pair) GorpKey() string { return p.Range.String() + keySeparator + p.Key }

// GorpKeyRange returns the range part of key, a key that GorpKey returns.
func (Pair) GorpKeyRange(key string) string {
	rng, _, _ := strings.Cut(key, keySeparator)
	return rng
}

// CustomTypeName supplies a custom type name for the Pair type.
func (Pair) CustomTypeName() string { return "KVPair" }
