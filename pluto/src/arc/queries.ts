// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc, NotFoundError, ontology, type task } from "@synnaxlabs/client";
import { primitive } from "@synnaxlabs/x";
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

export interface FluxSubStore extends Status.FluxSubStore, Task.FluxSubStore{
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

export const { useUpdate: useCreate } = Flux.createUpdate<
  arc.New,
  FluxSubStore,
  arc.Arc
>({
  name: RESOURCE_NAME,
  verbs: Flux.CREATE_VERBS,
  update: async ({ client, data, store, rollbacks }) => {
    const arc = await client.arcs.create(data);
    try {
      const task = await client.tasks.retrieve({ name: arc.key });
      await client.tasks.create({
        ...task.payload,
        config: {
          arcKey: arc.key,
        },
      });
    } catch (error) {
      if (NotFoundError.matches(error)) {
        const rack = await client.racks.retrieve({ key: 65538 });
        await rack.createTask({
          name: arc.key,
          type: "arc",
          config: {
            arcKey: arc.key,
          },
        });
      }
    }

    rollbacks.push(store.arcs.set(arc));
    return arc;
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
  update: async ({ client, store, data: { key, name }, rollbacks }) => {
    const arc = await retrieveSingle({ client, store, query: { key } });
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

export const {useRetrieve: useRetrieveTask} = Flux.createRetrieve<RetrieveTaskParams, task.Task | undefined, FluxSubStore>({
  name: "Task",
  retrieve: async ({client, query, store}) => {
    const cachedChild = store.relationships.get((r) => ontology.matchRelationship(r, {
      from: arc.ontologyID(query.arcKey),
      type: "child",
      to: { type: "task" },
    }))[0];
    let taskKey = cachedChild?.to.key;
    if (taskKey == null) {
      const children = await client.ontology.retrieveChildren(arc.ontologyID(query.arcKey), {
          types: ["task"]
      });
      children.forEach((c) => {
        const rel = {
          from: arc.ontologyID(query.arcKey),
          type: "child",
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
      query: { key: cachedChild.to.key },
    });
  },
  mountListeners: ({store, query, onChange}) => {
    if (!("arcKey" in query) || primitive.isZero(query.arcKey)) return [];
    return [store.tasks.onSet(onChange, query.arcKey)];
  },
});