// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type rack } from "@synnaxlabs/client";
import { verbs } from "@synnaxlabs/x";

import { Flux } from "@/flux";
import { flux } from "@/flux/aether";

const RESOURCE_NAME = "rack";
const PLURAL_RESOURCE_NAME = "racks";

export type RetrieveQuery = {
  key: rack.Key;
  includeStatus?: boolean;
};

const BASE_QUERY = { includeStatus: true } as const;

export type ListQuery = rack.RetrieveMultipleParams;

/** Lists racks with their statuses, also for a rack outside the list. */
export const useList = ({
  query,
  ...rest
}: Omit<
  Flux.UseListParams<ListQuery, rack.Rack>,
  "itemQuery" | "name"
> = {}): Flux.UseListReturn<rack.Key, rack.Rack> =>
  Flux.useList((client) => client.racks, {
    ...rest,
    name: PLURAL_RESOURCE_NAME,
    query: query === null ? null : { ...BASE_QUERY, ...query },
    itemQuery: (key) => ({ ...BASE_QUERY, key }),
  });

export const { use, useResult, useEnsure, createSelector } = Flux.createRetrieve<
  RetrieveQuery,
  rack.Rack
>({
  name: RESOURCE_NAME,
  normalizeQuery: (query) => ({ ...BASE_QUERY, ...query }),
  retrieve: async ({ client, query }) => await client.racks.retrieve(query),
  onChange: ({ client, query }, handler) => client.racks.watch(query).onChange(handler),
  getCached: ({ client, query }) => client.racks.watch(query).getCached(),
});

export const { use: useMultiple, useResult: useResultMultiple } = Flux.createRetrieve<
  ListQuery,
  rack.Rack[]
>({
  name: PLURAL_RESOURCE_NAME,
  normalizeQuery: (query) => ({ ...BASE_QUERY, ...query }),
  ...flux.listReads(({ client, query }) => client.racks.watch(query)),
});

export const useName = createSelector(({ name }) => name);

export const useIntegrations = createSelector(({ integrations }) => integrations);

export type UseDeleteParams = rack.Key | rack.Key[];

export const { useUpdate: useDelete } = Flux.createUpdate<UseDeleteParams>({
  name: RESOURCE_NAME,
  verbs: verbs.DELETE,
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
  verbs: verbs.RENAME,
  update: async ({ data, client }) => {
    const { key, name } = data;
    await client.racks.rename(key, name);
    return data;
  },
});
