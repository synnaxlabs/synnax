// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NotFoundError, query, type secret } from "@synnaxlabs/client";
import { verbs } from "@synnaxlabs/x";

import { Flux } from "@/flux";

const RESOURCE_NAME = "secret";
const PLURAL_RESOURCE_NAME = "secrets";

export type ListQuery = Omit<secret.RetrieveMultipleParams, "includeUses">;

export const useList = Flux.createList<ListQuery, secret.Key, secret.Secret>({
  name: PLURAL_RESOURCE_NAME,
  retrieve: async ({ client, query }) => await client.secrets.retrieve(query),
  retrieveByKey: async ({ client, key }) => await client.secrets.retrieve(key),
  onChange: ({ client, query }, handler) => client.secrets.onChange(query, handler),
  onChangeByKey: ({ client, key }, handler) => client.secrets.onChange(key, handler),
  getCached: ({ client, query }) => client.secrets.getCached(query),
});

export type RetrieveQuery = secret.RetrieveSingleParams & {
  /** Fills `uses` with the IDs of the resources that use the secret. */
  includeUses?: boolean;
};

// The client reads uses only through a query for a list of keys.
const usesQuery = (key: secret.Key): secret.RetrieveMultipleParams => ({
  keys: [key],
  includeUses: true,
});

const first = (
  secrets: query.Cached<secret.Secret[]> | undefined,
): secret.Secret | undefined => (query.isLive(secrets) ? secrets[0] : undefined);

export const { use } = Flux.createRetrieve<RetrieveQuery, secret.Secret>({
  name: RESOURCE_NAME,
  retrieve: async ({ client, query: { key, includeUses } }) => {
    if (includeUses !== true) return await client.secrets.retrieve(key);
    const [s] = await client.secrets.retrieve(usesQuery(key));
    if (s == null) throw new NotFoundError(`Secret with key ${key} not found`);
    return s;
  },
  onChange: ({ client, query: { key, includeUses } }, handler) =>
    includeUses === true
      ? client.secrets.onChange(usesQuery(key), (secrets) => handler(first(secrets)))
      : client.secrets.onChange(key, handler),
  getCached: ({ client, query: { key, includeUses } }) =>
    includeUses === true
      ? first(client.secrets.getCached(usesQuery(key)))
      : client.secrets.getCached(key),
});

/** Creates a secret, or replaces the secret whose key exists. */
export const { useUpdate: useCreate } = Flux.createUpdate<secret.New, secret.Secret>({
  name: RESOURCE_NAME,
  verbs: verbs.CREATE,
  update: async ({ client, data }) => await client.secrets.create(data),
});

export interface RenameParams extends Pick<secret.Secret, "key" | "name"> {}

export const { useUpdate: useRename } = Flux.createUpdate<RenameParams>({
  name: RESOURCE_NAME,
  verbs: verbs.RENAME,
  update: async ({ client, data, onOptimisticComplete }) => {
    const { key, name } = data;
    await client.secrets.rename(key, name, {
      onOptimistic: async () => await onOptimisticComplete(data),
    });
    return data;
  },
});

export type DeleteParams = secret.Key | secret.Key[];

export const { useUpdate: useDelete } = Flux.createUpdate<DeleteParams>({
  name: RESOURCE_NAME,
  verbs: verbs.DELETE,
  update: async ({ client, data, onOptimisticComplete }) => {
    await client.secrets.delete(data, {
      onOptimistic: async () => await onOptimisticComplete(data),
    });
    return data;
  },
});
