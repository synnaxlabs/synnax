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
	"github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/telem"
)

type Key = uuid.UUID

type Range struct {
	Key       Key             `json:"key" msgpack:"key"`
	Name      string          `json:"name" msgpack:"name"`
	TimeRange telem.TimeRange `json:"time_range" msgpack:"time_range"`
	Color     color.Color     `json:"color" msgpack:"color"`
}

var _ gorp.Entry[Key] = Range{}

func (r Range) GorpKey() Key      { return r.Key }
func (r Range) SetOptions() []any { return nil }

func OntologyID(k uuid.UUID) ontology.ID {
	return ontology.ID{Type: ontology.ResourceTypeRange, Key: k.String()}
}
