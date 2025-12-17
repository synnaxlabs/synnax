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
import { state } from "@/state";
import { Status } from "@/status";

export const FLUX_STORE_KEY = "racks";
const RESOURCE_NAME = "rack";
const PLURAL_RESOURCE_NAME = "racks";

export interface FluxStore extends Flux.UnaryStore<
  rack.Key,
  Omit<rack.Payload, "status">
> {}

export interface FluxSubStore
  extends Flux.Store, Status.FluxSubStore, Ontology.FluxSubStore {
  [FLUX_STORE_KEY]: FluxStore;
}

const SET_RACK_LISTENER: Flux.ChannelListener<FluxSubStore, typeof rack.rackZ> = {
  channel: rack.SET_CHANNEL_NAME,
  schema: rack.rackZ,
  onChange: ({ store, changed }) => store.racks.set(changed),
};

const DELETE_RACK_LISTENER: Flux.ChannelListener<FluxSubStore, typeof rack.keyZ> = {
  channel: rack.DELETE_CHANNEL_NAME,
  schema: rack.keyZ,
  onChange: ({ store, changed }) => store.racks.delete(changed),
};

export const FLUX_STORE_CONFIG: Flux.UnaryStoreConfig<FluxSubStore> = {
  listeners: [SET_RACK_LISTENER, DELETE_RACK_LISTENER],
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
  const cached = store.racks.get(query.key);
  if (cached != null) {
    const status = await Status.retrieveSingle<typeof rack.statusDetailsZ>({
      store,
      client,
      query: { key: rack.statusKey(query.key) },
      detailsSchema: rack.statusDetailsZ,
    });
    return { ...cached, status };
  }
  const res = await client.racks.retrieve({ ...BASE_QUERY, ...query });
  store.racks.set(res.key, res);
  if (res.status != null) store.statuses.set(res.status);
  return res;
};

export interface ListQuery extends rack.RetrieveMultipleParams {}

export const useList = Flux.createList<ListQuery, rack.Key, rack.Payload, FluxSubStore>(
  {
    name: PLURAL_RESOURCE_NAME,
    retrieveCached: ({ store }) => store.racks.list(),
    retrieve: async ({ client, query, store }) => {
      const racks = await client.racks.retrieve({ ...BASE_QUERY, ...query });
      store.racks.set(racks);
      const statuses = racks.map((r) => r.status).filter((s) => s != null);
      store.statuses.set(statuses);
      return racks;
    },
    retrieveByKey: async ({ client, key, store }) =>
      await retrieveSingle({ client, query: { key }, store }),
    mountListeners: ({ store, onChange, onDelete }) => [
      store.racks.onSet((rack) => onChange(rack.key, rack)),
      store.racks.onDelete(onDelete),
      store.statuses.onSet((s) => {
        const stat = rack.statusZ.safeParse(s);
        if (!stat.success) return;
        onChange(
          stat.data.details.rack,
          state.skipNull((p) => ({ ...p, status: stat.data })),
        );
      }),
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
    store.racks.onSet(
      (changed) => onChange((p) => ({ ...changed, status: p?.status })),
      key,
    ),
    store.statuses.onSet((status) => {
      const parsed = rack.statusZ.parse(status);
      onChange(state.skipUndefined((p) => ({ ...p, status: parsed })));
    }, rack.statusKey(key)),
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
    await client.racks.delete(keys);
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
    await client.racks.create({ ...r, name });
    return data;
  },
});
