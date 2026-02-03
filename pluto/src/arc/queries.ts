// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc, ontology, type rack, task } from "@synnaxlabs/client";
import { primitive, status } from "@synnaxlabs/x";
import z from "zod";

import { Flux } from "@/flux";
import { type List } from "@/list";
import { state } from "@/state";
import { type Status } from "@/status";
import { Task } from "@/task";

export interface FluxStore extends Flux.UnaryStore<arc.Key, arc.Arc> {}

export const FLUX_STORE_KEY = "arcs";
const RESOURCE_NAME = "Arc";
const PLURAL_RESOURCE_NAME = "Arcs";

export interface FluxSubStore extends Status.FluxSubStore, Task.FluxSubStore {
  [FLUX_STORE_KEY]: FluxStore;
}

const SET_ARC_LISTENER: Flux.ChannelListener<FluxSubStore, typeof arc.arcZ> = {
  channel: arc.SET_CHANNEL_NAME,
  schema: arc.arcZ,
  onChange: ({ store, changed }) => store.arcs.set(changed.key, changed),
};

const DELETE_ARC_LISTENER: Flux.ChannelListener<FluxSubStore, typeof arc.keyZ> = {
  channel: arc.DELETE_CHANNEL_NAME,
  schema: arc.keyZ,
  onChange: ({ store, changed }) => store.arcs.delete(changed),
};

export const FLUX_STORE_CONFIG: Flux.UnaryStoreConfig<FluxSubStore, arc.Key, arc.Arc> =
  {
    listeners: [SET_ARC_LISTENER, DELETE_ARC_LISTENER],
  };

export interface FluxSubStore extends Flux.Store {
  [FLUX_STORE_KEY]: FluxStore;
}

export interface RetrieveQuery {
  key: arc.Key;
  includeStatus?: boolean;
}

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

export interface ListQuery extends List.PagerParams {
  keys?: arc.Key[];
}

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

export const formSchema = arc.newZ.extend({
  name: z.string().min(1, "Name must not be empty"),
});

export const ZERO_FORM_VALUES: z.infer<typeof formSchema> = {
  name: "",
  version: "0.0.0",
  graph: { nodes: [], edges: [] },
  text: { raw: "" },
};

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

export const taskStatusDataZ = z.null().or(z.undefined());
export type TaskStatusData = z.infer<typeof taskStatusDataZ>;
const TASK_TYPE = "arc";
export const taskTypeZ = z.literal(TASK_TYPE);
export type TaskType = z.infer<typeof taskTypeZ>;
export const taskConfigZ = z.object({
  arcKey: z.string(),
});

export interface TaskConfig extends z.infer<typeof taskConfigZ> {}

const configuringStatus = (taskKey: task.Key): task.Status<typeof taskStatusDataZ> =>
  status.create<ReturnType<typeof task.statusDetailsZ<typeof taskStatusDataZ>>>({
    key: task.statusKey(taskKey),
    name: "Configuring task",
    variant: "loading",
    message: "Configuring task...",
    details: {
      task: taskKey,
      running: false,
      data: undefined,
    },
  });

const TASK_SCHEMAS: task.Schemas<
  typeof taskTypeZ,
  typeof taskConfigZ,
  typeof taskStatusDataZ
> = {
  typeSchema: taskTypeZ,
  configSchema: taskConfigZ,
  statusDataSchema: taskStatusDataZ,
};

export const { useUpdate: useCreate } = Flux.createUpdate<
  CreateParams,
  FluxSubStore,
  arc.Arc
>({
  name: RESOURCE_NAME,
  verbs: Flux.CREATE_VERBS,
  update: async ({ client, data, store, rollbacks }) => {
    const { rack } = data;
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
    const prog = await client.arcs.create(data);
    rollbacks.push(store.arcs.set(prog));
    const { key, name } = prog;
    if (taskKey == null) return prog;
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

export const { useRetrieve, useRetrieveObservable } = Flux.createRetrieve<
  RetrieveQuery,
  arc.Arc,
  FluxSubStore
>({
  name: RESOURCE_NAME,
  retrieve: retrieveSingle,
  mountListeners: ({ store, query, onChange }) => {
    if (!("key" in query) || primitive.isZero(query.key)) return [];
    return [store.arcs.onSet(onChange, query.key)];
  },
});

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
    const arc = await retrieveSingle({ client, store, query: { key } });
    const task = await retrieveTask({ client, store, query: { arcKey: key } });
    if (task != null) await Task.rename({ ...params, data: { key: task.key, name } });

    rollbacks.push(
      store.arcs.set(
        key,
        state.skipUndefined((p) => ({ ...p, name })),
      ),
    );
    await client.arcs.create({ ...arc, name });
    return { key, name };
  },
});

export interface RetrieveTaskParams {
  arcKey: arc.Key;
}

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
    const children = await client.ontology.retrieveChildren(arcOntologyID, {
      types: ["task"],
    });
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

  return await Task.retrieveSingle({
    store,
    client,
    query: { key: taskKey },
  });
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
            if (prev == null) return tsk as task.Task;
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
        const parsed = task.statusZ(z.null().or(z.undefined())).safeParse(status);
        if (!parsed.success) return;
        onChange((prev) => {
          if (prev == null) return prev;
          return client.tasks.sugar({ ...prev.payload, status: parsed.data });
        });
      }),
    ];
  },
});
