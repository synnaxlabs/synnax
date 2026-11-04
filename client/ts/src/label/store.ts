// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { LABELED_BY_ONTOLOGY_RELATIONSHIP_TYPE } from "@/label/payload";
import { type Key, type Label } from "@/label/types.gen";
import { ontology } from "@/ontology";
import { type query } from "@/query";
import { type resource } from "@/resource";

/** Reports whether the relationship labels the given resource ID. */
export const matchLabeledBy = (rel: ontology.Relationship, id: resource.ID): boolean =>
  ontology.matchRelationship(rel, {
    from: id,
    type: LABELED_BY_ONTOLOGY_RELATIONSHIP_TYPE,
  });

/** Returns the cached labels attached to the given resource ID. */
export const cachedLabelsOf = (
  cache: ontology.Cache,
  labels: query.Table<Key, Label>,
  id: resource.ID,
): Label[] =>
  labels.get(
    cache
      .relationshipsFrom(id)
      .filter((r) => r.type === LABELED_BY_ONTOLOGY_RELATIONSHIP_TYPE)
      .map((r) => r.to.key),
  );
