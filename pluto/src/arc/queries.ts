// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  arc,
  NotFoundError,
  ontology,
  type rack,
  status,
  task,
} from "@synnaxlabs/client";
import { errors, id, primitive, type record, TimeStamp, xy } from "@synnaxlabs/x";
import { useCallback } from "react";
import z from "zod";

import { Node } from "@/arc/graph/node";
import { Flux } from "@/flux";
import { useSyncedRef } from "@/hooks/ref";
import { type List } from "@/list";
import { state } from "@/state";
import { type Status } from "@/status";
import { Task } from "@/task";
import { Theming } from "@/theming";
import { type Diagram } from "@/vis/diagram";

const edgesToDiagram = (edges: arc.ir.Edge[]): Diagram.Edge[] =>
  edges.map((e) => ({
    key: arc.ir.edgeKey(e.source, e.target),
    source: e.source,
    target: e.target,
  }));

export interface FluxStore extends Flux.UndoableUnaryStore<
  arc.Key,
  arc.Arc,
  arc.Action
> {}

export const FLUX_STORE_KEY = "arcs";
const RESOURCE_NAME = "Arc";
const PLURAL_RESOURCE_NAME = "Arcs";

export interface FluxSubStore extends Status.FluxSubStore, Task.FluxSubStore {
  [FLUX_STORE_KEY]: FluxStore;
}

// kindOfTransaction classifies an action batch for the undo coalesce window. A
// node drag dispatches a stream of set_node_position actions for a single
// gesture; classifying them all as "move" collapses them into one undoable.
const kindOfTransaction = (actions: arc.Action[]): string => {
  if (actions.length === 0) return "default";
  const hasMove = actions.some((a) => a.type === "set_node_position");
  const onlyMove = actions.every((a) => a.type === "set_node_position");
  if (hasMove && onlyMove) return "move";
  if (actions.length === 1) return actions[0].type;
  return "transaction";
};

const undoableStoreConfig = Flux.createUndoableStore<
  arc.Key,
  arc.Arc,
  arc.Action,
  typeof FLUX_STORE_KEY,
  FluxSubStore
>({
  storeKey: FLUX_STORE_KEY,
  reduce: arc.reduceAll,
  channel: arc.SET_CHANNEL_NAME,
  schema: arc.scopedActionZ,
  isUndoable: arc.isUndoable,
  kindOf: kindOfTransaction,
});

const DELETE_ARC_LISTENER: Flux.ChannelListener<FluxSubStore, typeof arc.keyZ> = {
  channel: arc.DELETE_CHANNEL_NAME,
  schema: arc.keyZ,
  onChange: ({ store, changed }) => store.arcs.delete(changed),
};

export const FLUX_STORE_CONFIG: Flux.UnaryStoreConfig<FluxSubStore> = {
  ...undoableStoreConfig,
  listeners: [...undoableStoreConfig.listeners, DELETE_ARC_LISTENER],
};

export const { useDispatch, useUndo, useRedo } = Flux.createDispatch<
  arc.Key,
  arc.Arc,
  arc.Action,
  typeof FLUX_STORE_KEY,
  FluxSubStore
>({
  storeKey: FLUX_STORE_KEY,
  send: ({ client, key, actions, dispatchKey }) =>
    client.arcs.dispatch(key, dispatchKey, actions),
});

export interface SelectKeyArgs {
  key: arc.Key;
}

const requireArc = (store: FluxSubStore, key: arc.Key): arc.Arc => {
  const a = store.arcs.get(key);
  if (a == null) throw new NotFoundError(`Arc with key ${key} not found`);
  return a;
};

// useSelectNodes returns the graph nodes of the Arc with the given key as diagram
// nodes. graph.Node is a structural superset of Diagram.Node, so the stored array
// is returned by reference with no translation, keeping selections referentially
// stable across unrelated store updates.
export const useSelectNodes = Flux.createSelector<
  FluxSubStore,
  SelectKeyArgs,
  Diagram.Node[]
>({
  subscribe: (store, { key }, notify) => store.arcs.onSet(notify, key),
  select: (store, { key }) => requireArc(store, key).graph.nodes,
});

// useSelectEdges returns the graph edges of the Arc with the given key as keyed
// diagram edges. select returns the stored ir.Edge array by reference; transform
// derives the keyed diagram edges and is memoized on that reference, so it only
// re-runs when the edges actually change.
export const useSelectEdges = Flux.createSelector<
  FluxSubStore,
  SelectKeyArgs,
  Diagram.Edge[],
  arc.ir.Edge[]
