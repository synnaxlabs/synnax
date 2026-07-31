// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc, NotFoundError, query, type Synnax, type task } from "@synnaxlabs/client";
import { compare, type optional, primitive, type record, xy } from "@synnaxlabs/x";
import { useCallback } from "react";
import z from "zod";

import { Node } from "@/arc/graph/node";
import { Scope } from "@/arc/scope";
import { Flux } from "@/flux";
import { type List } from "@/list";
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

export interface SelectKeyParams {
  key: arc.Key;
}

const requireArc = (client: Synnax | null, key: arc.Key): arc.Arc => {
  const cached = client?.arcs.getCached({ key });
  if (cached == null) throw new NotFoundError(`Arc with key ${key} not found`);
  if (query.Deleted.matches(cached))
    throw new Flux.DeletedError(`${RESOURCE_NAME} was deleted`, cached.corpse);
  return cached;
};

const getArc = (client: Synnax | null, key: arc.Key): arc.Arc | undefined => {
  const cached = client?.arcs.getCached({ key });
  if (!query.isLive(cached)) return undefined;
  return cached;
};

const subscribe = (
  { client, args: { key } }: Flux.SelectorParams<SelectKeyParams>,
  notify: () => void,
) => (client == null ? () => {} : client.arcs.onChange({ key }, notify));

// useSelectAllNodes returns every graph node of the Arc with the given key as diagram
// nodes. graph.Node is a structural superset of Diagram.Node, so the cached array
// is returned by reference with no translation, keeping selections referentially
// stable across unrelated cache updates.
export const [useSelectAllNodes, useGetAllNodes] = Scope.bindSelector(
  Flux.createSelector<SelectKeyParams, Diagram.Node[]>({
    subscribe,
    select: ({ client, args: { key } }) => requireArc(client, key).graph.nodes,
  }),
);

export interface SelectNodesParams extends SelectKeyParams {
  keys: string[];
}

// useSelectNodes returns only the graph nodes whose keys are in the given set. The
// result is compared by value, so a consumer that tracks a selection re-renders only
// when its nodes change, not on every node mutation.
export const [useSelectNodes, useGetNodes] = Scope.bindSelector(
  Flux.createSelector<SelectNodesParams, Diagram.Node[]>({
    subscribe,
    select: ({ client, args: { key, keys } }) => {
      const a = getArc(client, key);
      if (a == null || keys.length === 0) return [];
      const keySet = new Set(keys);
      return a.graph.nodes.filter((n) => keySet.has(n.key));
    },
    equal: compare.arraysEqual,
  }),
);

// useSelectAllEdges returns every graph edge of the Arc with the given key as diagram
// edges. graph.Edge is a structural superset of Diagram.Edge, so the cached array
// is returned by reference with no translation, keeping selections referentially
// stable across unrelated cache updates.
export const [useSelectAllEdges, useGetAllEdges] = Scope.bindSelector(
  Flux.createSelector<SelectKeyParams, Diagram.Edge[]>({
    subscribe,
    select: ({ client, args: { key } }) => requireArc(client, key).graph.edges,
  }),
);

export interface SelectNodePropsParams extends SelectKeyParams {
  nodeKey: string;
}

// useSelectNodeConfig returns the typed config for a single graph node. Returned by
// reference, so the selection only re-runs when that node's config changes.
export const [useSelectNodeConfig, useGetNodeConfig] = Scope.bindSelector(
  Flux.createSelector<SelectNodePropsParams, Node.Config>({
    subscribe,
    select: ({ client, args: { key, nodeKey } }) =>
      requireArc(client, key).graph.inputs[nodeKey] as Node.Config,
  }),
);

// useSelectMode returns the representation mode of the Arc with the given key. It
// requires the arc to be cached, so callers must render it beneath an Arc.Suspended
// boundary that has retrieved the arc.
export const [useSelectMode, useGetMode] = Scope.bindSelector(
  Flux.createSelector<SelectKeyParams, arc.Mode>({
    subscribe,
    select: ({ client, args: { key } }) => requireArc(client, key).mode,
  }),
);

// useSelectHasText reports whether the Arc with the given key has a cached document.
// It returns a stable boolean, so an editor that drives its document imperatively
// re-renders only when the document first becomes available, not on every edit.
export const [useSelectHasText, useGetHasText] = Scope.bindSelector(
  Flux.createSelector<SelectKeyParams, boolean>({
    subscribe,
    select: ({ client, args: { key } }) => getArc(client, key)?.text.doc != null,
  }),
);

