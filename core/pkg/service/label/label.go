// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package label

import (
	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/gorp"
)

// Label represents a label that can be applied to arbitrary resources within the
// synnax ontology. A label has a unique UUID key, a name, and a color.
type Label struct {
	// Key is a unique identifier for the label.
	Key uuid.UUID `json:"key" msgpack:"key"`
	// Name is the human-readable name of the label.
	Name string `json:"name" msgpack:"name"`
	// Color is the color associated with the label.
	Color color.Color `json:"color" msgpack:"color"`
}

var _ gorp.Entry[uuid.UUID] = Label{}

// GorpKey implements gorp.Entry.
func (l Label) GorpKey() uuid.UUID { return l.Key }

// SetOptions implements gorp.Entry.
func (l Label) SetOptions() []any { return nil }

// OntologyID returns the unique ontology identifier for the label.
func (l Label) OntologyID() ontology.ID { return OntologyID(l.Key) }