>({
  subscribe: (store, { key }, notify) => store.arcs.onSet(notify, key),
  select: (store, { key }) => requireArc(store, key).graph.edges,
  transform: edgesToDiagram,
});

export interface SelectNodePropsArgs {
  key: arc.Key;
  nodeKey: string;
}

// useSelectNodeConfig returns the typed config for a single graph node. Returned by
// reference, so the selection only re-runs when that node's config changes.
export const useSelectNodeConfig = Flux.createSelector<
  FluxSubStore,
  SelectNodePropsArgs,
  Node.Config
>({
  subscribe: (store, { key }, notify) => store.arcs.onSet(notify, key),
  select: (store, { key, nodeKey }) =>
    requireArc(store, key).graph.configs[nodeKey] as Node.Config,
});

// useSelectMode returns the representation mode of the Arc with the given key,
// or undefined when it has not yet loaded into the store.
export const useSelectMode = Flux.createSelector<FluxSubStore, SelectKeyArgs, arc.Mode>(
  {
    subscribe: (store, { key }, notify) => store.arcs.onSet(notify, key),
    select: (store, { key }) => requireArc(store, key).mode,
  },
);

// useSelectHasText reports whether the Arc with the given key has loaded into the store.
// It returns a stable boolean, so an editor that drives its document imperatively re-renders
// only when the document first becomes available, not on every subsequent edit.
export const useSelectHasText = Flux.createSelector<
  FluxSubStore,
  SelectKeyArgs,
  boolean
>({
  subscribe: (store, { key }, notify) => store.arcs.onSet(notify, key),
  select: (store, { key }) => store.arcs.get(key)?.text.doc != null,
});

export interface AddNodeProps {
  key: string;
  type: string;
  position?: xy.Crude;
}

// useAddNode returns a callback that appends a node of the given function type at
// the given position, seeding its config from the type's default props.
export const useAddNode = (key: arc.Key) => {
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
          arc.setNodeConfig({
            key: nodeKey,
            config: spec.defaultConfig(theme) as record.Unknown,
          }),
        ],
      });
    },
    [key, dispatch, theme],
  );
};

export interface FluxSubStore extends Flux.Store {
  [FLUX_STORE_KEY]: FluxStore;
}

export type RetrieveQuery = {
  key: arc.Key;
  includeStatus?: boolean;
};

const retrieveSingle = async ({
  client,
  query,
  store,
}: Flux.RetrieveParams<RetrieveQuery, FluxSubStore>) => {
  const a = await client.arcs.retrieve({
    ...query,
    includeStatus: query.includeStatus ?? true,
  });
  store.arcs.set(query.key, a);
  return a;
};

export type ListQuery = List.PagerParams & {
  keys?: arc.Key[];
};

export const useList = Flux.createList<ListQuery, arc.Key, arc.Arc, FluxSubStore>({
  name: PLURAL_RESOURCE_NAME,
  retrieveCached: ({ store, query }) =>
    store.arcs.get((a) => {
      if (primitive.isNonZero(query.keys)) return query.keys.includes(a.key);
      return true;
    }),
  retrieve: async ({ client, query }) =>
    await client.arcs.retrieve({
      ...query,
      includeStatus: true,
    }),
  retrieveByKey: async ({ client, key, store }) => {
    const cached = store.arcs.get(key);
    if (cached != null) return cached;
    const arc = await client.arcs.retrieve({ key });
    store.arcs.set(key, arc);
    return arc;
  },
  mountListeners: ({ store, onChange, onDelete }) => [
    store.arcs.onSet((arc) => onChange(arc.key, arc)),
    store.arcs.onDelete(onDelete),
  ],
});

export const { useUpdate: useDelete } = Flux.createUpdate<
  arc.Key | arc.Key[],
  FluxSubStore
>({
  name: PLURAL_RESOURCE_NAME,
  verbs: Flux.DELETE_VERBS,
  update: async ({ client, data, store, rollbacks }) => {
    rollbacks.push(store.arcs.delete(data));
    await client.arcs.delete(data);
    return data;
  },
});

export const formSchema = arc.arcZ
  .partial({ key: true, name: true })
  .extend({ name: z.string() });

