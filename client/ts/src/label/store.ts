// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type cache } from "@/cache";
import { LABELED_BY_ONTOLOGY_RELATIONSHIP_TYPE } from "@/label/payload";
import { type Key, type Label } from "@/label/types.gen";
import { ontology } from "@/ontology";

/** Reports whether the relationship labels the given ontology ID. */
export const matchLabeledBy = (rel: ontology.Relationship, id: ontology.ID): boolean =>
  ontology.matchRelationship(rel, {
    from: id,
    type: LABELED_BY_ONTOLOGY_RELATIONSHIP_TYPE,
  });

/** Returns the cached labels attached to the given ontology ID. */
export const cachedLabelsOf = (
  relationships: cache.Table<string, ontology.Relationship>,
  labels: cache.Table<Key, Label>,
  id: ontology.ID,
): Label[] =>
  labels.get(relationships.get((r) => matchLabeledBy(r, id)).map((r) => r.to.key));
