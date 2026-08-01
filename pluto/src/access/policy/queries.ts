// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { access, ontology } from "@synnaxlabs/client";
import { array } from "@synnaxlabs/x";
import { z } from "zod";

import { Flux } from "@/flux";
import { type List } from "@/list";

const RESOURCE_NAME = "Policy";
const PLURAL_RESOURCE_NAME = "Policies";

export const formSchema = z.object({
  key: z.uuid().optional(),
  name: z.string().min(1, "Name is required"),
  objects: ontology.idZ.array(),
  actions: access.actionZ.array(),
});

export type RetrieveQuery = {
  key: access.policy.Key;
};

export const { useRetrieve } = Flux.createRetrieve<RetrieveQuery, access.policy.Policy>(
  {
    name: RESOURCE_NAME,
    retrieve: async ({ client, query: { key } }) =>
      await client.access.policies.retrieve(key),
    subscribe: ({ client, query: { key } }, handler) =>
      client.access.policies.onChange(key, handler),
    getCached: ({ client, query: { key } }) => client.access.policies.getCached(key),
  },
);

export type ListParams = List.PagerParams;

export const useList = Flux.createList<
  ListParams,
  access.policy.Key,
  access.policy.Policy
>({
  name: PLURAL_RESOURCE_NAME,
  retrieve: async ({ client, query }) => await client.access.policies.retrieve(query),
  retrieveByKey: async ({ client, key }) => await client.access.policies.retrieve(key),
  subscribe: ({ client, query }, handler) =>
    client.access.policies.onChange(query, handler),
  subscribeByKey: ({ client, key }, handler) =>
    client.access.policies.onChange(key, handler),
  getCached: ({ client, query }) => client.access.policies.getCached(query),
});

export type DeleteParams = access.policy.Key | access.policy.Key[];

export const { useUpdate: useDelete } = Flux.createUpdate<DeleteParams>({
  name: RESOURCE_NAME,
  verbs: Flux.DELETE_VERBS,
  update: async ({ client, data, onOptimisticComplete }) => {
    const keys = array.toArray(data);
    await client.access.policies.delete(keys, {
      onOptimistic: async () => await onOptimisticComplete(data),
    });
    return data;
  },
});

export interface RenameParams {
  key: access.policy.Key;
  name: string;
}

export const { useUpdate: useRename } = Flux.createUpdate<RenameParams>({
  name: RESOURCE_NAME,
  verbs: Flux.RENAME_VERBS,
  update: async ({ client, data, onOptimisticComplete }) => {
    const { key, name } = data;
    await client.access.policies.rename(key, name, {
      onOptimistic: async () => await onOptimisticComplete(data),
    });
    return data;
  },
});

export const useForm = Flux.createForm<Partial<RetrieveQuery>, typeof formSchema>({
  name: RESOURCE_NAME,
  schema: formSchema,
  initialValues: {
    key: undefined,
    name: "",
    objects: [],
    actions: [],
  },
  retrieve: async ({ client, query, reset }) => {
    if (query.key == null) return;
    reset(await client.access.policies.retrieve(query.key));
  },
  update: async ({ client, value, set }) => {
    const p = await client.access.policies.create(value());
    set("key", p.key);
  },
});
