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
	"github.com/synnaxlabs/x/telem"
)

type Range struct {
	Key       uuid.UUID       `json:"key" msgpack:"key"`
	Name      string          `json:"name" msgpack:"name"`
	TimeRange telem.TimeRange `json:"time_range" msgpack:"time_range"`
}

var _ gorp.Entry[uuid.UUID] = Range{}

// GorpKey implements gorp.Entry.
func (r Range) GorpKey() uuid.UUID { return r.Key }

// SetOptions implements gorp.Entry.
func (r Range) SetOptions() []interface{} { return nil }
