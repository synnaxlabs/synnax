// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type auth, rack } from "@synnaxlabs/client";
import { errors, verbs } from "@synnaxlabs/x";

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
  normalizeQuery: (query) => ({ ...BASE_QUERY, ...query }),
  retrieve: async ({ client, query }) => await client.racks.retrieve(query),
  retrieveByKey: async ({ client, key }) =>
    await client.racks.retrieve({ ...BASE_QUERY, key }),
  onChange: ({ client, query }, handler) => client.racks.onChange(query, handler),
  onChangeByKey: ({ client, key }, handler) => client.racks.onChange(key, handler),
  getCached: ({ client, query }) => client.racks.getCached(query),
});

export const { use, useResult, useEnsure, createSelector } = Flux.createRetrieve<
  RetrieveQuery,
  rack.Rack
>({
  name: RESOURCE_NAME,
  normalizeQuery: (query) => ({ ...BASE_QUERY, ...query }),
  retrieve: async ({ client, query }) => await client.racks.retrieve(query),
  onChange: ({ client, query }, handler) => client.racks.onChange(query, handler),
  getCached: ({ client, query }) => client.racks.getCached(query),
});

export const { use: useMultiple, useResult: useResultMultiple } = Flux.createRetrieve<
  ListQuery,
  rack.Rack[]
>({
  name: PLURAL_RESOURCE_NAME,
  normalizeQuery: (query) => ({ ...BASE_QUERY, ...query }),
  retrieve: async ({ client, query }) => await client.racks.retrieve(query),
  onChange: ({ client, query }, handler) => client.racks.onChange(query, handler),
  getCached: ({ client, query }) => client.racks.getCached(query),
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
  update: async ({ data, client, onOptimisticComplete }) => {
    const { key, name } = data;
    await client.racks.rename(key, name, {
      onOptimistic: async () => await onOptimisticComplete(data),
    });
    return data;
  },
});

export interface CreateParams {
  name: string;
}

export interface CreateResult {
  rack: rack.Rack;
  /** The rack's first API key. */
  key: auth.APIKey;
  /** The only copy of the key's secret. */
  token: string;
}

/**
 * Creates a rack together with the API key its Driver logs in with. Deletes the rack
 * when the key cannot be created.
 */
export const { useUpdate: useCreate } = Flux.createUpdate<CreateParams, CreateResult>({
  name: RESOURCE_NAME,
  verbs: verbs.CREATE,
  update: async ({ client, data: { name } }) => {
    const created = await client.racks.create({ name });
    try {
      const { key, token } = await client.auth.createAPIKey({
        subject: rack.ontologyID(created.key),
        name,
      });
      return { rack: created, key, token };
    } catch (err) {
      // a rack without a key is one no Driver can log in as
      await client.racks.delete(created.key);
      throw errors.fromUnknown(err);
    }
  },
});
