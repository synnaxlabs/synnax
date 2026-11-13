// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type project, query, schematic } from "@synnaxlabs/client";
import { compare, verbs } from "@synnaxlabs/x";

import { Flux } from "@/flux";
import { Edge } from "@/schematic/edge";
import { type ElementConfig, elementConfigZ } from "@/schematic/element";
import { Group } from "@/schematic/group";
import { Scope } from "@/schematic/scope";
import { Synnax } from "@/synnax";

const RESOURCE_NAME = "schematic";
const PLURAL_RESOURCE_NAME = "schematics";

export type RetrieveQuery = schematic.RetrieveSingleParams;

// Prefers the cached copy: it may hold locally replayed edits ahead of the server.
export const { use, useEnsure, useTombstone, createSelector } = Flux.createRetrieve<
  RetrieveQuery,
  schematic.Schematic
>({
  name: RESOURCE_NAME,
  retrieve: async ({ client, query }) => await client.schematics.retrieve(query.key),
  onChange: ({ client, query }, handler) =>
    client.schematics.watch(query.key).onChange(handler),
  getCached: ({ client, query }) => client.schematics.watch(query.key).getCached(),
  awaitCreation: true,
});

export type RetrieveMultipleQuery = {
  keys: schematic.Key[];
};

export const { use: useMultiple } = Flux.createRetrieve<
  RetrieveMultipleQuery,
  schematic.Schematic[]
>({
  name: PLURAL_RESOURCE_NAME,
  retrieve: async ({ client, query: { keys } }) =>
    await client.schematics.retrieve(keys),
  onChange: ({ client, query: { keys } }, handler) =>
    client.schematics.watch(keys).onChange(handler),
  getCached: ({ client, query: { keys } }) => client.schematics.watch(keys).getCached(),
});

export interface KeyParams {
  key: schematic.Key;
}

export const useAllNodes = Scope.bindHook(createSelector(({ nodes }) => nodes));

export const useAllEdges = Scope.bindHook(createSelector(({ edges }) => edges));

export const useAllConfigs = Scope.bindHook(createSelector(({ configs }) => configs));

// Value equality keeps the map's reference stable across unrelated config edits,
// so consumers memoizing on it re-run only when group membership changes.
export const useParentOf = Scope.bindHook(
  createSelector(({ configs }) => Group.buildParentOf(configs), compare.mapsEqual),
);

export interface ConfigParams extends KeyParams {
  elKey: string;
}

export const useElementConfig = Scope.bindHook(
  createSelector<ElementConfig | undefined, ConfigParams>(
    ({ configs }, { elKey }) => configs[elKey] as ElementConfig | undefined,
  ),
);

const useDocumentConfigForm = Flux.createDocumentForm<
  schematic.Key,
  schematic.Schematic,
  typeof elementConfigZ,
  ConfigParams
>({
  name: "schematic element",
  schema: elementConfigZ,
  domain: (client) => client.schematics,
  useEnsure,
  toValues: ({ configs }, { elKey }) => configs[elKey],
  pathOf: ({ elKey }) => ["configs", elKey],
});

const useConfigFormBase = (query: ConfigParams) => useDocumentConfigForm({ query });

/** A form over one element's config in the cached schematic. */
export const useConfigForm = Scope.bindHook(useConfigFormBase);

export interface ConfigsParams extends KeyParams {
  keys: string[];
}

export const useConfigs = Scope.bindHook(
  createSelector<Map<string, ElementConfig>, ConfigsParams>(({ configs }, { keys }) => {
    const result = new Map<string, ElementConfig>();
    for (const elKey of keys) {
      const cfg = configs?.[elKey];
      if (cfg != null) result.set(elKey, cfg);
    }
    return result;
  }, compare.mapsEqual),
);

export interface NodesParams extends KeyParams {
  keys: string[];
}

export const useNodes = Scope.bindHook(
  createSelector<schematic.Node[], NodesParams>(
    ({ nodes }, { keys }) => {
      if (keys.length === 0) return [];
      const keySet = new Set(keys);
      return nodes.filter((n) => keySet.has(n.key));
    },
    (a, b) => compare.arraysEqual(a, b),
  ),
);

export const useName = Scope.bindHook(createSelector(({ name }) => name));

export type DeleteParams = schematic.Key | schematic.Key[];

export const { useUpdate: useDelete } = Flux.createUpdate<DeleteParams>({
  name: RESOURCE_NAME,
  verbs: verbs.DELETE,
  update: async ({ client, data, onOptimisticComplete }) => {
    await client.schematics.delete(data, {
      onOptimistic: async () => await onOptimisticComplete(data),
    });
    return data;
  },
});

export interface CopyParams extends schematic.CopyParams {}

