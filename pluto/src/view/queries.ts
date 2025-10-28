// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { view } from "@synnaxlabs/client";
import type z from "zod";

import { Flux } from "@/flux";

export const FLUX_STORE_KEY = "views";
export const RESOURCE_NAME = "view";
export const PLURAL_RESOURCE_NAME = "views";

export interface FluxStore extends Flux.UnaryStore<view.Key, view.View> {}

export interface FluxSubStore extends Flux.Store {
  [FLUX_STORE_KEY]: FluxStore;
}

const SET_VIEW_LISTENER: Flux.ChannelListener<FluxSubStore, typeof view.viewZ> = {
  channel: view.SET_CHANNEL_NAME,
  schema: view.viewZ,
  onChange: ({ store, changed }) => store.views.set(changed.key, changed),
};

const DELETE_VIEW_LISTENER: Flux.ChannelListener<FluxSubStore, typeof view.keyZ> = {
  channel: view.DELETE_CHANNEL_NAME,
  schema: view.keyZ,
  onChange: ({ store, changed }) => store.views.delete(changed),
};

export const FLUX_STORE_CONFIG: Flux.UnaryStoreConfig<FluxSubStore> = {
  listeners: [SET_VIEW_LISTENER, DELETE_VIEW_LISTENER],
};

export interface RetrieveQuery extends view.RetrieveSingleParams {}

export const retrieveSingle = async ({
  store,
  query,
  client,
}: Flux.RetrieveParams<RetrieveQuery, FluxSubStore>) => {
  const cached = store.views.get(query.key);
  if (cached != null) return cached;
  const view = await client.views.retrieve(query);
  store.views.set(view.key, view);
  return view;
};

export interface ListQuery extends view.RetrieveMultipleParams {}

export const useList = Flux.createList<ListQuery, view.Key, view.View, FluxSubStore>({
  name: PLURAL_RESOURCE_NAME,
  retrieveCached: ({ query, store }) => {
    const { types = [], keys = [] } = query;
    const typesSet = types.length > 0 ? new Set(types) : undefined;
    const keysSet = keys.length > 0 ? new Set(keys) : undefined;
    return store.views.get((v) => {
      if (typesSet != null && !typesSet.has(v.type)) return false;
      if (keysSet != null && !keysSet.has(v.key)) return false;
      return true;
    });
  },
  retrieve: async ({ client, query }) => await client.views.retrieve(query),
  retrieveByKey: async ({ client, key }) => await client.views.retrieve({ key }),
  mountListeners: ({ store, onChange, onDelete, query: { keys, types } }) => {
    const keysSet = keys ? new Set(keys) : undefined;
    const typesSet = types ? new Set(types) : undefined;
    return [
      store.views.onSet((view) => {
        if (
          (keysSet != null && !keysSet.has(view.key)) ||
          (typesSet != null && !typesSet.has(view.type))
        )
          return;
        onChange(view.key, view, { mode: "prepend" });
      }),
      store.views.onDelete(onDelete),
    ];
  },
});

interface FormQuery {
  key?: view.Key;
}

export const formSchema = view.viewZ.partial({ key: true });

const INITIAL_VALUES: z.infer<typeof formSchema> = {
  name: "",
  type: "",
  query: {},
};

export const useForm = Flux.createForm<FormQuery, typeof formSchema, FluxSubStore>({
  name: RESOURCE_NAME,
  initialValues: INITIAL_VALUES,
  schema: formSchema,
  retrieve: async ({ client, query: { key }, reset }) => {
    if (key == null) return;
    reset(await client.views.retrieve({ key }));
  },
  update: async ({ client, value, reset }) => {
    const updated = await client.views.create(value());
    reset(updated);
  },
  mountListeners: ({ store, query: { key }, reset }) => [
    store.views.onSet(async (view) => {
      if (key == null || view.key !== key) return;
      reset(view);
    }, key),
  ],
});

export type DeleteParams = view.Key | view.Key[];

export const { useUpdate: useDelete } = Flux.createUpdate<DeleteParams, FluxSubStore>({
  name: RESOURCE_NAME,
  verbs: Flux.DELETE_VERBS,
  update: async ({ client, data }) => {
    await client.views.delete(data);
    return data;
  },
});

export interface RetrieveMultipleParams {
  keys: view.Key[];
}

export const { useRetrieve: useRetrieveMultiple } = Flux.createRetrieve<
  RetrieveMultipleParams,
  view.View[],
  FluxSubStore
>({
  name: PLURAL_RESOURCE_NAME,
  retrieve: async ({ client, query: { keys }, store }) => {
    const cached = store.views.get((v) => keys.includes(v.key));
    const missing = keys.filter((k) => !store.views.has(k));
    if (missing.length === 0) return cached;
    const retrieved = await client.views.retrieve({ keys: missing });
    store.views.set(retrieved);
    return [...cached, ...retrieved];
  },
  mountListeners: ({ store, query: { keys }, onChange }) => {
    const keysSet = new Set(keys);
    return [
      store.views.onSet(async (view) => {
        if (!keysSet.has(view.key)) return;
        onChange((prev) => {
          if (prev == null) return [view];
          return [...prev.filter((v) => v.key !== view.key), view];
        });
      }),
      store.views.onDelete(async (key) => {
        keysSet.delete(key);
        onChange((prev) => (prev == null ? [] : prev.filter((v) => v.key !== key)));
      }),
    ];
  },
});