export const ZERO_FORM_VALUES: z.infer<typeof formSchema> = formSchema.parse({
  name: "",
  mode: "text",
});

export const useForm = Flux.createForm<
  Partial<RetrieveQuery>,
  typeof formSchema,
  FluxSubStore
>({
  name: RESOURCE_NAME,
  schema: formSchema,
  initialValues: ZERO_FORM_VALUES,
  retrieve: async ({ client, query, reset, store }) => {
    if (!("key" in query) || primitive.isZero(query.key)) return;
    reset(await retrieveSingle({ client, query: query as RetrieveQuery, store }));
  },
  update: async ({ client, value, reset, store, rollbacks }) => {
    const updated = await client.arcs.create(value());
    reset(updated);
    rollbacks.push(store.arcs.set(updated.key, updated));
  },
});

export interface CreateParams extends arc.New {
  rack?: rack.Key;
}

const TASK_TYPE = "arc";

const taskStatusDataZ = z.null().optional();

const configuringStatus = (taskKey: task.Key): task.Status<typeof taskStatusDataZ> =>
  status.create<ReturnType<typeof task.statusDetailsZ<typeof taskStatusDataZ>>>({
    key: task.statusKey(taskKey),
    name: "Configuring task",
    variant: "loading",
    message: "Configuring task...",
    details: { task: taskKey, running: false, cmd: "", data: undefined },
  });

const TASK_SCHEMAS = {
  type: z.literal(TASK_TYPE),
  config: z.object({ arcKey: z.string() }),
  statusData: taskStatusDataZ,
} as const satisfies task.Schemas;

export const { useUpdate: useCreate } = Flux.createUpdate<
  CreateParams,
  FluxSubStore,
  arc.Arc
>({
  name: RESOURCE_NAME,
  verbs: Flux.CREATE_VERBS,
  update: async ({ client, data, store, rollbacks }) => {
    const { rack } = data;
    const optimistic: arc.Arc = arc.arcZ.parse(data);
    let taskKey: task.Key | undefined;
    // If the caller selected a rack to deploy the arc on, we need to create a task
    // for it.
    if (rack != null) {
      taskKey = task.newKey(rack, 0);
      if (data.key != null) {
        const tsk = await retrieveTask({ client, store, query: { arcKey: data.key } });
        if (tsk != null)
          if (task.rackKey(tsk.key) != rack) {
            // This means a previous task was created for a different rack, and we need
            // to delete it.
            rollbacks.push(store.tasks.delete(tsk.key));
            rollbacks.push(
              store.relationships.delete(
                ontology.relationshipToString({
                  from: arc.ontologyID(data.key),
                  type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
                  to: task.ontologyID(tsk.key),
                }),
              ),
            );
            await client.tasks.delete([tsk.key]);
          } else taskKey = tsk.key;
      }
    }
    rollbacks.push(store.arcs.set(optimistic));
    const prog = await client.arcs.create(optimistic);
    if (taskKey == null) return prog;
    const { key, name } = prog;
    const newTsk = await client.tasks.create(
      {
        key: taskKey,
        name,
        type: TASK_TYPE,
        config: { arcKey: key },
        status: configuringStatus(taskKey),
      },
      TASK_SCHEMAS,
    );
    await client.ontology.addChildren(arc.ontologyID(key), task.ontologyID(newTsk.key));
    return prog;
  },
});

export const { useRetrieve, useRetrieveObservable, useEnsureRetrieved } =
  Flux.createRetrieve<RetrieveQuery, arc.Arc, FluxSubStore>({
    name: RESOURCE_NAME,
    retrieve: retrieveSingle,
    mountListeners: ({ store, query, onChange }) => {
      if (!("key" in query) || primitive.isZero(query.key)) return [];
      return [store.arcs.onSet(onChange, query.key)];
    },
  });

export const useRetrieveObservableName = ({
  onChange,
  ...params
}: Omit<Flux.UseRetrieveObservableParams<RetrieveQuery, arc.Arc>, "onChange"> & {
  onChange: (name: string) => void;
}): Flux.UseRetrieveObservableReturn<RetrieveQuery> => {
  const onChangeRef = useSyncedRef(onChange);
  return useRetrieveObservable({
    ...params,
    onChange: useCallback((result) => {
      if (result.variant !== "success") return;
      onChangeRef.current(result.data.name);
    }, []),
  });
};

