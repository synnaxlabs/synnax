// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package framer

import (
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
)

const OntologyType ontology.Type = "framer"

// OntologyID returns a unique identifier for a Channel for use within a resource
// ontology.
func OntologyID(k channel.Key) ontology.ID {
	return ontology.ID{Type: OntologyType, Key: k.String()}
}

// OntologyIDs returns the ontology.ID for each key.
func OntologyIDs(ks channel.Keys) []ontology.ID {
	return lo.Map(ks, func(key channel.Key, _ int) ontology.ID { return OntologyID(key) })
}
