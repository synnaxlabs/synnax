// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { rack } from "@synnaxlabs/client";

import { Flux } from "@/flux";

export const FLUX_STORE_KEY = "racks";

export interface FluxStore extends Flux.UnaryStore<rack.Key, rack.Payload> {}

interface SubStore extends Flux.Store {
  [FLUX_STORE_KEY]: FluxStore;
}

const SET_RACK_LISTENER: Flux.ChannelListener<SubStore, typeof rack.keyZ> = {
  channel: rack.SET_CHANNEL_NAME,
  schema: rack.keyZ,
  onChange: async ({ store, changed, client }) => {
    const r = await client.hardware.racks.retrieve({
      key: changed,
      includeStatus: true,
    });
    store.racks.set(changed, r.payload);
  },
};

const DELETE_RACK_LISTENER: Flux.ChannelListener<SubStore, typeof rack.keyZ> = {
  channel: rack.DELETE_CHANNEL_NAME,
  schema: rack.keyZ,
  onChange: ({ store, changed }) => store.racks.delete(changed),
};

const SET_STATUS_LISTENER: Flux.ChannelListener<SubStore, typeof rack.statusZ> = {
  channel: rack.STATUS_CHANNEL_NAME,
  schema: rack.statusZ,
  onChange: ({ store, changed }) =>
    store.racks.set(changed.details.rack, (prev) =>
      prev == null ? prev : { ...prev, status: changed },
    ),
};

export const FLUX_STORE_CONFIG: Flux.UnaryStoreConfig<SubStore> = {
  listeners: [SET_RACK_LISTENER, DELETE_RACK_LISTENER, SET_STATUS_LISTENER],
};

export interface ListParams {
  term?: string;
  offset?: number;
  limit?: number;
  includeStatus?: boolean;
}

const DEFAULT_PARAMS = {
  includeStatus: true,
};

const retrieveFn = async ({
  client,
  params,
  store,
}: Flux.RetrieveArgs<RetrieveParams, SubStore>) => {
  let rack = store.racks.get(params.key);
  if (rack == null) {
    rack = await client.hardware.racks.retrieve({
      ...DEFAULT_PARAMS,
      ...params,
    });
    store.racks.set(rack.key, rack);
  }
  return rack;
};
export const useList = Flux.createList<ListParams, rack.Key, rack.Payload, SubStore>({
  name: "Racks",
  retrieveCached: ({ store }) => store.racks.list(),
  retrieve: async ({ client, params, store }) => {
    const racks = await client.hardware.racks.retrieve({
      ...DEFAULT_PARAMS,
      ...params,
    });
    store.racks.set(racks);
    return racks;
  },
  retrieveByKey: async ({ client, key, store }) =>
    await retrieveFn({ client, params: { key }, store }),
  mountListeners: ({ store, onChange, onDelete }) => [
    store.racks.onSet((rack) => onChange(rack.key, rack)),
    store.racks.onDelete(onDelete),
  ],
});

export interface RetrieveParams {
  key: rack.Key;
  includeStatus?: boolean;
}

export const { useRetrieve, useRetrieveStateful } = Flux.createRetrieve<
  RetrieveParams,
  rack.Payload,
  SubStore
>({
  name: "Rack",
  retrieve: retrieveFn,
  mountListeners: ({ store, onChange, params: { key } }) => [
    store.racks.onSet(onChange, key),
  ],
});
