// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package constraint

import (
	"context"

	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/x/gorp"
)

func resolveRelationship(
	ctx context.Context,
	params EnforceParams,
	from ontology.ID,
	relType ontology.RelationshipType,
) []ontology.ID {
	var relationships []ontology.Relationship
	if err := gorp.NewRetrieve[[]byte, ontology.Relationship]().
		Where(func(_ gorp.Context, rel *ontology.Relationship) (bool, error) {
			return rel.From == from && rel.Type == relType, nil
		}).
		Entries(&relationships).
		Exec(ctx, params.Tx); err != nil {
		return nil
	}
	relatedIDs := make([]ontology.ID, 0, len(relationships))
	for _, rel := range relationships {
		relatedIDs = append(relatedIDs, rel.To)
	}
	return relatedIDs
}

func containsID(ids []ontology.ID, target ontology.ID) bool {
	for _, id := range ids {
		if id == target {
			return true
		}
	}
	return false
}
