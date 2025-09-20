// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology, schematic, type workspace } from "@synnaxlabs/client";
import { array } from "@synnaxlabs/x";

import { Flux } from "@/flux";
import { Ontology } from "@/ontology";

export type UseDeleteArgs = schematic.Params;

export const FLUX_STORE_CONFIG: Flux.UnaryStoreConfig<
  FluxStore,
  schematic.Key,
  schematic.Schematic
> = { listeners: [] };

export const FLUX_STORE_KEY = "schematics";

export interface FluxStore
  extends Flux.UnaryStore<schematic.Key, schematic.Schematic> {}

interface FluxStore extends Flux.Store {
  [Ontology.RELATIONSHIPS_FLUX_STORE_KEY]: Ontology.RelationshipFluxStore;
  [Ontology.RESOURCES_FLUX_STORE_KEY]: Ontology.ResourceFluxStore;
  [FLUX_STORE_KEY]: FluxStore;
}

export type UseRetrieveArgs = schematic.RetrieveSingleParams;

export const retrieveSingle = async ({
  store,
  client,
  params: { key },
}: Flux.RetrieveArgs<UseRetrieveArgs, FluxStore>) => {
  const cached = store.schematics.get(key);
  if (cached != null) return cached;
  const s = await client.workspaces.schematics.retrieve({ key });
  store.schematics.set(s);
  return s;
};

export const { useRetrieve } = Flux.createRetrieve<
  UseRetrieveArgs,
  schematic.Schematic,
  FluxStore
>({
  name: "Schematic",
  retrieve: retrieveSingle,
  mountListeners: ({ store, params: { key }, onChange }) => [
    store.schematics.onSet(onChange, key),
  ],
});

export const { useUpdate: useDelete } = Flux.createUpdate<UseDeleteArgs, FluxStore>({
  name: "Schematic",
  update: async ({ client, data, rollbacks, store }) => {
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
  FluxStore,
  schematic.Schematic
>({
  name: "Schematic",
  update: async ({ client, data }) => await client.workspaces.schematics.copy(value),
});

export interface UseCreateArgs extends schematic.New {
  workspace: workspace.Key;
}

export interface UseCreateResult extends schematic.Schematic {
  workspace: workspace.Key;
}

export const { useUpdate: useCreate } = Flux.createUpdate<
  UseCreateArgs,
  FluxStore,
  UseCreateResult
>({
  name: "Schematic",
  update: async ({ client, data, store }) => {
    const { workspace, ...rest } = value;
    const s = await client.workspaces.schematics.create(workspace, rest);
    store.schematics.set(s.key, s);
    return { ...s, workspace };
  },
});

export interface SnapshotPair extends Pick<schematic.Schematic, "key" | "name"> {}

export interface UseSnapshotArgs {
  schematics: SnapshotPair | SnapshotPair[];
  parentID: ontology.ID;
}

export const { useUpdate: useCreateSnapshot } = Flux.createUpdate<UseSnapshotArgs>({
  name: "Schematic",
  update: async ({ client, data }) => {
    const { schematics, parentID } = value;
    const ids = await Promise.all(
      array.toArray(schematics).map(async (s) => {
        const newSchematic = await client.workspaces.schematics.copy({
          key: s.key,
          name: `${s.name} (Snapshot)`,
          snapshot: true,
        });
        return schematic.ontologyID(newSchematic.key);
      }),
    );
    await client.ontology.addChildren(parentID, ...ids);
    return value;
  },
});

export interface UseRenameArgs {
  key: schematic.Key;
  name: string;
}

export const { useUpdate: useRename } = Flux.createUpdate<UseRenameArgs, FluxStore>({
  name: "Schematic",
  update: async ({ client, data, rollbacks, store }) => {
    const { key, name } = value;
    await client.workspaces.schematics.rename(key, name);
    rollbacks.add(Flux.partialUpdate(store.schematics, key, { name }));
    rollbacks.add(Ontology.renameFluxResource(store, schematic.ontologyID(key), name));
    return value;
  },
});
