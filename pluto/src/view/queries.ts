// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { view } from "@synnaxlabs/client";
import { array, type optional } from "@synnaxlabs/x";
import { useEffect } from "react";

import { Flux } from "@/flux";
import { Ontology } from "@/ontology";
import { state } from "@/state";

export const FLUX_STORE_KEY = "views";
export const RESOURCE_NAME = "view";
export const PLURAL_RESOURCE_NAME = "views";

export interface FluxStore extends Flux.UnaryStore<view.Key, view.View> {}

export interface FluxSubStore extends Flux.Store {
  [FLUX_STORE_KEY]: FluxStore;
  [Ontology.RELATIONSHIPS_FLUX_STORE_KEY]: Ontology.RelationshipFluxStore;
  [Ontology.RESOURCES_FLUX_STORE_KEY]: Ontology.ResourceFluxStore;
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
    const keysSet = keys != null ? new Set(keys) : undefined;
    const typesSet = types != null ? new Set(types) : undefined;
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

const retrieveSingle = async ({
  client,
  query: { key },
  store,
}: Flux.RetrieveParams<view.RetrieveSingleParams, FluxSubStore>) => {
  const cached = store.views.get(key);
  if (cached != null) return cached;
  const v = await client.views.retrieve({ key });
  store.views.set(v);
  return v;
};

const ZERO_VALUES = {
  name: "",
  type: "",
  query: {},
};
export interface FormQuery extends optional.Optional<
  view.RetrieveSingleParams,
  "key"
> {}

export const useForm = Flux.createForm<FormQuery, typeof view.newZ, FluxSubStore>({
  name: RESOURCE_NAME,
  schema: view.newZ,
  initialValues: ZERO_VALUES,
  retrieve: async ({ client, query: { key }, store, reset }) => {
    if (key == null) return;
    reset(await retrieveSingle({ client, store, query: { key } }));
  },
  update: async ({ client, value, reset, store, rollbacks }) => {
    const updated = await client.views.create(value());
    reset(updated);
    rollbacks.push(store.views.set(updated.key, updated));
  },
  mountListeners: ({ store, get, reset }) => [
    store.views.onSet((view) => {
      const prevKey = get<string>("key", { optional: true })?.value;
      if (prevKey == null || view.key !== prevKey) return;
      reset(view);
    }),
  ],
});

export interface RenameParams extends Pick<view.View, "key" | "name"> {}

export const { useUpdate: useRename } = Flux.createUpdate<RenameParams, FluxSubStore>({
  name: RESOURCE_NAME,
  verbs: Flux.RENAME_VERBS,
  update: async ({ client, data, store, rollbacks }) => {
    const { key, name } = data;
    const v = await retrieveSingle({ client, store, query: { key } });
    rollbacks.push(
      store.views.set(
        key,
        state.skipUndefined((p) => ({ ...p, name })),
      ),
    );
    rollbacks.push(Ontology.renameFluxResource(store, view.ontologyID(key), name));
    await client.views.create({ ...v, name });
    return data;
  },
});

export const useSetSynchronizer = (onSet: (view: view.View) => void): void => {
  const store = Flux.useStore<FluxSubStore>();
  useEffect(() => store.views.onSet(onSet), [store, onSet]);
};

export const useDeleteSynchronizer = (onDelete: (key: view.Key) => void): void => {
  const store = Flux.useStore<FluxSubStore>();
  useEffect(() => store.views.onDelete(onDelete), [store, onDelete]);
};
