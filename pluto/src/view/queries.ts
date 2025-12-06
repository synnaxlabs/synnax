// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { view as View } from "@synnaxlabs/client";
import { array, type optional } from "@synnaxlabs/x";

import { Flux } from "@/flux";
import { Ontology } from "@/ontology";

export const FLUX_STORE_KEY = "views";
export const RESOURCE_NAME = "view";
export const PLURAL_RESOURCE_NAME = "views";

export interface FluxStore extends Flux.UnaryStore<View.Key, View.View> {}

export interface FluxSubStore extends Flux.Store {
  [FLUX_STORE_KEY]: FluxStore;
}

const SET_VIEW_LISTENER: Flux.ChannelListener<FluxSubStore, typeof View.viewZ> = {
  channel: View.SET_CHANNEL_NAME,
  schema: View.viewZ,
  onChange: ({ store, changed }) => store.views.set(changed.key, changed),
};

const DELETE_VIEW_LISTENER: Flux.ChannelListener<FluxSubStore, typeof View.keyZ> = {
  channel: View.DELETE_CHANNEL_NAME,
  schema: View.keyZ,
  onChange: ({ store, changed }) => store.views.delete(changed),
};

export const FLUX_STORE_CONFIG: Flux.UnaryStoreConfig<FluxSubStore> = {
  listeners: [SET_VIEW_LISTENER, DELETE_VIEW_LISTENER],
};

export interface ListQuery extends View.RetrieveMultipleParams {}

export const useList = Flux.createList<ListQuery, View.Key, View.View, FluxSubStore>({
  name: PLURAL_RESOURCE_NAME,
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
        onChange(view.key, view);
      }),
      store.views.onDelete(onDelete),
    ];
  },
});

export const { useUpdate: useCreate } = Flux.createUpdate<View.New, FluxSubStore>({
  name: RESOURCE_NAME,
  verbs: Flux.CREATE_VERBS,
  update: async ({ client, data, store, rollbacks }) => {
    const views = await client.views.create(data);
    rollbacks.push(store.views.set(views));
    return views;
  },
});

export type DeleteParams = View.Key | View.Key[];

export const { useUpdate: useDelete } = Flux.createUpdate<DeleteParams, FluxSubStore>({
  name: RESOURCE_NAME,
  verbs: Flux.DELETE_VERBS,
  update: async ({ client, data, store, rollbacks }) => {
    const keys = array.toArray(data);
    const ids = keys.map((key) => View.ontologyID(key));
    const relFilter = Ontology.filterRelationshipsThatHaveIDs(ids);
    rollbacks.push(store.relationships.delete(relFilter));
    rollbacks.push(store.views.delete(keys));
    rollbacks.push(store.resources.delete(keys));
    await client.views.delete(data);
    return data;
  },
});

const retrieveSingle = async ({
  client,
  query: { key },
  store,
}: Flux.RetrieveParams<View.RetrieveSingleParams, FluxSubStore>) => {
  const cached = store.views.get(key);
  if (cached != null) return cached;
  const v = await client.views.retrieve({ key });
  store.views.set(v);
  return v;
};

export const { useRetrieve } = Flux.createRetrieve<
  View.RetrieveSingleParams,
  View.View,
  FluxSubStore
>({
  name: RESOURCE_NAME,
  retrieve: retrieveSingle,
  mountListeners: ({ store, query: { key }, onChange }) => [
    store.views.onSet(onChange, key),
  ],
});

const zeroValues = {
  name: "",
  type: "",
  query: {},
};
export interface FormQuery
  extends optional.Optional<View.RetrieveSingleParams, "key"> {}

export const useForm = Flux.createForm<FormQuery, typeof View.newZ, FluxSubStore>({
  name: RESOURCE_NAME,
  schema: View.newZ,
  initialValues: zeroValues,
  retrieve: async ({ client, query: { key }, store, reset }) => {
    if (key == null) return;
    reset(await retrieveSingle({ client, store, query: { key } }));
  },
  update: async ({ client, value, reset, store, rollbacks }) => {
    const updated = await client.views.create(value());
    reset(updated);
    rollbacks.push(store.views.set(updated.key, updated));
  },
  mountListeners: ({ store, query: { key }, reset }) => [
    store.views.onSet((view) => {
      if (key == null || view.key !== key) return;
      reset(view);
    }),
  ],
});
