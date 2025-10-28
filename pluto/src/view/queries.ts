// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { view } from "@synnaxlabs/client";

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
  update: async ({ client, data }) => {
    await client.views.delete(data);
    return data;
  },
});
