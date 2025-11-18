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
import { useCallback, useMemo, useSyncExternalStore } from "react";

import { Flux } from "@/flux";
import { proxyMemo } from "@/memo/proxyMemo";
import { Ontology } from "@/ontology";
import { state } from "@/state";

export const FLUX_STORE_KEY = "schematics";
const RESOURCE_NAME = "Schematic";

const SET_LISTENER: Flux.ChannelListener<FluxSubStore, typeof schematic.scopedActionZ> =
  {
    channel: schematic.SET_CHANNEL_NAME,
    schema: schematic.scopedActionZ,
    onChange: ({ store, changed }) =>
      store.schematics.set(
        changed.key,
        state.skipNull((p) => schematic.reducer(p, changed)),
      ),
  };

export const FLUX_STORE_CONFIG: Flux.UnaryStoreConfig<
  FluxSubStore,
  schematic.Key,
  schematic.Schematic
> = { listeners: [SET_LISTENER] };

export interface FluxStore
  extends Flux.UnaryStore<schematic.Key, schematic.Schematic> {}

interface FluxSubStore extends Flux.Store {
  [FLUX_STORE_KEY]: FluxStore;
  [Ontology.RELATIONSHIPS_FLUX_STORE_KEY]: Ontology.RelationshipFluxStore;
  [Ontology.RESOURCES_FLUX_STORE_KEY]: Ontology.ResourceFluxStore;
}

export type RetrieveQuery = schematic.RetrieveSingleParams;

export const retrieveSingle = async ({
  store,
  client,
  query: { key },
}: Flux.RetrieveParams<RetrieveQuery, FluxSubStore>) => {
  const cached = store.schematics.get(key);
  if (cached != null) return cached;
  const s = await client.workspaces.schematics.retrieve({ key });
  store.schematics.set(s);
  return s;
};

export const { useRetrieve, useRetrieveObservable } = Flux.createRetrieve<
  RetrieveQuery,
  schematic.Schematic,
  FluxSubStore
>({
  name: RESOURCE_NAME,
  retrieve: retrieveSingle,
  mountListeners: ({ store, query: { key }, onChange }) => [
    store.schematics.onSet(onChange, key),
  ],
});

export type DeleteParams = schematic.Params;

export const { useUpdate: useDelete } = Flux.createUpdate<DeleteParams, FluxSubStore>({
  name: RESOURCE_NAME,
  verbs: Flux.DELETE_VERBS,
  update: async ({ client, data, rollbacks, store }) => {
    const keys = array.toArray(data);
    const ids = keys.map((k) => schematic.ontologyID(k));
    const relFilter = Ontology.filterRelationshipsThatHaveIDs(ids);
    rollbacks.push(store.relationships.delete(relFilter));
    await client.workspaces.schematics.delete(data);
    return data;
  },
});

export interface CopyParams extends schematic.CopyArgs {}

export const { useUpdate: useCopy } = Flux.createUpdate<
  CopyParams,
  FluxSubStore,
  schematic.Schematic
>({
  name: RESOURCE_NAME,
  verbs: Flux.COPY_VERBS,
  update: async ({ client, data, store }) => {
    const copy = await client.workspaces.schematics.copy(data);
    store.schematics.set(copy);
    return copy;
  },
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
  name: RESOURCE_NAME,
  verbs: Flux.CREATE_VERBS,
  update: async ({ client, data, store }) => {
    const { workspace, ...rest } = data;
    const s = await client.workspaces.schematics.create(workspace, rest);
    store.schematics.set(s.key, s);
    return { ...s, workspace };
  },
});

export interface SnapshotPair extends Pick<schematic.Schematic, "key" | "name"> {}

export interface SnapshotParams {
  schematics: SnapshotPair | SnapshotPair[];
  parentID: ontology.ID;
}

export const { useUpdate: useSnapshot } = Flux.createUpdate<
  SnapshotParams,
  FluxSubStore
>({
  name: RESOURCE_NAME,
  verbs: Flux.SNAPSHOT_VERBS,
  update: async ({ client, data }) => {
    const { schematics, parentID } = data;
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
    return data;
  },
});

export interface RenameParams extends Pick<schematic.Schematic, "key" | "name"> {}

export const { useUpdate: useRename } = Flux.createUpdate<RenameParams, FluxSubStore>({
  name: RESOURCE_NAME,
  verbs: Flux.RENAME_VERBS,
  update: async ({ client, data, rollbacks, store }) => {
    const { key, name } = data;
    await client.workspaces.schematics.rename(key, name);
    rollbacks.push(Flux.partialUpdate(store.schematics, key, { name }));
    rollbacks.push(Ontology.renameFluxResource(store, schematic.ontologyID(key), name));
    return data;
  },
});

export const { useUpdate } = Flux.createUpdate<schematic.ScopedAction, FluxSubStore>({
  name: RESOURCE_NAME,
  verbs: Flux.UPDATE_VERBS,
  update: async ({ client, data, store, rollbacks }) => {
    rollbacks.push(
      store.schematics.set(
        data.key,
        state.skipNull((p) => schematic.reducer(p, data)),
      ),
    );
    await client.workspaces.schematics.update(data.key, data);
    return data;
  },
});

export const useSelect = <V extends (s: schematic.Schematic) => any>(
  key: string,
  selector: V,
): ReturnType<V> => {
  const store = Flux.useStore();
  const memoizedSelector = useMemo(() => proxyMemo(selector), [selector]);
  const subscribe = useCallback(
    (onStoreChange: () => void) => store.schematics.onSet(onStoreChange, key),
    [store, key],
  );
  const getSnapshot = useCallback(
    () => memoizedSelector(store.schematics.get(key)),
    [store, key, memoizedSelector],
  );
  return useSyncExternalStore(subscribe, getSnapshot);
};

export const useSelectNodePropsAndType = (key: string) =>
  useSelect(
    key,
    useCallback(
      (s) => ({
        props: s.props[key],
        type: s.nodes.find((n) => n.key === key)?.type,
      }),
      [key],
    ),
  );
