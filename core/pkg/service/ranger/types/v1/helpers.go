// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1

import (
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/gorp"
)

var _ gorp.Entry[Key] = Range{}

// GorpKey implements gorp.Entry.
func (r Range) GorpKey() Key { return r.Key }

// SetOptions implements gorp.Entry.
func (r Range) SetOptions() []any { return nil }

// OntologyID returns the semantic ID for this range to look it up from within the
func (r Range) OntologyID() ontology.ID {
	return ontology.ID{Type: ontology.ResourceTypeRange, Key: r.Key.String()}
}
