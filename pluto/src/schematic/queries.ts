// Copyright 2026 Synnax Labs, Inc.
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
import { type Diagram } from "@/vis/diagram";

export const FLUX_STORE_KEY = "schematics";
const RESOURCE_NAME = "schematic";

const ACTION_LISTENER: Flux.ChannelListener<
  FluxSubStore,
  typeof schematic.scopedActionZ
> = {
  channel: "sy_schematic_set",
  schema: schematic.scopedActionZ,
  onChange: ({ changed, store, client }) => {
    if (client != null && changed.sessionKey === client.key) return;
    const current = store.schematics.get(changed.key);
    if (current == null) return;
    const next = schematic.reduceAll(current, changed.actions);
    store.schematics.set(changed.key, next);
  },
};

export const FLUX_STORE_CONFIG: Flux.UnaryStoreConfig<
  FluxSubStore,
  schematic.Key,
  schematic.Schematic
> = { listeners: [ACTION_LISTENER] };

export interface FluxStore extends Flux.UnaryStore<
  schematic.Key,
  schematic.Schematic
> {}

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
  const s = await client.schematics.retrieve({ key });
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

export interface SelectPropsArgs {
  key: schematic.Key;
  propKey: string;
}

export const useSelectProps = Flux.createSelector<
  FluxSubStore,
  SelectPropsArgs,
  Record<string, unknown> | undefined
>({
  subscribe: (store, { key }, notify) => store.schematics.onSet(notify, key),
  select: (store, { key, propKey }) =>
    store.schematics.get(key)?.props?.[propKey] as Record<string, unknown> | undefined,
});

export interface SelectEdgeArgs {
  key: schematic.Key;
  edgeKey: string;
}

export const useSelectEdge = Flux.createSelector<
  FluxSubStore,
  SelectEdgeArgs,
  Diagram.Edge | undefined
>({
  subscribe: (store, { key }, notify) => store.schematics.onSet(notify, key),
  select: (store, { key, edgeKey }) => {
    const s = store.schematics.get(key);
    return s?.edges?.find((e) => e.key === edgeKey);
  },
});

export interface ElementDigest {
  key: string;
  type: "node" | "edge";
}

export interface SelectElementDigestsArgs {
  key: schematic.Key;
  keys: string[];
}

export const useSelectElementDigests = Flux.createSelector<
  FluxSubStore,
  SelectElementDigestsArgs,
  ElementDigest[]
>({
  subscribe: (store, { key }, notify) => store.schematics.onSet(notify, key),
  select: (store, { key, keys }) => {
    const s = store.schematics.get(key);
    if (s == null || keys.length === 0) return [];
    const keySet = new Set(keys);
    const digests: ElementDigest[] = [];
    for (const node of s.nodes)
      if (keySet.has(node.key)) digests.push({ key: node.key, type: "node" });
    for (const edge of s.edges)
      if (keySet.has(edge.key)) digests.push({ key: edge.key, type: "edge" });
    return digests;
  },
});

export interface NodeElementInfo {
  key: string;
  type: "node";
  node: Diagram.Node;
  props: Record<string, unknown>;
}

export interface EdgeElementInfo {
  key: string;
  type: "edge";
  edge: Diagram.Edge;
}

export type ElementInfo = NodeElementInfo | EdgeElementInfo;

export interface SelectElementsInfoArgs {
  key: schematic.Key;
  keys: string[];
}

export const useSelectElementsInfo = Flux.createSelector<
  FluxSubStore,
  SelectElementsInfoArgs,
  ElementInfo[]
>({
  subscribe: (store, { key }, notify) => store.schematics.onSet(notify, key),
  select: (store, { key, keys }) => {
    const s = store.schematics.get(key);
    if (s == null || keys.length === 0) return [];
    const keySet = new Set(keys);
    const result: ElementInfo[] = [];
    for (const node of s.nodes)
      if (keySet.has(node.key))
        result.push({
          key: node.key,
          type: "node",
          node,
          props: (s.props?.[node.key] as Record<string, unknown>) ?? {},
        });
    for (const edge of s.edges)
      if (keySet.has(edge.key)) result.push({ key: edge.key, type: "edge", edge });
    return result;
  },
});

