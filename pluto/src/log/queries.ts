// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { log, type workspace } from "@synnaxlabs/client";
import { array } from "@synnaxlabs/x";

import { Flux } from "@/flux";
import { Ontology } from "@/ontology";

export const FLUX_STORE_CONFIG: Flux.UnaryStoreConfig<FluxSubStore, log.Key, log.Log> =
  { listeners: [] };

export const FLUX_STORE_KEY = "logs";

export interface FluxStore extends Flux.UnaryStore<log.Key, log.Log> {}

export type UseDeleteArgs = log.Params;

interface FluxSubStore extends Flux.Store {
  [Ontology.RELATIONSHIPS_FLUX_STORE_KEY]: Ontology.RelationshipFluxStore;
  [Ontology.RESOURCES_FLUX_STORE_KEY]: Ontology.ResourceFluxStore;
  [FLUX_STORE_KEY]: FluxStore;
}

export const { useUpdate: useDelete } = Flux.createUpdate<UseDeleteArgs, FluxSubStore>({
  name: "Log",
  update: async ({ client, value, rollbacks, store }) => {
    const keys = array.toArray(value);
    const ids = keys.map((key) => log.ontologyID(key));
    const relFilter = Ontology.filterRelationshipsThatHaveIDs(ids);
    rollbacks.add(store.relationships.delete(relFilter));
    await client.workspaces.logs.delete(value);
    return value;
  },
});

export interface UseCreateArgs extends log.New {
  workspace: workspace.Key;
}

export interface UseCreateResult extends log.Log {
  workspace: workspace.Key;
}

export const { useUpdate: useCreate } = Flux.createUpdate<
  UseCreateArgs,
  FluxSubStore,
  UseCreateResult
>({
  name: "Log",
  update: async ({ client, value, store }) => {
    const { workspace, ...rest } = value;
    const l = await client.workspaces.logs.create(workspace, rest);
    store.logs.set(l.key, l);
    return { ...l, workspace };
  },
});
