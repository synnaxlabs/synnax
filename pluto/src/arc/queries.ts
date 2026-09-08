// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc, type rack, status, task } from "@synnaxlabs/client";
import { compare, type record, verbs, xy } from "@synnaxlabs/x";
import { useCallback } from "react";
import z from "zod";

import { Node } from "@/arc/graph/node";
import { Scope } from "@/arc/scope";
import { Flux } from "@/flux";
import { type List } from "@/list";
import { Task } from "@/task";
import { Theming } from "@/theming";
import { type Diagram } from "@/vis/diagram";

const RESOURCE_NAME = "Arc";
const PLURAL_RESOURCE_NAME = "Arcs";

const {
  useDispatch,
  useUndo: useUndoBase,
  useRedo: useRedoBase,
  useSingleDispatch: useSingleDispatchBase,
} = Flux.createDispatch<arc.Key, arc.Arc, arc.Action>({
  domain: (client) => client.arcs,
});

export { useDispatch };
export const useUndo = Scope.bindHook(useUndoBase);
export const useRedo = Scope.bindHook(useRedoBase);
export const useSingleDispatch = Scope.bindHook(useSingleDispatchBase);

export interface KeyParams {
  key: arc.Key;
}

export type RetrieveQuery = {
  key: arc.Key;
  includeStatus?: boolean;
};

export const { use, useEnsure, useTombstone, useResult, createSelector } =
  Flux.createRetrieve<RetrieveQuery, arc.Arc>({
    name: RESOURCE_NAME,
    retrieve: async ({ client, query }) => await client.arcs.retrieve(query),
    onChange: ({ client, query: { key, includeStatus } }, handler) =>
      client.arcs.onChange({ key, includeStatus }, handler),
    getCached: ({ client, query: { key, includeStatus } }) =>
      client.arcs.getCached({ key, includeStatus }),
    awaitCreation: true,
  });

// useAllNodes returns every graph node of the Arc with the given key as diagram
// nodes. graph.Node is a structural superset of Diagram.Node, so the cached array
// is returned by reference with no translation, keeping selections referentially
// stable across unrelated cache updates.
export const useAllNodes = Scope.bindHook(
  createSelector<Diagram.Node[]>(({ graph }) => graph.nodes),
);

export interface NodesParams extends KeyParams {
  keys: string[];
}

// useNodes returns only the graph nodes whose keys are in the given set. The
// result is compared by value, so a consumer that tracks a selection re-renders only
// when its nodes change, not on every node mutation.
export const useNodes = Scope.bindHook(
  createSelector<Diagram.Node[], NodesParams>(
    ({ graph }, { keys }) => {
      if (keys.length === 0) return [];
      const keySet = new Set(keys);
      return graph.nodes.filter((n) => keySet.has(n.key));
    },
    (a, b) => compare.arraysEqual(a, b),
  ),
);

// useAllEdges returns every graph edge of the Arc with the given key as diagram
// edges. graph.Edge is a structural superset of Diagram.Edge, so the cached array
// is returned by reference with no translation, keeping selections referentially
// stable across unrelated cache updates.
export const useAllEdges = Scope.bindHook(
  createSelector<Diagram.Edge[]>(({ graph }) => graph.edges),
);

export interface NodePropsParams extends KeyParams {
  nodeKey: string;
}

// useNodeConfig returns the typed config for a single graph node. Returned by
// reference, so the selection only re-runs when that node's config changes.
export const useNodeConfig = Scope.bindHook(
  createSelector<Node.Config, NodePropsParams>(
    ({ graph }, { nodeKey }) => graph.inputs[nodeKey] as Node.Config,
  ),
);

// useMode returns the representation mode of the Arc with the given key. It
// requires the arc to be cached, so callers must render it beneath an Arc.Suspended
// boundary that has retrieved the arc.
export const useMode = Scope.bindHook(createSelector(({ mode }) => mode));

// useHasText reports whether the Arc with the given key has a cached document.
// It returns a stable boolean, so an editor that drives its document imperatively
// re-renders only when the document first becomes available, not on every edit.
export const useHasText = Scope.bindHook(
  createSelector(({ text }) => text.doc != null),
);

export const useName = Scope.bindHook(createSelector(({ name }) => name));

export interface AddNodeProps {
  key: string;
  type: string;
  position?: xy.Crude;
}

// useAddNode returns a callback that appends a node of the given function type at
// the given position, seeding its config from the type's default props. The Arc key
// is resolved from the surrounding scope unless overridden.
export const useAddNode = (keyOverride?: arc.Key) => {
  const key = Scope.use(keyOverride);
  const theme = Theming.use();
  const { dispatch } = useDispatch();
  return useCallback(
    ({ key: nodeKey, type, position }: AddNodeProps) => {
      const spec = (Node.REGISTRY as Record<string, Node.Spec>)[type];
      if (spec == null) return;
      dispatch({
        key,
        actions: [
          arc.setNode({
            node: { key: nodeKey, position: xy.construct(position ?? xy.ZERO) },
          }),
          arc.setNodeInputs({
            key: nodeKey,
            inputs: spec.defaultConfig(theme) as record.Unknown,
          }),
        ],
      });
    },
    [key, dispatch, theme],
  );
};

export type ListQuery = List.PagerParams & {
  keys?: arc.Key[];
};

