// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology, status, TimeStamp } from "@synnaxlabs/client";
import { useEffect } from "react";
import type z from "zod";

import { Flux } from "@/flux";
import { createForm } from "@/flux/form";
import { createList } from "@/flux/list";
import { createRetrieve, type RetrieveArgs } from "@/flux/retrieve";
import { createUpdate } from "@/flux/update";

export const FLUX_STORE_KEY = "statuses";

export interface FluxStore extends Flux.UnaryStore<status.Key, status.Status> {}

interface SubStore extends Flux.Store {
  statuses: FluxStore;
}

const SET_STATUS_LISTENER: Flux.ChannelListener<SubStore, typeof status.statusZ> = {
  channel: status.SET_CHANNEL_NAME,
  schema: status.statusZ,
  onChange: ({ store, changed }) => store.statuses.set(changed.key, changed),
};

const DELETE_STATUS_LISTENER: Flux.ChannelListener<SubStore, typeof status.keyZ> = {
  channel: status.DELETE_CHANNEL_NAME,
  schema: status.keyZ,
  onChange: ({ store, changed }) => store.statuses.delete(changed),
};

export const FLUX_STORE_CONFIG: Flux.UnaryStoreConfig<SubStore> = {
  listeners: [SET_STATUS_LISTENER, DELETE_STATUS_LISTENER],
};

export interface ListParams extends status.MultiRetrieveArgs {}

export const useList = createList<ListParams, status.Key, status.Status, SubStore>({
  name: "Statuses",
  retrieve: async ({ client, params }) => await client.statuses.retrieve(params),
  retrieveByKey: async ({ client, key }) => await client.statuses.retrieve({ key }),
  mountListeners: ({ store, onChange, onDelete, params: { keys } }) => {
    const keysSet = keys ? new Set(keys) : undefined;
    return [
      store.statuses.onSet(async (status) => {
        if (keysSet != null && !keysSet.has(status.key)) return;
        onChange(status.key, status, { mode: "prepend" });
      }),
      store.statuses.onDelete(async (key) => onDelete(key)),
    ];
  },
});

export const { useUpdate: useDelete } = createUpdate<
  status.Key | status.Key[],
  SubStore
>({
  name: "Status",
  update: async ({ client, value }) => await client.statuses.delete(value),
});

export interface SetArgs {
  statuses: status.New | status.New[];
  parent?: ontology.ID;
}

export const { useUpdate: useSet } = createUpdate<SetArgs, SubStore>({
  name: "Status",
  update: async ({ client, value: { statuses, parent } }) => {
    if (Array.isArray(statuses)) await client.statuses.set(statuses, { parent });
    else await client.statuses.set(statuses, { parent });
  },
});

interface UseRetrieveArgs extends status.SingleRetrieveArgs {}

const cachedSingleRetrieve = async ({
  store,
  client,
  params,
}: RetrieveArgs<status.SingleRetrieveArgs, SubStore>) => {
  const cached = store.statuses.get(params.key);
  if (cached != null) return cached;
  const res = await client.statuses.retrieve(params);
  store.statuses.set(params.key, res);
  return res;
};

export const { useRetrieve } = createRetrieve<UseRetrieveArgs, status.Status, SubStore>(
  {
    name: "Status",
    retrieve: cachedSingleRetrieve,
    mountListeners: ({ store, params: { key }, onChange }) => [
      store.statuses.onSet(onChange, key),
    ],
  },
);

export const useSetSynchronizer = (onSet: (status: status.Status) => void): void => {
  const store = Flux.useStore<SubStore>();
  useEffect(() => store.statuses.onSet(onSet), [store]);
};

export const formSchema = status.statusZ;

const INITIAL_VALUES: z.infer<typeof formSchema> = {
  key: "",
  variant: "success",
  message: "",
  time: TimeStamp.now(),
  name: "",
  description: "",
  details: undefined,
};

export const useForm = createForm<
  Partial<UseRetrieveArgs>,
  typeof formSchema,
  SubStore
>({
  name: "Status",
  schema: formSchema,
  initialValues: INITIAL_VALUES,
  retrieve: async ({ reset, ...args }) => {
    const { params } = args;
    if (!("key" in params) || params.key == null) return;
    const res = await cachedSingleRetrieve({
      ...args,
      params: params as UseRetrieveArgs,
    });
    reset(res);
  },
  update: async ({ client, value }) => {
    const v = value();
    console.log(v);
    await client.statuses.set(v);
  },
  mountListeners: ({ store, params: { key }, reset }) => [
    store.statuses.onSet(reset, key),
  ],
});
