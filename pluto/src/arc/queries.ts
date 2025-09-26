// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc } from "@synnaxlabs/client";
import { primitive } from "@synnaxlabs/x";
import z from "zod";

import { Flux } from "@/flux";
import { type List } from "@/list";

export interface FluxStore extends Flux.UnaryStore<arc.Key, arc.Arc> {}

export const FLUX_STORE_KEY = "arcs";
const RESOURCE_NAME = "Arc";
const PLURAL_RESOURCE_NAME = "Arcs";

export interface FluxSubStore extends Flux.Store {
  [FLUX_STORE_KEY]: FluxStore;
}

export const FLUX_STORE_CONFIG: Flux.UnaryStoreConfig<FluxSubStore, arc.Key, arc.Arc> =
  {
    listeners: [],
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
  if ("key" in query) {
    const cached = store.arcs.get(query.key);
    if (cached != null) {
      const status = await store.statuses.get(cached.key);
      if (status != null) cached.status = status;
      return cached;
    }
    const arc = await client.arcs.retrieve(query);
    store.arcs.set(query.key, arc);
    return arc;
  }
  const arc = await client.arcs.retrieve(query);
  store.arcs.set(arc.key, arc);
  return arc;
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
  retrieve: async ({ client, query }) => await client.arcs.retrieve(query),
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
    await client.arcs.delete(data);
    rollbacks.push(store.arcs.delete(data));
    return data;
  },
});

export const formSchema = arc.newZ.extend({
  name: z.string().min(1, "Name must not be empty"),
});

export const ZERO_FORM_VALUES: z.infer<typeof formSchema> = {
  name: "",
  graph: { nodes: [], edges: [] },
  text: { raw: "" },
  deploy: true,
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

export const { useUpdate: useCreate } = Flux.createUpdate<arc.New, FluxSubStore>({
  name: RESOURCE_NAME,
  verbs: Flux.CREATE_VERBS,
  update: async ({ client, data, store, rollbacks }) => {
    const arc = await client.arcs.create(data);
    rollbacks.push(store.arcs.set(arc));
    return data;
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
