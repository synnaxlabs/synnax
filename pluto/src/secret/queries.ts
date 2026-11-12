// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type secret } from "@synnaxlabs/client";
import { verbs } from "@synnaxlabs/x";

import { Flux } from "@/flux";
import { flux } from "@/flux/aether";

const RESOURCE_NAME = "secret";
const PLURAL_RESOURCE_NAME = "secrets";

// Distributes over the union so each variant keeps its own fields.
export type ListQuery = secret.RetrieveMultipleParams extends infer P
  ? P extends unknown
    ? Omit<P, "includeUses">
    : never
  : never;

/** Lists secrets by name, unless the query sets an order. */
export const useList = ({
  query,
  ...rest
}: Omit<Flux.UseListParams<ListQuery, secret.Secret>, "name"> = {}): Flux.UseListReturn<
  secret.Key,
  secret.Secret
> =>
  Flux.useList((client) => client.secrets, {
    ...rest,
    name: PLURAL_RESOURCE_NAME,
    query: query === null ? null : { order: { field: "name" }, ...query },
  });

export type RetrieveQuery = secret.RetrieveSingleParams;

export const { use } = Flux.createRetrieve<RetrieveQuery, secret.Secret>({
  name: RESOURCE_NAME,
  retrieve: async ({ client, query }) => await client.secrets.retrieve(query),
  onChange: ({ client, query }, handler) =>
    client.secrets.watch(query).onChange(handler),
  getCached: ({ client, query }) => client.secrets.watch(query).getCached(),
});

export type RetrieveMultipleQuery = secret.RetrieveMultipleParams;

export const { use: useMultiple } = Flux.createRetrieve<
  RetrieveMultipleQuery,
  secret.Secret[]
>({
  name: PLURAL_RESOURCE_NAME,
  ...flux.listReads(({ client, query }) => client.secrets.watch(query)),
});

/** Creates a secret, or replaces the secret whose key exists. */
export const { useUpdate: useCreate } = Flux.createUpdate<secret.New, secret.Secret>({
  name: RESOURCE_NAME,
  verbs: verbs.CREATE,
  update: async ({ client, data }) => await client.secrets.set(data),
});

export interface RenameParams extends Pick<secret.Secret, "key" | "name"> {}

export const { useUpdate: useRename } = Flux.createUpdate<RenameParams>({
  name: RESOURCE_NAME,
  verbs: verbs.RENAME,
  update: async ({ client, data }) => {
    const { key, name } = data;
    await client.secrets.rename(key, name);
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
