// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ontology, rack } from "@synnaxlabs/client";
import { array } from "@synnaxlabs/x";

import { Flux } from "@/flux";
import { Ontology } from "@/ontology";

export const FLUX_STORE_KEY = "racks";
const RESOURCE_NAME = "Rack";
const PLURAL_RESOURCE_NAME = "Racks";

export interface FluxStore extends Flux.UnaryStore<rack.Key, rack.Payload> {}

interface FluxSubStore extends Flux.Store {
  [FLUX_STORE_KEY]: FluxStore;
  [Ontology.RELATIONSHIPS_FLUX_STORE_KEY]: Ontology.RelationshipFluxStore;
  [Ontology.RESOURCES_FLUX_STORE_KEY]: Ontology.ResourceFluxStore;
}

const SET_RACK_LISTENER: Flux.ChannelListener<FluxSubStore, typeof rack.keyZ> = {
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

const DELETE_RACK_LISTENER: Flux.ChannelListener<FluxSubStore, typeof rack.keyZ> = {
  channel: rack.DELETE_CHANNEL_NAME,
  schema: rack.keyZ,
  onChange: ({ store, changed }) => store.racks.delete(changed),
};

const SET_STATUS_LISTENER: Flux.ChannelListener<FluxSubStore, typeof rack.statusZ> = {
  channel: rack.STATUS_CHANNEL_NAME,
  schema: rack.statusZ,
  onChange: ({ store, changed }) =>
    store.racks.set(changed.details.rack, (prev) =>
      prev == null ? prev : { ...prev, status: changed },
    ),
};

export const FLUX_STORE_CONFIG: Flux.UnaryStoreConfig<FluxSubStore> = {
  listeners: [SET_RACK_LISTENER, DELETE_RACK_LISTENER, SET_STATUS_LISTENER],
};

export interface RetrieveQuery {
  key: rack.Key;
  includeStatus?: boolean;
}

const BASE_QUERY: Partial<RetrieveQuery> = { includeStatus: true };

const retrieveSingle = async ({
  client,
  query,
  store,
}: Flux.RetrieveParams<RetrieveQuery, FluxSubStore>) => {
  let rack = store.racks.get(query.key);
  if (rack == null) {
    rack = await client.hardware.racks.retrieve({ ...BASE_QUERY, ...query });
    store.racks.set(rack.key, rack);
  }
  return rack;
};

export interface ListQuery extends rack.RetrieveMultipleParams {}

export const useList = Flux.createList<ListQuery, rack.Key, rack.Payload, FluxSubStore>(
  {
    name: PLURAL_RESOURCE_NAME,
    retrieveCached: ({ store }) => store.racks.list(),
    retrieve: async ({ client, query, store }) => {
      const racks = await client.hardware.racks.retrieve({ ...BASE_QUERY, ...query });
      store.racks.set(racks);
      return racks;
    },
    retrieveByKey: async ({ client, key, store }) =>
      await retrieveSingle({ client, query: { key }, store }),
    mountListeners: ({ store, onChange, onDelete }) => [
      store.racks.onSet((rack) => onChange(rack.key, rack)),
      store.racks.onDelete(onDelete),
    ],
  },
);

export const { useRetrieve, useRetrieveStateful } = Flux.createRetrieve<
  RetrieveQuery,
  rack.Payload,
  FluxSubStore
>({
  name: RESOURCE_NAME,
  retrieve: retrieveSingle,
  mountListeners: ({ store, onChange, query: { key } }) => [
    store.racks.onSet(onChange, key),
  ],
});

export type UseDeleteParams = rack.Key | rack.Key[];

export const { useUpdate: useDelete } = Flux.createUpdate<
  UseDeleteParams,
  FluxSubStore
>({
  name: RESOURCE_NAME,
  verbs: Flux.DELETE_VERBS,
  update: async ({ client, data, store, rollbacks }) => {
    const keys = array.toArray(data);
    const ids = rack.ontologyID(keys);
    const relFilter = Ontology.filterRelationshipsThatHaveIDs(ids);
    rollbacks.push(store.relationships.delete(relFilter));
    rollbacks.push(store.resources.delete(ontology.idToString(ids)));
    rollbacks.push(store.racks.delete(keys));
    await client.hardware.racks.delete(keys);
    return data;
  },
});

export interface RenameParams extends Pick<rack.Rack, "key" | "name"> {}

export const { useUpdate: useRename } = Flux.createUpdate<RenameParams, FluxSubStore>({
  name: RESOURCE_NAME,
  verbs: Flux.RENAME_VERBS,
  update: async ({ data, client, rollbacks, store }) => {
    const { key, name } = data;
    rollbacks.push(Flux.partialUpdate(store.racks, key, { name }));
    rollbacks.push(Ontology.renameFluxResource(store, rack.ontologyID(key), name));
    const r = await retrieveSingle({ client, query: { key }, store });
    await client.hardware.racks.create({ ...r, name });
    return data;
  },
});
