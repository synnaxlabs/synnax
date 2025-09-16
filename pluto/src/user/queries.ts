// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ontology, user } from "@synnaxlabs/client";
import { array } from "@synnaxlabs/x";

import { Flux } from "@/flux";
import { Ontology } from "@/ontology";

export type UseDeleteArgs = user.Key | user.Key[];

export interface SubStore extends Flux.Store {
  [Ontology.RELATIONSHIPS_FLUX_STORE_KEY]: Ontology.RelationshipFluxStore;
  [Ontology.RESOURCES_FLUX_STORE_KEY]: Ontology.ResourceFluxStore;
}

export const { useUpdate: useDelete } = Flux.createUpdate<UseDeleteArgs, SubStore>({
  name: "User",
  update: async ({ client, value, store, rollbacks }) => {
    const keys = array.toArray(value);
    const ids = keys.map((k) => user.ontologyID(k));
    const relFilter = Ontology.filterRelationshipsThatHaveIDs(ids);
    rollbacks.add(store.relationships.delete(relFilter));
    rollbacks.add(store.resources.delete(ontology.idToString(ids)));
    await client.users.delete(keys);
    return value;
  },
});
