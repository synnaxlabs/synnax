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

type keyValue struct {
	Range uuid.UUID `json:"range" msgpack:"range"`
	Key   []byte    `json:"key" msgpack:"key"`
	Value []byte    `json:"value" msgpack:"value"`
}

var _ gorp.Entry[[]byte] = keyValue{}

// GorpKey implements gorp.Entry.
func (k keyValue) GorpKey() []byte { return append(k.Range[:], k.Key...) }

// SetOptions implements gorp.Entry.
func (k keyValue) SetOptions() []interface{} { return nil }
