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
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/gorp"
)

var _ gorp.Entry[Key] = Label{}

// GorpKey implements gorp.Entry.
func (l Label) GorpKey() Key { return l.Key }

// SetOptions implements gorp.Entry.
func (Label) SetOptions() []any { return nil }

// OntologyID returns the unique ontology identifier for the label.
func (l Label) OntologyID() ontology.ID {
	return ontology.ID{Type: ontology.ResourceTypeLabel, Key: l.Key.String()}
}