export const useList = Flux.createList<ListQuery, arc.Key, arc.Arc>({
  name: PLURAL_RESOURCE_NAME,
  normalizeQuery: (query) => ({ ...query, includeStatus: true }),
  retrieve: async ({ client, query }) => await client.arcs.retrieve(query),
  retrieveByKey: async ({ client, key }) => await client.arcs.retrieve(key),
  onChange: ({ client, query }, handler) => client.arcs.onChange(query, handler),
  getCached: ({ client, query }) => client.arcs.getCached(query),
});

export const { useUpdate: useDelete } = Flux.createUpdate<arc.Key | arc.Key[]>({
  name: PLURAL_RESOURCE_NAME,
  verbs: verbs.DELETE,
  update: async ({ client, data, onOptimisticComplete }) => {
    await client.arcs.delete(data, {
      onOptimistic: async () => await onOptimisticComplete(data),
    });
    return data;
  },
});

export const formSchema = z.object({
  key: arc.keyZ.optional(),
  name: z.string().min(1, "Name is required"),
  mode: arc.modeZ,
});

export const ZERO_FORM_VALUES: z.infer<typeof formSchema> = {
  name: "",
  mode: "text",
};

export type FormQuery = Record<string, never>;

export const useForm = Flux.createForm<FormQuery, typeof formSchema>({
  name: RESOURCE_NAME,
  schema: formSchema,
  initialValues: ZERO_FORM_VALUES,
  update: async ({ client, value, set }) => {
    const res = await client.arcs.create(value());
    set("key", res.key);
  },
});

export interface CreateParams extends arc.New {}

export const { useUpdate: useCreate } = Flux.createUpdate<CreateParams, arc.Arc>({
  name: RESOURCE_NAME,
  verbs: verbs.CREATE,
  update: async ({ client, data, onOptimisticComplete }) =>
    await client.arcs.create(data, {
      onOptimistic: async ([optimistic]) => await onOptimisticComplete(optimistic),
    }),
});

export interface RenameParams extends Pick<arc.Arc, "key" | "name"> {}

export const { useUpdate: useRename } = Flux.createUpdate<RenameParams>({
  name: RESOURCE_NAME,
  verbs: verbs.RENAME,
  update: async ({ client, data, onOptimisticComplete }) => {
    const { key, name } = data;
    await client.arcs.rename(key, name, {
      onOptimistic: async () => await onOptimisticComplete(data),
    });
    return data;
  },
});

export interface SetRackParams {
  key: arc.Key;
  /** Target rack. Zero clears the binding, deleting the arc's task. */
  rack: rack.Key;
}

export const { useUpdate: useSetRack } = Flux.createUpdate<
  SetRackParams,
  task.Task | null
>({
  name: `${RESOURCE_NAME} rack`,
  verbs: verbs.SET,
  update: async ({ client, data }) => await client.arcs.setRack(data.key, data.rack),
});

export type RetrieveTaskParams = {
  arcKey: arc.Key;
};

export const {
  use: useTask,
  useResult: useResultTask,
  createResultSelector: createTaskResultSelector,
} = Flux.createRetrieve<RetrieveTaskParams, task.Task | null>({
  name: "task",
  retrieve: async ({ client, query }) => await client.arcs.task.retrieve(query.arcKey),
  onChange: ({ client, query }, handler) =>
    client.arcs.task.onChange(query.arcKey, handler),
  getCached: ({ client, query }) => client.arcs.task.getCached(query.arcKey),
});

const useDriftedResult = createTaskResultSelector(
  (tsk) => tsk != null && task.drifted(tsk),
);

/**
 * Whether the arc's running instance was deployed from different content, config, or
 * rack than its task now holds. The Core rewrites the task config whenever the arc's
 * semantic content changes, so content drift surfaces as ordinary task config drift.
 * Arcs that are not running never drift. Re-renders only when the boolean changes.
 */
export const useDrifted = (query: RetrieveTaskParams): boolean =>
  useDriftedResult(query).data === true;

export interface UseTaskControlsReturn {
  running: boolean;
  taskKey: task.Key;
  taskRack: rack.Key;
  /** Starts the task. The driver rebuilds from the current config when it drifted. */
  onStart: () => void;
  /** Stops the running instance. */
  onStop: () => void;
  onStartStop: () => void;
  taskStatus: status.Status;
}

const notDeployedYet = (name: string) =>
  status.create({ name, variant: "disabled", message: "Not deployed yet" });

/** Running state and start/stop controls for the arc's task. */
export const useTaskControls = (key: arc.Key, name: string): UseTaskControlsReturn => {
  const { data: tsk, variant, status: readStatus } = useResultTask({ arcKey: key });
  const cmd = Task.useCommand();
  const isRunning = tsk?.status?.details.running ?? false;
  const taskKey = tsk?.key ?? "";
  const taskRack = tsk?.rack ?? 0;
  const exec = useCallback(
    (type: "start" | "stop") => {
      if (taskKey === "") return;
      cmd.update([{ task: taskKey, type }]);
    },
    [cmd, taskKey],
  );
  const onStart = useCallback(() => exec("start"), [exec]);
  const onStop = useCallback(() => exec("stop"), [exec]);
  const onStartStop = useCallback(
    () => (isRunning ? onStop() : onStart()),
    [isRunning, onStop, onStart],
  );
  return {
    running: isRunning,
    taskKey,
    taskRack,
    onStart,
    onStop,
    onStartStop,
    taskStatus:
      variant !== "success" ? readStatus : (tsk?.status ?? notDeployedYet(name)),
  };
};
