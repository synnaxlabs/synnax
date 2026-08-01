// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type group, type ontology } from "@synnaxlabs/client";
import { verbs } from "@synnaxlabs/x";

import { Flux } from "@/flux";

export const RESOURCE_NAME = "group";
export const PLURAL_RESOURCE_NAME = "groups";

export interface CreateParams extends group.CreateParams {}

export const { useUpdate: useCreate } = Flux.createUpdate<CreateParams>({
  name: RESOURCE_NAME,
  verbs: verbs.CREATE,
  update: async ({ data, client }) => {
    const { parent } = data;
    const res = await client.groups.create(data);
    return { ...res, parent };
  },
});

export type ListQuery = {
  parent?: ontology.ID;
  searchTerm?: string;
  offset?: number;
  limit?: number;
};

export const useList = Flux.createList<ListQuery, group.Key, group.Group>({
  name: PLURAL_RESOURCE_NAME,
  retrieve: async ({ client, query }) => {
    const { parent } = query;
    if (parent == null) return [];
    return await client.groups.retrieve({ ...query, parent });
  },
  retrieveByKey: async ({ client, key }) => await client.groups.retrieve({ key }),
  subscribe: ({ client, query }, handler) => {
    const { parent } = query;
    if (parent == null) return () => {};
    return client.groups.onChange({ ...query, parent }, handler);
  },
  getCached: ({ client, query }) => {
    const { parent } = query;
    if (parent == null) return undefined;
    return client.groups.getCached({ ...query, parent });
  },
  subscribeByKey: ({ client, key }, handler) =>
    client.groups.onChange({ key }, handler),
});

export interface DeleteParams {
  key: string;
}

export const { useUpdate: useDelete } = Flux.createUpdate<DeleteParams>({
  name: RESOURCE_NAME,
  verbs: verbs.DELETE,
  update: async ({ client, data }) => {
    await client.groups.delete(data.key);
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
