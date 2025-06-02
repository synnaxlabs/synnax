// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package annotation

import (
	"github.com/google/uuid"
	"github.com/synnaxlabs/x/gorp"
)

type Annotation struct {
	Key     uuid.UUID `json:"key" msgpack:"key"`
	Variant string    `json:"variant" msgpack:"variant"`
	Message string    `json:"message" msgpack:"message"`
	Notify  bool      `json:"notify" msgpack:"notify"`
}

var _ gorp.Entry[uuid.UUID] = Annotation{}

// GorpKey implements gorp.Entry.
func (a Annotation) GorpKey() uuid.UUID { return a.Key }

// SetOptions implements gorp.Entry.
func (a Annotation) SetOptions() []any { return nil }
