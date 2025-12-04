// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { view } from "@synnaxlabs/client";
import { array } from "@synnaxlabs/x";

import { Flux } from "@/flux";
import { Ontology } from "@/ontology";

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
  retrieve: async ({ client, query, store }) => {
    const views = await client.views.retrieve(query);
    store.views.set(views);
    return views;
  },
  retrieveByKey: async ({ client, key, store }) => {
    let v = store.views.get(key);
    if (v == null) {
      v = await client.views.retrieve({ key });
      store.views.set(v);
    }
    return v;
  },
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

export const { useUpdate: useCreate } = Flux.createUpdate<view.New, FluxSubStore>({
  name: RESOURCE_NAME,
  verbs: Flux.CREATE_VERBS,
  update: async ({ client, data, store, rollbacks }) => {
    const views = await client.views.create(data);
    rollbacks.push(store.views.set(views));
    return views;
  },
});

export type DeleteParams = view.Key | view.Key[];

export const { useUpdate: useDelete } = Flux.createUpdate<DeleteParams, FluxSubStore>({
  name: RESOURCE_NAME,
  verbs: Flux.DELETE_VERBS,
  update: async ({ client, data, store, rollbacks }) => {
    const keys = array.toArray(data);
    const ids = keys.map((key) => view.ontologyID(key));
    const relFilter = Ontology.filterRelationshipsThatHaveIDs(ids);
    rollbacks.push(store.relationships.delete(relFilter));
    rollbacks.push(store.views.delete(keys));
    rollbacks.push(store.resources.delete(keys));
    await client.views.delete(data);
    return data;
  },
});

export const { useRetrieve } = Flux.createRetrieve<
  view.RetrieveSingleParams,
  view.View,
  FluxSubStore
>({
  name: RESOURCE_NAME,
  retrieve: async ({ store, client, query: { key } }) => {
    const cached = store.views.get(key);
    if (cached != null) return cached;
    const v = await client.views.retrieve({ key });
    store.views.set(v);
    return v;
  },
  mountListeners: ({ store, query: { key }, onChange }) => [
    store.views.onSet(onChange, key),
  ],
});

export const { useRetrieve: useRetrieveMultiple } = Flux.createRetrieve<
  view.RetrieveMultipleParams,
  view.View[],
  FluxSubStore
>({
  name: PLURAL_RESOURCE_NAME,
  retrieve: async ({ client, query: { keys, types }, store }) => {
    // if keys are provided, we can first check the store for the views and potentially
    // avoid a network call
    const typeSet = types != null ? new Set(types) : undefined;
    if (keys != null) {
      const views = store.views.get(keys);
      if (views.length === keys.length) {
        // if views length is the same as the keys length, we can return the views once
        // we filter for the types
        if (typeSet == null) return views;
        return views.filter((v) => typeSet.has(v.type));
      }
      const missing = keys.filter((k) => !store.views.has(k));
      const retrieved = await client.views.retrieve({ keys: missing, types });
      store.views.set(retrieved);
      return [...views, ...retrieved];
    }
    // if keys are not provided, we will have to retrieve all views and filter them
    const views = await client.views.retrieve({ keys, types });
    store.views.set(views);
    return views;
  },
  mountListeners: ({ store, onChange, query: { keys, types } }) => {
    const keysSet = keys ? new Set(keys) : undefined;
    const typesSet = types ? new Set(types) : undefined;
    return [
      store.views.onSet(async (view) => {
        if (keysSet != null && !keysSet.has(view.key)) return;
        if (typesSet != null && !typesSet.has(view.type)) return;
        onChange((prev) => {
          if (prev == null) return [view];
          return [...prev.filter((v) => v.key !== view.key), view];
        });
      }),
    ];
  },
});
