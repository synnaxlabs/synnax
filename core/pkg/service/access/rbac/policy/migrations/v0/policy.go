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
	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/x/gorp"
)

// Policy represents the old policy format with a Subjects field that
// associated policies directly with users. Used only for reading legacy data
// during migration.
type Policy struct {
	Subjects []ontology.ID   `json:"subjects" msgpack:"subjects"`
	Objects  []ontology.ID   `json:"objects" msgpack:"objects"`
	Actions  []access.Action `json:"actions" msgpack:"actions"`
	Key      uuid.UUID       `json:"key" msgpack:"key"`
}

var _ gorp.Entry[uuid.UUID] = Policy{}

func (p Policy) GorpKey() uuid.UUID { return p.Key }
func (p Policy) SetOptions() []any  { return nil }
