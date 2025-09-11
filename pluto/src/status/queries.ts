// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology, status } from "@synnaxlabs/client";

import { Flux } from "@/flux";

export const FLUX_STORE_KEY = "statuses";

export interface FluxStore extends Flux.UnaryStore<status.Key, status.Status> {}

interface SubStore extends Flux.Store {
  statuses: FluxStore;
}

const SET_STATUS_LISTENER: Flux.ChannelListener<SubStore, typeof status.statusZ> = {
  channel: status.SET_CHANNEL_NAME,
  schema: status.statusZ,
  onChange: ({ store, changed }) => store.statuses.set(changed.key, changed),
};

const DELETE_STATUS_LISTENER: Flux.ChannelListener<SubStore, typeof status.keyZ> = {
  channel: status.DELETE_CHANNEL_NAME,
  schema: status.keyZ,
  onChange: ({ store, changed }) => store.statuses.delete(changed),
};

export const FLUX_STORE_CONFIG: Flux.UnaryStoreConfig<SubStore> = {
  listeners: [SET_STATUS_LISTENER, DELETE_STATUS_LISTENER],
};

export interface ListParams extends status.MultiRetrieveArgs {}

export const useList = Flux.createList<ListParams, status.Key, status.Status, SubStore>(
  {
    name: "Statuses",
    retrieve: async ({ client, params }) => await client.statuses.retrieve(params),
    retrieveByKey: async ({ client, key }) => await client.statuses.retrieve({ key }),
    mountListeners: ({ store, onChange, onDelete, params: { keys } }) => {
      const keysSet = keys ? new Set(keys) : undefined;
      return [
        store.statuses.onSet(async (status) => {
          if (keysSet != null && !keysSet.has(status.key)) return;
          onChange(status.key, status, { mode: "prepend" });
        }),
        store.statuses.onDelete(async (key) => onDelete(key)),
      ];
    },
  },
);

export interface DeleteParams {
  key: status.Key | status.Key[];
}

export const useDelete = Flux.createUpdate<DeleteParams, void>({
  name: "Status",
  update: async ({ client, params: { key } }) => await client.statuses.delete(key),
}).useDirect;

export interface SetParams {
  statuses: status.New | status.New[];
  parent?: ontology.ID;
}

export const useSet = Flux.createUpdate<SetParams, void>({
  name: "Status",
  update: async ({ client, params: { statuses, parent } }) => {
    if (Array.isArray(statuses)) await client.statuses.set(statuses, { parent });
    else await client.statuses.set(statuses, { parent });
  },
}).useDirect;

interface UseStatusParams {
  key: status.Key;
}

export const useRetrieve = Flux.createRetrieve<
  UseStatusParams,
  status.Status,
  SubStore
>({
  name: "Status",
  retrieve: async ({ client, params: { key } }) =>
    await client.statuses.retrieve({ key }),
  mountListeners: ({ store, params: { key }, onChange }) => [
    store.statuses.onSet(onChange, key),
  ],
});
