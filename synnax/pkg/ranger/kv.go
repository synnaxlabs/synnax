// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ranger

import (
	"github.com/google/uuid"
	"github.com/synnaxlabs/x/gorp"
)

type KVPair struct {
	Range uuid.UUID `json:"range" msgpack:"range"`
	Key   string    `json:"key" msgpack:"key"`
	Value string    `json:"value" msgpack:"value"`
}

var _ gorp.Entry[string] = KVPair{}

// GorpKey implements gorp.Entry.
func (k KVPair) GorpKey() string { return k.Range.String() + "<--->" + k.Key }

// SetOptions implements gorp.Entry.
func (k KVPair) SetOptions() []interface{} { return nil }
