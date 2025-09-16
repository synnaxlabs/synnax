// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schematic, type workspace } from "@synnaxlabs/client";
import { array } from "@synnaxlabs/x";

import { Flux } from "@/flux";
import { Ontology } from "@/ontology";

export type UseDeleteArgs = schematic.Params;

export const FLUX_STORE_CONFIG: Flux.UnaryStoreConfig<
  FluxSubStore,
  schematic.Key,
  schematic.Schematic
> = { listeners: [] };

export const FLUX_STORE_KEY = "schematics";

export interface FluxStore
  extends Flux.UnaryStore<schematic.Key, schematic.Schematic> {}

interface FluxSubStore extends Flux.Store {
  [Ontology.RELATIONSHIPS_FLUX_STORE_KEY]: Ontology.RelationshipFluxStore;
  [Ontology.RESOURCES_FLUX_STORE_KEY]: Ontology.ResourceFluxStore;
  [FLUX_STORE_KEY]: FluxStore;
}

export const { useUpdate: useDelete } = Flux.createUpdate<UseDeleteArgs, FluxSubStore>({
  name: "Schematic",
  update: async ({ client, value, rollbacks, store }) => {
    const keys = array.toArray(value);
    const ids = keys.map((k) => schematic.ontologyID(k));
    const relFilter = Ontology.filterRelationshipsThatHaveIDs(ids);
    rollbacks.add(store.relationships.delete(relFilter));
    await client.workspaces.schematics.delete(value);
    return value;
  },
});

export interface UseCopyArgs extends schematic.CopyArgs {}

export const { useUpdate: useCopy } = Flux.createUpdate<
  UseCopyArgs,
  FluxSubStore,
  schematic.Schematic
>({
  name: "Schematic",
  update: async ({ client, value }) => await client.workspaces.schematics.copy(value),
});

export interface UseCreateArgs extends schematic.New {
  workspace: workspace.Key;
}

export interface UseCreateResult extends schematic.Schematic {
  workspace: workspace.Key;
}

export const { useUpdate: useCreate } = Flux.createUpdate<
  UseCreateArgs,
  FluxSubStore,
  UseCreateResult
>({
  name: "Schematic",
  update: async ({ client, value, store }) => {
    const { workspace, ...rest } = value;
    const s = await client.workspaces.schematics.create(workspace, rest);
    store.schematics.set(s.key, s);
    return { ...s, workspace };
  },
});