export interface RenameParams extends Pick<arc.Arc, "key" | "name"> {}

export const { useUpdate: useRename } = Flux.createUpdate<RenameParams, FluxSubStore>({
  name: RESOURCE_NAME,
  verbs: Flux.RENAME_VERBS,
  update: async (params) => {
    const {
      client,
      store,
      data: { key, name },
      rollbacks,
    } = params;
    const task = await retrieveTask({ client, store, query: { arcKey: key } });
    if (task != null) await Task.rename({ ...params, data: { key: task.key, name } });

    rollbacks.push(
      store.arcs.set(
        key,
        state.skipUndefined((p) => ({ ...p, name })),
      ),
    );
    await client.arcs.dispatch(key, id.create(), [arc.rename({ name })]);
    return { key, name };
  },
});

export type RetrieveTaskParams = {
  arcKey: arc.Key;
};

export const retrieveTask = async ({
  client,
  query,
  store,
}: Flux.RetrieveParams<RetrieveTaskParams, FluxSubStore>): Promise<
  task.Task | undefined
> => {
  const arcOntologyID = arc.ontologyID(query.arcKey);
  const cachedChild = store.relationships.get((r) =>
    ontology.matchRelationship(r, {
      from: arcOntologyID,
      type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
      to: { type: "task" },
    }),
  )[0];

  let taskKey = cachedChild?.to.key;

  if (taskKey == null) {
    let children: ontology.Resource[];
    try {
      children = await client.ontology.retrieveChildren(arcOntologyID, {
        types: ["task"],
      });
    } catch (e) {
      // if the arc doesn't exist then it can't have a task.
      if (NotFoundError.matches(e)) return undefined;
      throw errors.fromUnknown(e);
    }
    children.forEach((c) => {
      const rel: ontology.Relationship = {
        from: arcOntologyID,
        type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
        to: c.id,
      };
      store.relationships.set(ontology.relationshipToString(rel), rel);
    });
    if (children.length === 0) return undefined;
    taskKey = children[0].id.key;
  }

  return await Task.retrieveSingle({ store, client, query: { key: taskKey } });
};

export const { useRetrieve: useRetrieveTask } = Flux.createRetrieve<
  RetrieveTaskParams,
  task.Task | undefined,
  FluxSubStore
>({
  name: "Task",
  retrieve: retrieveTask,
  mountListeners: ({ store, query, onChange, client }) => {
    if (!("arcKey" in query) || primitive.isZero(query.arcKey)) return [];
    const arcOntologyID = arc.ontologyID(query.arcKey);

    return [
      store.relationships.onSet(async (rel) => {
        const isTaskChild = ontology.matchRelationship(rel, {
          from: arcOntologyID,
          type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
          to: { type: "task" },
        });
        if (!isTaskChild) return;
        const tsk = await Task.retrieveSingle({
          store,
          client,
          query: { key: rel.to.key },
        });
        onChange(tsk);
      }),

      store.relationships.onDelete(async (relKey) => {
        const rel = ontology.relationshipZ.parse(relKey);
        const isTaskChild = ontology.matchRelationship(rel, {
          from: arcOntologyID,
          type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
          to: { type: "task" },
        });
        if (!isTaskChild) return;
        onChange(undefined);
      }),

      store.tasks.onSet(async (tsk) => {
        const isChild =
          store.relationships.get((r) =>
            ontology.matchRelationship(r, {
              from: arcOntologyID,
              type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
              to: task.ontologyID(tsk.key),
            }),
          ).length > 0;
        if (isChild)
          onChange((prev) => {
            if (prev == null) return tsk;
            return client.tasks.sugar({ ...tsk.payload, status: prev.status });
          });
      }),

      store.statuses.onSet(async (status) => {
        if (!status.key.startsWith("task")) return;
        const cachedRel = store.relationships.get((r) =>
          ontology.matchRelationship(r, {
            from: arcOntologyID,
            type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
            to: { type: "task" },
          }),
        )[0];
        if (cachedRel == null) return;
        const taskStatusKey = task.statusKey(cachedRel.to.key);
        if (status.key !== taskStatusKey) return;
        const parsed = task.statusZ(z.unknown().optional()).safeParse(status);
        if (!parsed.success) return;
        onChange((prev) => {
          if (prev == null) return prev;
          return client.tasks.sugar({ ...prev.payload, status: parsed.data });
        });
      }),
    ];
  },
});
