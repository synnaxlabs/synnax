// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { table, type workspace } from "@synnaxlabs/client";
import { array } from "@synnaxlabs/x";

import { Flux } from "@/flux";
import { Ontology } from "@/ontology";

export const FLUX_STORE_CONFIG: Flux.UnaryStoreConfig<
  SubStore,
  table.Key,
  table.Table
> = { listeners: [] };

export const FLUX_STORE_KEY = "tables";

export interface FluxStore extends Flux.UnaryStore<table.Key, table.Table> {}

export type UseDeleteArgs = table.Params;

interface SubStore extends Flux.Store {
  [Ontology.RELATIONSHIPS_FLUX_STORE_KEY]: Ontology.RelationshipFluxStore;
  [Ontology.RESOURCES_FLUX_STORE_KEY]: Ontology.ResourceFluxStore;
  [FLUX_STORE_KEY]: FluxStore;
}

export const { useUpdate: useDelete } = Flux.createUpdate<UseDeleteArgs, SubStore>({
  name: "Table",
  update: async ({ client, value, rollbacks, store }) => {
    const keys = array.toArray(value);
    const ids = keys.map((k) => table.ontologyID(k));
    const relFilter = Ontology.filterRelationshipsThatHaveIDs(ids);
    rollbacks.add(store.relationships.delete(relFilter));
    await client.workspaces.tables.delete(value);
    return value;
  },
});

export interface UseCreateArgs extends table.New {
  workspace: workspace.Key;
}

export interface UseCreateResult extends table.Table {
  workspace: workspace.Key;
}

export const { useUpdate: useCreate } = Flux.createUpdate<
  UseCreateArgs,
  SubStore,
  UseCreateResult
>({
  name: "Table",
  update: async ({ client, value, store }) => {
    const { workspace, ...rest } = value;
    const t = await client.workspaces.tables.create(workspace, rest);
    store.tables.set(t.key, t);
    return { ...t, workspace };
  },
});
