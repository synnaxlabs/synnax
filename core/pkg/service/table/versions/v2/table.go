// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v2

import (
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/gorp"
)

var _ gorp.Entry[Key] = Table{}

// GorpKey implements gorp.Entry.
func (t Table) GorpKey() Key { return t.Key }

// SetOptions implements gorp.Entry.
func (Table) SetOptions() []any { return nil }

// OntologyID returns the unique ontology identifier for the table.
func (t Table) OntologyID() ontology.ID {
	return ontology.ID{Type: ontology.ResourceTypeTable, Key: t.Key.String()}
}
