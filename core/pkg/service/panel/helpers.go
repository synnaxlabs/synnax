// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package panel

import (
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/gorp"
)

var _ gorp.Entry[Key] = Panel{}

// GorpKey implements gorp.Entry.
func (p Panel) GorpKey() Key { return p.Key }

// SetOptions implements gorp.Entry.
func (p Panel) SetOptions() []any { return nil }

// OntologyID returns the ontology.ID of the resource.
func (p Panel) OntologyID() ontology.ID { return OntologyID(p.Key) }
