// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { log } from "@synnaxlabs/client";
import { array } from "@synnaxlabs/x";

import { Flux } from "@/flux";
import { Ontology } from "@/ontology";

export type UseDeleteArgs = log.Params;

interface SubStore extends Flux.Store {
  [Ontology.RELATIONSHIPS_FLUX_STORE_KEY]: Ontology.RelationshipFluxStore;
}

export const { useUpdate: useDelete } = Flux.createUpdate<UseDeleteArgs, SubStore>({
  name: "Log",
  update: async ({ client, value, rollbacks, store }) => {
    const keys = array.toArray(value);
    const ids = keys.map((key) => log.ontologyID(key));
    const relFilter = Ontology.filterRelationshipsThatHaveResource(ids);
    rollbacks.add(store.relationships.delete(relFilter));
    await client.workspaces.logs.delete(value);
  },
});
