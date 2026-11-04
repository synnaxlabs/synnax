// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type group } from "@synnaxlabs/client";
import { verbs } from "@synnaxlabs/x";

import { Flux } from "@/flux";

export const RESOURCE_NAME = "group";
const PLURAL_RESOURCE_NAME = "groups";

export interface CreateParams extends group.CreateParams {}

export const { useUpdate: useCreate } = Flux.createUpdate<CreateParams>({
  name: RESOURCE_NAME,
  verbs: verbs.CREATE,
  update: async ({ data, client }) => await client.groups.create(data),
});

export type ListQuery = Pick<
  group.RetrieveRequest,
  "parents" | "scopes" | "nested" | "searchTerm" | "offset" | "limit"
>;

export const useList = Flux.createList<ListQuery, group.Key, group.Group>({
  name: PLURAL_RESOURCE_NAME,
  retrieve: async ({ client, query }) => await client.groups.retrieve(query),
  retrieveByKey: async ({ client, key }) => await client.groups.retrieve(key),
  onChange: ({ client, query }, handler) => client.groups.onChange(query, handler),
  getCached: ({ client, query }) => client.groups.getCached(query),
  onChangeByKey: ({ client, key }, handler) => client.groups.onChange(key, handler),
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
  update: async ({ client, data, onOptimisticComplete }) => {
    const { key, name } = data;
    await client.groups.rename(key, name, {
      onOptimistic: async () => await onOptimisticComplete(data),
    });
    return data;
  },
});
