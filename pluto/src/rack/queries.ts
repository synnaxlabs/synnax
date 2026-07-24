// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type rack } from "@synnaxlabs/client";

import { Flux } from "@/flux";

const RESOURCE_NAME = "rack";
const PLURAL_RESOURCE_NAME = "racks";

export type RetrieveQuery = {
  key: rack.Key;
  includeStatus?: boolean;
};

const BASE_QUERY = { includeStatus: true } as const;

export type ListQuery = rack.RetrieveMultipleParams;

export const useList = Flux.createList<ListQuery, rack.Key, rack.Rack>({
  name: PLURAL_RESOURCE_NAME,
  retrieve: async ({ client, query }) =>
    await client.racks.retrieve({ ...BASE_QUERY, ...query }),
  retrieveByKey: async ({ client, key }) =>
    await client.racks.retrieve({ ...BASE_QUERY, key }),
  subscribe: ({ client, query }, handler) =>
    client.racks.onChange({ ...BASE_QUERY, ...query }, handler),
  getCached: ({ client, query }) => client.racks.getCached({ ...BASE_QUERY, ...query }),
});

export const { useRetrieve, useRetrieveStateful } = Flux.createRetrieve<
  RetrieveQuery,
  rack.Rack
>({
  name: RESOURCE_NAME,
  retrieve: async ({ client, query }) =>
    await client.racks.retrieve({ ...BASE_QUERY, ...query }),
  subscribe: ({ client, query }, handler) =>
    client.racks.onChange({ ...BASE_QUERY, ...query }, handler),
  getCached: ({ client, query }) => client.racks.getCached({ ...BASE_QUERY, ...query }),
});

export type UseDeleteParams = rack.Key | rack.Key[];

export const { useUpdate: useDelete } = Flux.createUpdate<UseDeleteParams>({
  name: RESOURCE_NAME,
  verbs: Flux.DELETE_VERBS,
  update: async ({ client, data, onOptimisticComplete }) => {
    await client.racks.delete(data, {
      onOptimistic: async () => await onOptimisticComplete(data),
    });
    return data;
  },
});

export interface RenameParams extends Pick<rack.Rack, "key" | "name"> {}

export const { useUpdate: useRename } = Flux.createUpdate<RenameParams>({
  name: RESOURCE_NAME,
  verbs: Flux.RENAME_VERBS,
  update: async ({ data, client, onOptimisticComplete }) => {
    const { key, name } = data;
    await client.racks.rename(key, name, {
      onOptimistic: async () => await onOptimisticComplete(data),
    });
    return data;
  },
});