export const [useSelectName, useGetName] = Scope.bindSelector(
  Flux.createSelector<SelectKeyParams, string>({
    subscribe,
    select: ({ client, args: { key } }) => requireArc(client, key).name,
  }),
);

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

export type RetrieveQuery = {
  key: arc.Key;
  includeStatus?: boolean;
};

export type ListQuery = List.PagerParams & {
  keys?: arc.Key[];
};

export const useList = Flux.createList<ListQuery, arc.Key, arc.Arc>({
  name: PLURAL_RESOURCE_NAME,
  retrieve: async ({ client, query }) =>
    await client.arcs.retrieve({ ...query, includeStatus: true }),
  retrieveByKey: async ({ client, key }) => await client.arcs.retrieve({ key }),
  subscribe: ({ client, query }, handler) =>
    client.arcs.onChange({ ...query, includeStatus: true }, handler),
  getCached: ({ client, query }) =>
    client.arcs.getCached({ ...query, includeStatus: true }),
});

export const { useUpdate: useDelete } = Flux.createUpdate<arc.Key | arc.Key[]>({
  name: PLURAL_RESOURCE_NAME,
  verbs: Flux.DELETE_VERBS,
  update: async ({ client, data, onOptimisticComplete }) => {
    await client.arcs.delete(data, {
      onOptimistic: async () => await onOptimisticComplete(data),
    });
    return data;
  },
});

export type FormValues = optional.Optional<arc.Arc, "key">;

export const formSchema: z.ZodType<FormValues> = arc.arcZ
  .partial({ key: true, name: true })
  .extend({ name: z.string() });

export const ZERO_FORM_VALUES: z.infer<typeof formSchema> = formSchema.parse({
  name: "",
  mode: "text",
});

export const useForm = Flux.createForm<Partial<RetrieveQuery>, typeof formSchema>({
  name: RESOURCE_NAME,
  schema: formSchema,
  initialValues: ZERO_FORM_VALUES,
  retrieve: async ({ client, query: { key, ...rest }, reset }) => {
    if (key == null || primitive.isZero(key)) return;
    // Prefer the cached copy: it may hold locally replayed edits ahead of the
    // server.
    const cached = client.arcs.getCached({ key });
    if (query.isLive(cached)) return reset(cached);
    reset(await client.arcs.retrieve({ key, ...rest }));
  },
  update: async ({ client, value, reset }) => {
    reset(await client.arcs.create(value()));
  },
});

export interface CreateParams extends arc.CreateParams {}

export const { useUpdate: useCreate } = Flux.createUpdate<CreateParams, arc.Arc>({
  name: RESOURCE_NAME,
  verbs: Flux.CREATE_VERBS,
  update: async ({ client, data, onOptimisticComplete }) =>
    await client.arcs.create(data, {
      onOptimistic: async ([optimistic]) => await onOptimisticComplete(optimistic),
    }),
});

export const { useRetrieve, useRetrieveObservable, useEnsureRetrieved } =
  Flux.createRetrieve<RetrieveQuery, arc.Arc>({
    name: RESOURCE_NAME,
    retrieve: async ({ client, query }) => await client.arcs.retrieve(query),
    subscribe: ({ client, query }, handler) => client.arcs.onChange(query, handler),
    getCached: ({ client, query }) => client.arcs.getCached(query),
  });

export interface RenameParams extends Pick<arc.Arc, "key" | "name"> {}

export const { useUpdate: useRename } = Flux.createUpdate<RenameParams>({
  name: RESOURCE_NAME,
  verbs: Flux.RENAME_VERBS,
  update: async ({ client, data, onOptimisticComplete }) => {
    const { key, name } = data;
    await client.arcs.rename(key, name, {
      onOptimistic: async () => await onOptimisticComplete(data),
    });
    return data;
  },
});

export type RetrieveTaskParams = {
  arcKey: arc.Key;
};

export const { useRetrieve: useRetrieveTask } = Flux.createRetrieve<
  RetrieveTaskParams,
  task.Task | null
>({
  name: "Task",
  retrieve: async ({ client, query }) => await client.arcs.task.retrieve(query.arcKey),
  subscribe: ({ client, query }, handler) =>
    client.arcs.task.onChange(query.arcKey, handler),
  getCached: ({ client, query }) => client.arcs.task.getCached(query.arcKey),
});