export const { useUpdate: useCopy } = Flux.createUpdate<
  CopyParams,
  schematic.Schematic
>({
  name: RESOURCE_NAME,
  verbs: verbs.COPY,
  update: async ({ client, data }) => await client.schematics.copy(data),
});

export interface UseCreateParams extends schematic.New {
  project: project.Key;
}

export const { useUpdate: useCreate } = Flux.createUpdate<
  UseCreateParams,
  schematic.Schematic
>({
  name: RESOURCE_NAME,
  verbs: verbs.CREATE,
  update: async ({ client, data, onOptimisticComplete }) =>
    await client.schematics.create(data, {
      onOptimistic: async ([optimistic]) => await onOptimisticComplete(optimistic),
    }),
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
  const extra = updates.flatMap((u) => {
    const existing = current.configs[u.key];
    if (existing == null || !("segments" in existing)) return [];
    return schematic.setChanged({
      key: u.key,
      base: existing,
      next: { ...existing, segments: u.segments },
    });
  });
  return [...actions, ...extra];
};

const isEdgeConfig = (c: schematic.ElementConfig): c is schematic.EdgeConfig =>
  c.variant in schematic.EDGE_CONFIG_SCHEMAS;

// An edge added in this batch whose config names no stroke takes its source symbol's
// stroke, so a pipe drawn out of a colored symbol matches it.
const inheritEdgeColor = (
  current: schematic.Schematic,
  actions: schematic.Action[],
): schematic.Action[] => {
  const added = new Map<string, schematic.Edge>();
  for (const a of actions)
    if (a.type === "add_edge") added.set(a.addEdge.edge.key, a.addEdge.edge);
  if (added.size === 0) return actions;
  return actions.map((a) => {
    if (a.type !== "set_config") return a;
    const { key, config } = a.setConfig;
    const edge = added.get(key);
    if (edge == null || !isEdgeConfig(config) || config.strokeColor != null) return a;
    const source = current.configs[edge.source.node];
    if (source == null || !("strokeColor" in source) || source.strokeColor == null)
      return a;
    return schematic.setConfig({
      key,
      config: { ...config, strokeColor: source.strokeColor },
    });
  });
};

export const {
  useDispatch,
  useUndo: useUndoBase,
  useRedo: useRedoBase,
  useSingleDispatch: useSingleDispatchBase,
} = Flux.createDispatch<schematic.Key, schematic.Schematic, schematic.Action>({
  domain: (client) => client.schematics,
  // Group fan-out runs first so member moves also get edge segment updates.
  preprocess: (current, actions) =>
    inheritEdgeColor(
      current,
      augmentWithEdgeSegments(current, Group.fanOutMoves(current, actions)),
    ),
});

export const useSingleDispatch = Scope.bindHook(useSingleDispatchBase);
export const useUndo = Scope.bindHook(useUndoBase);
export const useRedo = Scope.bindHook(useRedoBase);

export interface RenameParams extends Pick<schematic.Schematic, "key" | "name"> {}

export const { useUpdate: useRename } = Flux.createUpdate<RenameParams>({
  name: RESOURCE_NAME,
  verbs: verbs.RENAME,
  update: async ({ client, data, onOptimisticComplete }) => {
    const { key, name } = data;
    await client.schematics.rename(key, name, {
      onOptimistic: async () => await onOptimisticComplete(data),
    });
    return data;
  },
});

/**
 * useGroup returns a callback that groups the given selection, dispatched as a
 * single undoable step. Returns the keys to select, the new group first, or null
 * when the selection cannot be grouped.
 */
export const useGroup = (): ((selected: readonly string[]) => string[] | null) => {
  const key = Scope.use();
  const client = Synnax.use();
  const dispatch = useSingleDispatch();
  return (selected) => {
    const s = client?.schematics.watch({ key }).getCached();
    if (!query.isLive(s)) return null;
    const result = Group.createActions({
      selected,
      nodes: s.nodes,
      configs: s.configs,
    });
    if (result == null) return null;
    dispatch(result.actions);
    return result.selection;
  };
};

/**
 * useUngroup returns a callback that dissolves the groups the given selection
 * resolves to, dispatched as a single undoable step. Returns the freed member
 * keys, or null when the selection touches no group.
 */
export const useUngroup = (): ((selected: readonly string[]) => string[] | null) => {
  const key = Scope.use();
  const client = Synnax.use();
  const dispatch = useSingleDispatch();
  return (selected) => {
    const s = client?.schematics.watch({ key }).getCached();
    if (!query.isLive(s)) return null;
    const result = Group.ungroupActions(selected, s.configs);
    if (result == null) return null;
    dispatch(result.actions);
    return result.freed;
  };
};
