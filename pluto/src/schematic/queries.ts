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
  query,
  schematic,
  type Synnax as Client,
} from "@synnaxlabs/client";
import { array, compare, type record, uuid, verbs, xy } from "@synnaxlabs/x";
import { useCallback } from "react";

import { Flux } from "@/flux";
import { Edge } from "@/schematic/edge";
import { type ElementConfig } from "@/schematic/element";
import { Node } from "@/schematic/node";
import { Scope } from "@/schematic/scope";
import { Synnax } from "@/synnax";
import { Theming } from "@/theming";

const RESOURCE_NAME = "schematic";

export type RetrieveQuery = schematic.RetrieveSingleParams;

// Prefers the cached copy: it may hold locally replayed edits ahead of the
// server.
export const { useRetrieveSuspended, useRetrieveObservable, useEnsureRetrieved } =
  Flux.createRetrieve<RetrieveQuery, schematic.Schematic>({
    name: RESOURCE_NAME,
    retrieve: async ({ client, query }) => await client.schematics.retrieve(query),
    subscribe: ({ client, query }, handler) =>
      client.schematics.onChange(query, handler),
    getCached: ({ client, query }) => client.schematics.getCached(query),
  });

export interface SelectKeyParams {
  key: schematic.Key;
}

const requireSchematic = (
  client: Client | null,
  key: schematic.Key,
): schematic.Schematic => {
  const cached = client?.schematics.getCached({ key });
  if (cached == null || query.Deleted.matches(cached))
    throw new NotFoundError(`Schematic with key ${key} not found`);
  return cached;
};

const getSchematic = (
  client: Client | null,
  key: schematic.Key,
): schematic.Schematic | undefined => {
  const cached = client?.schematics.getCached({ key });
  if (cached == null || query.Deleted.matches(cached)) return undefined;
  return cached;
};

const subscribe = (
  { client, args: { key } }: Flux.SelectorParams<SelectKeyParams>,
  notify: () => void,
) => (client == null ? () => {} : client.schematics.onChange({ key }, notify));

export const [useSelectAllNodes, useGetAllNodes] = Scope.bindSelector(
  Flux.createSelector<SelectKeyParams, schematic.Node[]>({
    subscribe,
    select: ({ client, args: { key } }) => requireSchematic(client, key).nodes,
  }),
);

export const [useSelectAllEdges, useGetAllEdges] = Scope.bindSelector(
  Flux.createSelector<SelectKeyParams, schematic.Edge[]>({
    subscribe,
    select: ({ client, args: { key } }) => requireSchematic(client, key).edges,
  }),
);

export interface SelectConfigParams extends SelectKeyParams {
  elKey: string;
}

export const [useSelectElementConfig, useGetElementConfig] = Scope.bindSelector(
  Flux.createSelector<SelectConfigParams, ElementConfig | undefined>({
    subscribe,
    select: ({ client, args: { key, elKey } }) =>
      requireSchematic(client, key).configs[elKey] as ElementConfig | undefined,
  }),
);

export interface SelectConfigsParams extends SelectKeyParams {
  keys: string[];
}

export const [useSelectConfigs, useGetConfigs] = Scope.bindSelector(
  Flux.createSelector<SelectConfigsParams, Map<string, ElementConfig>>({
    subscribe,
    select: ({ client, args: { key, keys } }) => {
      const result = new Map<string, ElementConfig>();
      const s = getSchematic(client, key);
      if (s == null || keys.length === 0) return result;
      for (const elKey of keys) {
        const cfg = s.configs?.[elKey];
        if (cfg != null) result.set(elKey, cfg as ElementConfig);
      }
      return result;
    },
    equal: compare.mapsEqual,
  }),
);

export interface SelectNodesParams extends SelectKeyParams {
  keys: string[];
}

export const [useSelectNodes, useGetNodes] = Scope.bindSelector(
  Flux.createSelector<SelectNodesParams, schematic.Node[]>({
    subscribe,
    select: ({ client, args: { key, keys } }) => {
      const s = getSchematic(client, key);
      if (s == null || keys.length === 0) return [];
      const keySet = new Set(keys);
      return s.nodes.filter((n) => keySet.has(n.key));
    },
    equal: compare.arraysEqual,
  }),
);

export interface SelectFieldParams {
  key: schematic.Key;
}

export const [useSelectSnapshot, useGetSnapshot] = Scope.bindSelector(
  Flux.createSelector<SelectFieldParams, boolean>({
    subscribe,
    select: ({ client, args: { key } }) => requireSchematic(client, key).snapshot,
  }),
);

export const [useSelectName, useGetName] = Scope.bindSelector(
  Flux.createSelector<SelectKeyParams, string>({
    subscribe,
    select: ({ client, args: { key } }) => requireSchematic(client, key).name,
  }),
);

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
  const extra = updates.map((u) => {
    const existing = current.configs[u.key] as record.Unknown | undefined;
    return schematic.setConfig({
      key: u.key,
      config: { ...existing, segments: u.segments },
    });
  });
  return [...actions, ...extra];
};

export const {
  useDispatch,
  useUndo: useUndoBase,
  useRedo: useRedoBase,
  useSingleDispatch: useSingleDispatchBase,
} = Flux.createDispatch<schematic.Key, schematic.Schematic, schematic.Action>({
  domain: (client) => client.schematics,
  preprocess: augmentWithEdgeSegments,
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
    await onOptimisticComplete(data);
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

export const useAddNode = () => {
  const client = Synnax.use();
  const theme = Theming.use();
  const dispatch = useSingleDispatch();

  return useCallback(
    ({ key, variant, position, specKey, config: override }: AddNodeProps) => {
      const config = Node.resolveSpec(variant).defaultConfig(theme);
      if (Node.isCustomConfig(config) && specKey != null) {
        config.specKey = specKey;
        const sym = client?.schematics.symbols.getCached({ key: specKey });
        if (config.label != null && sym != null && !query.Deleted.matches(sym))
          config.label.label = sym.name;
      }
      dispatch(
        schematic.setNode({
          node: { key, position: position ?? xy.ZERO },
          config: { ...config, ...override, variant },
        }),
      );
    },
    [dispatch, theme, client],
  );
};