export interface SelectElementNamesArgs {
  key: schematic.Key;
  keys: string[];
}

export const useSelectElementNames = Flux.createSelector<
  FluxSubStore,
  SelectElementNamesArgs,
  (string | null)[]
>({
  subscribe: (store, { key }, notify) => store.schematics.onSet(notify, key),
  select: (store, { key, keys }) => {
    const s = store.schematics.get(key);
    if (s == null || keys.length === 0) return [];
    const keySet = new Set(keys);
    const result: (string | null)[] = [];
    for (const node of s.nodes) {
      if (!keySet.has(node.key)) continue;
      const p = s.props?.[node.key] as Record<string, unknown> | undefined;
      const label = (p?.label as Record<string, unknown> | undefined)?.label;
      result.push(typeof label === "string" ? label : null);
    }
    return result;
  },
});

export interface SelectFieldArgs {
  key: schematic.Key;
}

export const useSelectSnapshot = Flux.createSelector<
  FluxSubStore,
  SelectFieldArgs,
  boolean | undefined
>({
  subscribe: (store, { key }, notify) => store.schematics.onSet(notify, key),
  select: (store, { key }) => store.schematics.get(key)?.snapshot,
});

export const useSelectAuthority = Flux.createSelector<
  FluxSubStore,
  SelectFieldArgs,
  number | undefined
>({
  subscribe: (store, { key }, notify) => store.schematics.onSet(notify, key),
  select: (store, { key }) => store.schematics.get(key)?.authority,
});

export const useSelectViewport = Flux.createSelector<
  FluxSubStore,
  SelectFieldArgs,
  Diagram.Viewport | undefined
>({
  subscribe: (store, { key }, notify) => store.schematics.onSet(notify, key),
  select: (store, { key }) => store.schematics.get(key)?.viewport,
});

export type DeleteParams = schematic.Key | schematic.Key[];

export const { useUpdate: useDelete } = Flux.createUpdate<DeleteParams, FluxSubStore>({
  name: RESOURCE_NAME,
  verbs: Flux.DELETE_VERBS,
  update: async ({ client, data, rollbacks, store }) => {
    const keys = array.toArray(data);
    const ids = schematic.ontologyID(keys);
    const relFilter = Ontology.filterRelationshipsThatHaveIDs(ids);
    rollbacks.push(store.relationships.delete(relFilter));
    await client.schematics.delete(data);
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
    const copy = await client.schematics.copy(data);
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
    const s = await client.schematics.create(workspace, rest);
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
        const newSchematic = await client.schematics.copy({
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

export interface DispatchParams {
  key: schematic.Key;
  actions: schematic.Action | schematic.Action[];
}

export const { useUpdate: useDispatch } = Flux.createUpdate<
  DispatchParams,
  FluxSubStore
>({
  name: RESOURCE_NAME,
  verbs: Flux.UPDATE_VERBS,
  update: async ({ client, data, store, rollbacks }) => {
    const { key, actions } = data;
    const actionArray = Array.isArray(actions) ? actions : [actions];
    const current = store.schematics.get(key);
    if (current != null) {
      const next = schematic.reduceAll(current, actionArray);
      rollbacks.push(store.schematics.set(key, next));
    }
    await client.schematics.dispatch(key, actionArray, client.key);
    return data;
  },
});

export interface RenameParams extends Pick<schematic.Schematic, "key" | "name"> {}

export const { useUpdate: useRename } = Flux.createUpdate<RenameParams, FluxSubStore>({
  name: RESOURCE_NAME,
  verbs: Flux.RENAME_VERBS,
  update: async ({ client, data, rollbacks, store }) => {
    const { key, name } = data;
    await client.schematics.rename(key, name);
    rollbacks.push(Flux.partialUpdate(store.schematics, key, { name }));
    rollbacks.push(Ontology.renameFluxResource(store, schematic.ontologyID(key), name));
    return data;
  },
});
