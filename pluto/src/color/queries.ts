// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color } from "@synnaxlabs/client";
import { verbs } from "@synnaxlabs/x";

import { Flux } from "@/flux";

const RESOURCE_NAME = "named color";
const PLURAL_RESOURCE_NAME = "named colors";

export type RetrieveQuery = color.RetrieveSingleParams;

export const { use: useRetrieve, useResult } = Flux.createRetrieve<
  RetrieveQuery,
  color.Named
>({
  name: RESOURCE_NAME,
  retrieve: async ({ client, query: { key } }) => await client.colors.retrieve(key),
  onChange: ({ client, query: { key } }, handler) =>
    client.colors.watch(key).onChange(handler),
  getCached: ({ client, query: { key } }) => client.colors.watch(key).getCached(),
});

export type RetrieveMultipleQuery = {
  keys: color.Key[];
};

export const { use: useRetrieveMultiple, useResult: useResultMultiple } =
  Flux.createRetrieve<RetrieveMultipleQuery, color.Named[]>({
    name: PLURAL_RESOURCE_NAME,
    retrieve: async ({ client, query: { keys } }) => await client.colors.retrieve(keys),
    onChange: ({ client, query: { keys } }, handler) =>
      client.colors.watch(keys).onChange(handler),
    getCached: ({ client, query: { keys } }) => client.colors.watch(keys).getCached(),
  });

export const { useUpdate: useCreate } = Flux.createUpdate<color.New, color.Named>({
  name: RESOURCE_NAME,
  verbs: verbs.CREATE,
  update: async ({ client, data }) => await client.colors.create(data),
});

export interface RenameParams extends Pick<color.Named, "key" | "name"> {}

export const { useUpdate: useRename } = Flux.createUpdate<RenameParams>({
  name: RESOURCE_NAME,
  verbs: verbs.RENAME,
  update: async ({ client, data }) => {
    const { key, name } = data;
    // A dispatch reaches the Core only for a cached named color.
    await client.colors.retrieve(key);
    await client.colors.dispatch(key, color.set({ path: ["name"], value: name }));
    return data;
  },
});

export type DeleteParams = color.Key | color.Key[];

export const { useUpdate: useDelete } = Flux.createUpdate<DeleteParams>({
  name: RESOURCE_NAME,
  verbs: verbs.DELETE,
  update: async ({ client, data, onOptimisticComplete }) => {
    await client.colors.delete(data, {
      onOptimistic: async () => await onOptimisticComplete(data),
    });
    return data;
  },
});
