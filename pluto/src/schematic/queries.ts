// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology, schematic, type workspace } from "@synnaxlabs/client";
import { array, caseconv, id, type record, xy } from "@synnaxlabs/x";
import { useCallback } from "react";
import z from "zod";

import { Flux } from "@/flux";
import { Ontology } from "@/ontology";
import { connector } from "@/schematic/edge/connector";
import { Symbol } from "@/schematic/symbol";
import { Theming } from "@/theming";

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
  record.Unknown | undefined
>({
  subscribe: (store, { key }, notify) => store.schematics.onSet(notify, key),
  select: (store, { key, propKey }) => {
    const schem = store.schematics.get(key);
    if (schem == null) return undefined;
    const props = schem.props[propKey] as record.Unknown | undefined;
    if (props != null) return props;
    return schem.props[caseconv.snakeToCamel(propKey)] as
      | record.Unknown
      | undefined;
  },
});

export interface SelectEdgeArgs {
  key: schematic.Key;
  edgeKey: string;
}

export const useSelectEdge = Flux.createSelector<
  FluxSubStore,
  SelectEdgeArgs,
  schematic.Edge | undefined
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
  node: schematic.Node;
  props: record.Unknown;
}

export interface EdgeElementInfo {
  key: string;
  type: "edge";
  edge: schematic.Edge;
  props: record.Unknown;
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
          props: (s.props?.[node.key] as record.Unknown) ?? {},
        });
    for (const edge of s.edges)
      if (keySet.has(edge.key))
        result.push({
          key: edge.key,
          type: "edge",
          edge,
          props: (s.props?.[edge.key] as record.Unknown) ?? {},
        });
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
      const p = s.props?.[node.key] as record.Unknown | undefined;
      const label = (p?.label as record.Unknown | undefined)?.label;
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
  workspace?: workspace.Key;
}

export interface UseCreateResult extends schematic.Schematic {
  workspace?: workspace.Key;
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

const augmentWithEdgeSegments = (
  current: schematic.Schematic,
  actions: schematic.Action[],
): schematic.Action[] => {
  const changes: connector.NodePositionChange[] = [];
  for (const action of actions)
    if (action.type === "set_node_position")
      changes.push({
        key: action.setNodePosition.key,
        newPos: action.setNodePosition.position,
      });
  if (changes.length === 0) return actions;
  const updates = connector.updateSegmentsForPositionChanges({
    nodes: current.nodes,
    edges: current.edges,
    props: current.props as Record<
      string,
      { segments?: connector.Segment[] } | undefined
    >,
    changes,
  });
  if (updates.length === 0) return actions;
  const extra = updates.map((u) => {
    const edgeProps = current.props[u.key] as
      | { segments?: connector.Segment[]; variant?: string; color?: string }
      | undefined;
    return schematic.setProps({
      key: u.key,
      props: {
        segments: u.segments,
        variant: edgeProps?.variant,
        color: edgeProps?.color,
      },
    });
  });
  return [...actions, ...extra];
};

export const { useUpdate: useDispatch } = Flux.createUpdate<
  DispatchParams,
  FluxSubStore
>({
  name: RESOURCE_NAME,
  verbs: Flux.UPDATE_VERBS,
  update: async ({ client, data, store, rollbacks }) => {
    const { key, actions } = data;
    let actionArray = Array.isArray(actions) ? actions : [actions];
    const current = store.schematics.get(key);
    if (current != null) {
      actionArray = augmentWithEdgeSegments(current, actionArray);
      const next = schematic.reduceAll(current, actionArray);
      rollbacks.push(store.schematics.set(key, next));
    }
    await client.schematics.dispatch(key, client.key, actionArray);
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

const dropDataZ = z.object({
  specKey: schematic.symbol.keyZ,
});

export const useAddNode = (resourceKey: string) => {
  const store = Flux.useStore<Symbol.FluxSubStore>();
  const theme = Theming.use();
  const { update: dispatch } = useDispatch();

  return useCallback(
    (key: string, position?: xy.XY, data?: unknown) => {
      let variant: Symbol.Variant;
      let initialName: string | undefined;
      let symbol: schematic.symbol.Symbol | undefined;
      const parsedData = dropDataZ.safeParse(data);
      if (parsedData.success)
        symbol = store.schematicSymbols.get(parsedData.data.specKey);
      if (symbol != null) {
        variant = symbol.data.states.length === 1 ? "customStatic" : "customActuator";
        initialName = symbol.name;
      } else variant = key as Symbol.Variant;
      const spec = Symbol.REGISTRY[variant];
      const initialProps = spec.defaultProps(theme) as record.Unknown & {
        specKey?: string;
        label?: { label?: string };
      };
      if (symbol != null) {
        initialProps.specKey = key;
        if (initialProps.label != null && initialName != null)
          initialProps.label.label = initialName;
      }
      const nodeKey = id.create();
      const node: schematic.Node = {
        key: nodeKey,
        position: position ?? xy.ZERO,
      };
      const props: record.Unknown = {
        variant,
        ...initialProps,
        ...(parsedData.success ? parsedData.data : {}),
      };
      dispatch({
        key: resourceKey,
        actions: [schematic.addNode({ node, props })],
      });
    },
    [dispatch, resourceKey, theme, store],
  );
};
