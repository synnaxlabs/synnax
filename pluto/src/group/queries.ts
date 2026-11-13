// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type group, query, type Synnax } from "@synnaxlabs/client";
import { verbs } from "@synnaxlabs/x";

import { Flux } from "@/flux";
import { flux } from "@/flux/aether";

export const RESOURCE_NAME = "group";
const PLURAL_RESOURCE_NAME = "groups";

export interface CreateParams extends group.CreateParams {}

export const { useUpdate: useCreate } = Flux.createUpdate<CreateParams>({
  name: RESOURCE_NAME,
  verbs: verbs.CREATE,
  update: async ({ data, client }) => await client.groups.create(data),
});

export const { useResult: useResultList } = Flux.createRetrieve<
  group.RetrieveMultipleParams,
  group.Group[]
>({
  name: PLURAL_RESOURCE_NAME,
  ...flux.listReads(({ client, query }) => client.groups.watch(query)),
});

export type RetrieveAncestorsQuery = { keys: group.Key[] };

const getCachedAncestry = (
  client: Synnax,
  keys: group.Key[],
): group.Group[] | undefined => {
  const groups = client.groups.watch(keys).getCached();
  const ancestors = client.groups.getCachedAncestors(keys);
  if (!query.isLive(groups) || ancestors == null) return undefined;
  return [...groups, ...ancestors];
};

/**
 * Retrieves the groups with the keys and every group above them. A rename or a move of
 * one of them updates the answer.
 */
export const { useResult: useResultAncestors } = Flux.createRetrieve<
  RetrieveAncestorsQuery,
  group.Group[]
>({
  name: "group ancestors",
  retrieve: async ({ client, query: { keys } }) => {
    const [groups, ancestors] = await Promise.all([
      client.groups.retrieve(keys),
      client.groups.retrieveAncestors(keys),
    ]);
    return [...groups, ...ancestors];
  },
  onChange: ({ client, query: { keys } }, handler) =>
    client.groups.onAncestorsChange(keys, () =>
      handler(getCachedAncestry(client, keys)),
    ),
  getCached: ({ client, query: { keys } }) => getCachedAncestry(client, keys),
});

export interface DeleteParams {
  key: string;
}

export const { useUpdate: useDelete } = Flux.createUpdate<DeleteParams>({
  name: RESOURCE_NAME,
  verbs: verbs.DELETE,
  update: async ({ client, data, onOptimisticComplete }) => {
    await client.groups.delete(data.key, {
      onOptimistic: async () => await onOptimisticComplete(data),
    });
    return data;
  },
});

export interface RenameParams extends Pick<group.Group, "key" | "name"> {}

export const { useUpdate: useRename } = Flux.createUpdate<RenameParams>({
  name: RESOURCE_NAME,
  verbs: verbs.RENAME,
  update: async ({ client, data }) => {
    const { key, name } = data;
    await client.groups.rename(key, name);
    return data;
  },
});
