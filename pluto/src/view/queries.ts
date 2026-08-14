// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { query, view } from "@synnaxlabs/client";
import { verbs } from "@synnaxlabs/x";
import { useEffect } from "react";
import { type z } from "zod";

import { Flux } from "@/flux";
import { Synnax } from "@/synnax";

export const RESOURCE_NAME = "view";
export const PLURAL_RESOURCE_NAME = "views";

export type ListQuery = view.RetrieveMultipleParams;

export const useList = Flux.createList<ListQuery, view.Key, view.View>({
  name: PLURAL_RESOURCE_NAME,
  retrieve: async ({ client, query }) => await client.views.retrieve(query),
  retrieveByKey: async ({ client, key }) => await client.views.retrieve(key),
  onChange: ({ client, query }, handler) => client.views.onChange(query, handler),
  onChangeByKey: ({ client, key }, handler) => client.views.onChange(key, handler),
  getCached: ({ client, query }) => client.views.getCached(query),
});

export const { useUpdate: useCreate } = Flux.createUpdate<view.New>({
  name: RESOURCE_NAME,
  verbs: verbs.CREATE,
  update: async ({ client, data }) => await client.views.create(data),
});

export type DeleteParams = view.Key | view.Key[];

export const { useUpdate: useDelete } = Flux.createUpdate<DeleteParams>({
  name: RESOURCE_NAME,
  verbs: verbs.DELETE,
  update: async ({ client, data, onOptimisticComplete }) => {
    await client.views.delete(data, {
      onOptimistic: async () => await onOptimisticComplete(data),
    });
    return data;
  },
});

export const formSchema = view.viewZ.partial({ key: true });

const ZERO_VALUES: z.infer<typeof formSchema> = {
  name: "",
  type: "",
  query: {},
};
export type FormQuery = view.RetrieveSingleParams;

export const useForm = Flux.createForm<FormQuery, typeof formSchema>({
  name: RESOURCE_NAME,
  schema: formSchema,
  initialValues: ZERO_VALUES,
  retrieve: async ({ client, query: { key } }) => await client.views.retrieve(key),
  getCached: ({ client, query: { key } }) => {
    const cached = client.views.getCached(key);
    return query.isLive(cached) ? cached : undefined;
  },
  update: async ({ client, value, reset }) => {
    const updated = await client.views.create(value());
    reset(updated);
  },
  mountListeners: ({ client, query: { key }, reset }) =>
    client.views.onChange(key, (result) => {
      if (query.isLive(result)) reset(result);
    }),
});

export interface RenameParams extends Pick<view.View, "key" | "name"> {}

export const { useUpdate: useRename } = Flux.createUpdate<RenameParams>({
  name: RESOURCE_NAME,
  verbs: verbs.RENAME,
  update: async ({ client, data, onOptimisticComplete }) => {
    const { key, name } = data;
    await client.views.rename(key, name, {
      onOptimistic: async () => await onOptimisticComplete(data),
    });
    return data;
  },
});

export const useSetSynchronizer = (onSet: (view: view.View) => void): void => {
  const client = Synnax.use();
  useEffect(() => client?.views.onSet(onSet), [client, onSet]);
};

export const useDeleteSynchronizer = (onDelete: (key: view.Key) => void): void => {
  const client = Synnax.use();
  useEffect(() => client?.views.onDelete(onDelete), [client, onDelete]);
};
