// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package view

import (
	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/x/gorp"
)

type View struct {
	Key   uuid.UUID      `json:"key" msgpack:"key"`
	Name  string         `json:"name" msgpack:"name"`
	Type  string         `json:"type" msgpack:"type"`
	Query map[string]any `json:"query" msgpack:"query"`
}

var _ gorp.Entry[uuid.UUID] = (*View)(nil)

func (s View) OntologyID() ontology.ID { return OntologyID(s.Key) }

// GorpKey implements gorp.Entry.
func (s View) GorpKey() uuid.UUID { return s.Key }

// SetOptions implements gorp.Entry.
func (s View) SetOptions() []any { return nil }
