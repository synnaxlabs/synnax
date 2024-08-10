/*
 * Copyright 2023 Synnax Labs, Inc.
 *
 * Use of this software is governed by the Business Source License included in the file
 * licenses/BSL.txt.
 *
 * As of the Change Date specified in that file, in accordance with the Business Source
 * License, use of this software will be governed by the Apache License, Version 2.0,
 * included in the file licenses/APL.txt.
 */

package label

import (
	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/gorp"
)

type Label struct {
	Key   uuid.UUID   `json:"key" msgpack:"key"`
	Name  string      `json:"name" msgpack:"name"`
	Color color.Color `json:"color" msgpack:"color"`
}

var _ gorp.Entry[uuid.UUID] = Label{}

// GorpKey implements gorp.Entry.
func (l Label) GorpKey() uuid.UUID { return l.Key }

// SetOptions implements gorp.Entry.
func (l Label) SetOptions() []interface{} { return nil }

func (l Label) OntologyID() ontology.ID { return OntologyID(l.Key) }
