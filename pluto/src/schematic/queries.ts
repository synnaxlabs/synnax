// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  NotFoundError,
  type ontology,
  type project,
  schematic,
} from "@synnaxlabs/client";
import { array, compare, type record, uuid, xy } from "@synnaxlabs/x";
import { useCallback } from "react";

import { Flux } from "@/flux";
import { useSyncedRef } from "@/hooks/ref";
import { Ontology } from "@/ontology";
import { Edge } from "@/schematic/edge";
import { type ElementConfig } from "@/schematic/element";
import { Node } from "@/schematic/node";
import { type Symbol } from "@/schematic/symbol";
import { Theming } from "@/theming";

export const FLUX_STORE_KEY = "schematics";
const RESOURCE_NAME = "schematic";

export interface FluxStore extends Flux.UndoableUnaryStore<
  schematic.Key,
  schematic.Schematic,
  schematic.Action
> {}

export interface FluxSubStore extends Flux.Store {
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

export const { useRetrieveSuspended, useRetrieveObservable, useEnsureRetrieved } =
  Flux.createRetrieve<RetrieveQuery, schematic.Schematic, FluxSubStore>({
    name: RESOURCE_NAME,
    retrieve: retrieveSingle,
    mountListeners: ({ store, query: { key }, onChange }) =>
      store.schematics.onSet(onChange, key),
  });

export interface useRetrieveObservableNameParams extends Omit<
  Flux.UseRetrieveObservableParams<RetrieveQuery, schematic.Schematic>,
  "onChange"
> {
  onChange: (name: string) => void;
}

export const useRetrieveObservableName = ({
  onChange,
  ...params
}: useRetrieveObservableNameParams): Flux.UseRetrieveObservableReturn<RetrieveQuery> => {
  const onChangeRef = useSyncedRef(onChange);
  return useRetrieveObservable({
    ...params,
    onChange: useCallback(
      (result) => result.variant === "success" && onChangeRef.current(result.data.name),
      [],
    ),
  });
};

export interface SelectKeyArgs {
  key: schematic.Key;
}

const requireSchematic = (
  store: FluxSubStore,
  key: schematic.Key,
): schematic.Schematic => {
  const schem = store.schematics.get(key);
  if (schem == null) throw new NotFoundError(`Schematic with key ${key} not found`);
  return schem;
};

export const useSelectAllNodes = Flux.createSelector<
  FluxSubStore,
  SelectKeyArgs,
  schematic.Node[]
>({
  subscribe: (store, { key }, notify) => store.schematics.onSet(notify, key),
  select: (store, { key }) => requireSchematic(store, key).nodes,
});

export const useSelectAllEdges = Flux.createSelector<
  FluxSubStore,
  SelectKeyArgs,
  schematic.Edge[]
>({
  subscribe: (store, { key }, notify) => store.schematics.onSet(notify, key),
  select: (store, { key }) => requireSchematic(store, key).edges,
});

export interface SelectConfigArgs {
  key: schematic.Key;
  elKey: string;
}

export const useSelectElementConfig = Flux.createSelector<
  FluxSubStore,
  SelectConfigArgs,
  ElementConfig
>({
  subscribe: (store, { key }, notify) => store.schematics.onSet(notify, key),
  select: (store, { key, elKey }) =>
    requireSchematic(store, key).configs[elKey] as ElementConfig,
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

export interface SelectConfigsArgs {
  key: schematic.Key;
  keys: string[];
}

export const useSelectConfigs = Flux.createSelector<
  FluxSubStore,
  SelectConfigsArgs,
  Map<string, ElementConfig>
>({
  subscribe: (store, { key }, notify) => store.schematics.onSet(notify, key),
  select: (store, { key, keys }) => {
    const result = new Map<string, ElementConfig>();
    const s = store.schematics.get(key);
    if (s == null || keys.length === 0) return result;
    for (const elKey of keys) {
      const cfg = s.configs?.[elKey];
      if (cfg != null) result.set(elKey, cfg as ElementConfig);
    }
    return result;
  },
  equal: compare.mapsEqual,
});

export interface SelectNodesArgs {
  key: schematic.Key;
  keys: string[];
}

export const useSelectNodes = Flux.createSelector<
  FluxSubStore,
  SelectNodesArgs,
  schematic.Node[]
>({
  subscribe: (store, { key }, notify) => store.schematics.onSet(notify, key),
  select: (store, { key, keys }) => {
    const s = store.schematics.get(key);
    if (s == null || keys.length === 0) return [];
    const keySet = new Set(keys);
    return s.nodes.filter((n) => keySet.has(n.key));
  },
  equal: compare.arraysEqual,
});

export interface SelectFieldArgs {
  key: schematic.Key;
}

export const useSelectSnapshot = Flux.createSelector<
  FluxSubStore,
  SelectFieldArgs,
  boolean
>({
  subscribe: (store, { key }, notify) => store.schematics.onSet(notify, key),
  select: (store, { key }) => requireSchematic(store, key).snapshot,
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
    rollbacks.push(store.schematics.delete(keys));
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
  project?: project.Key;
}

export const { useUpdate: useCreate } = Flux.createUpdate<
  UseCreateArgs,
  FluxSubStore,
  schematic.Schematic
>({
  name: RESOURCE_NAME,
  verbs: Flux.CREATE_VERBS,
  update: async ({ client, data, store, rollbacks }) => {
    const optimistic = schematic.schematicZ.parse(data);
    rollbacks.push(store.schematics.set(optimistic));
    const project = data.project ?? uuid.ZERO;
    const created = await client.schematics.create(project, optimistic);
    store.schematics.set(created);
    return created;
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

const augmentWithEdgeSegments = (
  current: schematic.Schematic,
  actions: schematic.Action[],
): schematic.Action[] => {
  const changes: Edge.Segmented.NodePositionChange[] = [];
  for (const action of actions)
    if (action.type === "set_node_position")
      changes.push({
        key: action.setNodePosition.key,
        newPos: action.setNodePosition.position,
      });
  if (changes.length === 0) return actions;
  const updates = Edge.Segmented.updateSegmentsForPositionChanges({
    nodes: current.nodes,
    edges: current.edges,
    props: current.configs,
    changes,
  });
  if (updates.length === 0) return actions;
  const extra = updates.map((u) => {
    const existing = current.configs[u.key] as record.Unknown | undefined;
    return schematic.setConfig({
      key: u.key,
      config: { ...existing, segments: u.segments },
    });
  });
  return [...actions, ...extra];
};

const kindOfTransaction = (actions: schematic.Action[]): string => {
  if (actions.length === 0) return "default";
  // A drag dispatches a stream of `set_node_position` per frame, plus
  // `set_config` companions synthesized by augmentWithEdgeSegments for any
  // affected edges. Both shapes are part of one user gesture and must coalesce
  // together — classify them all as "move" so the per-kind coalesce window
  // collapses them into a single undoable.
  const hasMove = actions.some((a) => a.type === "set_node_position");
  const onlyMoveOrSegment = actions.every(
    (a) => a.type === "set_node_position" || a.type === "set_config",
  );
  if (hasMove && onlyMoveOrSegment) return "move";
  if (actions.length === 1) return actions[0].type;
  return "transaction";
};

export const FLUX_STORE_CONFIG = Flux.createUndoableStore<
  schematic.Key,
  schematic.Schematic,
  schematic.Action,
  typeof FLUX_STORE_KEY,
  FluxSubStore
>({
  storeKey: FLUX_STORE_KEY,
  reduce: schematic.reduceAll,
  preprocess: augmentWithEdgeSegments,
  channel: schematic.SET_CHANNEL_NAME,
  schema: schematic.scopedActionZ,
  kindOf: kindOfTransaction,
});

export const { useDispatch, useUndo, useRedo } = Flux.createDispatch<
  schematic.Key,
  schematic.Schematic,
  schematic.Action,
  typeof FLUX_STORE_KEY,
  FluxSubStore
>({
  storeKey: FLUX_STORE_KEY,
  send: ({ client, key, actions, dispatchKey }) =>
    client.schematics.dispatch(key, dispatchKey, actions),
});

export interface RenameParams extends Pick<schematic.Schematic, "key" | "name"> {}

export const { useUpdate: useRename } = Flux.createUpdate<RenameParams, FluxSubStore>({
  name: RESOURCE_NAME,
  verbs: Flux.RENAME_VERBS,
  update: async ({ client, data, rollbacks, store }) => {
    const { key, name } = data;
    const current = store.schematics.get(key);
    if (current != null)
      rollbacks.push(store.schematics.set(key, { ...current, name }));
    rollbacks.push(Ontology.renameFluxResource(store, schematic.ontologyID(key), name));
    await client.schematics.rename(key, name);
    return data;
  },
});

export interface AddNodeProps {
  key: string;
  variant: Node.Variant;
  position?: xy.XY;
  specKey?: string;
  config?: Node.Config;
}

export const useAddNode = (resourceKey: string) => {
  const store = Flux.useStore<Symbol.FluxSubStore>();
  const theme = Theming.use();
  const { dispatch } = useDispatch();

  return useCallback(
    ({ key, variant, position, specKey, config: override }: AddNodeProps) => {
      const config = Node.resolveSpec(variant).defaultConfig(theme);
      if (Node.isCustomConfig(config) && specKey != null) {
        config.specKey = specKey;
        const sym = store.schematicSymbols.get(specKey);
        if (config.label != null && sym != null) config.label.label = sym.name;
      }
      dispatch({
        key: resourceKey,
        actions: [
          schematic.setNode({
            node: { key, position: position ?? xy.ZERO },
            config: { ...config, ...override, variant },
          }),
        ],
      });
    },
    [dispatch, resourceKey, theme, store],
  );
};
