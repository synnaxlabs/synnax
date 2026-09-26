// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology, type project, query, schematic } from "@synnaxlabs/client";
import { array, compare, uuid, verbs, xy } from "@synnaxlabs/x";
import { useCallback } from "react";

import { Flux } from "@/flux";
import { Edge } from "@/schematic/edge";
import { type ElementConfig } from "@/schematic/element";
import { Group } from "@/schematic/group";
import { Node } from "@/schematic/node";
import { Scope } from "@/schematic/scope";
import { Synnax } from "@/synnax";

const RESOURCE_NAME = "schematic";

export type RetrieveQuery = schematic.RetrieveSingleParams;

// Prefers the cached copy: it may hold locally replayed edits ahead of the server.
export const { use, useEnsure, useTombstone, createSelector } = Flux.createRetrieve<
  RetrieveQuery,
  schematic.Schematic
>({
  name: RESOURCE_NAME,
  retrieve: async ({ client, query }) => await client.schematics.retrieve(query),
  onChange: ({ client, query }, handler) => client.schematics.onChange(query, handler),
  getCached: ({ client, query }) => client.schematics.getCached(query),
  awaitCreation: true,
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

export const useIsSnapshot = Scope.bindHook(createSelector(({ snapshot }) => snapshot));

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
  project?: project.Key;
}

export const { useUpdate: useCreate } = Flux.createUpdate<
  UseCreateParams,
  schematic.Schematic
>({
  name: RESOURCE_NAME,
  verbs: verbs.CREATE,
  update: async ({ client, data, onOptimisticComplete }) =>
    await client.schematics.create(data.project ?? uuid.ZERO, data, {
      onOptimistic: async ([optimistic]) => await onOptimisticComplete(optimistic),
    }),
});

export interface SnapshotPair extends Pick<schematic.Schematic, "key" | "name"> {}

export interface SnapshotParams {
  schematics: SnapshotPair | SnapshotPair[];
  parentID: ontology.ID;
}

export const { useUpdate: useSnapshot } = Flux.createUpdate<SnapshotParams>({
  name: RESOURCE_NAME,
  verbs: verbs.SNAPSHOT,
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
  const extra = updates.flatMap((u) => {
    const existing = current.configs[u.key];
    if (existing == null || !("segments" in existing)) return [];
    return schematic.setConfig({
      key: u.key,
      config: { ...existing, segments: u.segments },
    });
  });
  return [...actions, ...extra];
};

const isEdgeConfig = (c: schematic.ElementConfig): c is schematic.EdgeConfig =>
  c.variant in schematic.EDGE_CONFIG_SCHEMAS;

// An edge added in this batch whose config names no color takes its source
// symbol's color, so a pipe drawn out of a colored symbol matches it.
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
    if (edge == null || !isEdgeConfig(config) || config.color != null) return a;
    const source = current.configs[edge.source.node];
    if (source == null || !("color" in source) || source.color == null) return a;
    return schematic.setConfig({ key, config: { ...config, color: source.color } });
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

export interface AddNodeProps<V extends Node.Variant = Node.Variant> {
  key: string;
  position?: xy.XY;
  config: Node.Input<V>;
}

export const useAddNode = () => {
  const client = Synnax.use();
  const dispatch = useSingleDispatch();

  return useCallback(
    <V extends Node.Variant>({ key, position, config: input }: AddNodeProps<V>) => {
      const config: Node.Config = Node.createConfig(input);
      if (Node.isCustomConfig(config)) {
        const sym = client?.schematics.symbols.getCached(config.specKey);
        if (query.isLive(sym)) config.label.label = sym.name;
      }
      dispatch(
        schematic.setNode({ node: { key, position: position ?? xy.ZERO }, config }),
      );
    },
    [dispatch, client],
  );
};

/**
 * useGroup returns a callback that groups the given selection, dispatched as a
 * single undoable step. Returns the keys to select, the new group first, or null
 * when the selection cannot be grouped.
 */
export const useGroup = (): ((selected: readonly string[]) => string[] | null) => {
  const key = Scope.use();
  const client = Synnax.use();
  const dispatch = useSingleDispatch();
  return useCallback(
    (selected) => {
      const s = client?.schematics.getCached({ key });
      if (!query.isLive(s)) return null;
      const result = Group.createActions({
        selected,
        nodes: s.nodes,
        configs: s.configs,
      });
      if (result == null) return null;
      dispatch(result.actions);
      return result.selection;
    },
    [client, key, dispatch],
  );
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
  return useCallback(
    (selected) => {
      const s = client?.schematics.getCached({ key });
      if (!query.isLive(s)) return null;
      const result = Group.ungroupActions(selected, s.configs);
      if (result == null) return null;
      dispatch(result.actions);
      return result.freed;
    },
    [client, key, dispatch],
  );
};
