// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, DataType, type group, query, type ranger } from "@synnaxlabs/client";
import { array, control, type optional, TimeSpan } from "@synnaxlabs/x";
import { z } from "zod";

import { Flux } from "@/flux";

const RESOURCE_NAME = "channel";
const PLURAL_RESOURCE_NAME = "channels";

export const formSchema = channel.payloadZ
  .required({ expression: true })
  .extend({
    name: channel.nameZ,
    dataType: DataType.z.transform((v) => v.toString()),
    requires: channel.keyZ.array().optional(),
  })
  .refine(
    (v) => !v.isIndex || DataType.z.parse(v.dataType).equals(DataType.TIMESTAMP),
    {
      message: "Index channel must have data type TIMESTAMP",
      path: ["dataType"],
    },
  )
  .refine((v) => v.isIndex || v.index !== 0 || v.virtual || v.expression !== "", {
    message: "Data channel must have an index",
    path: ["index"],
  });

export const calculatedFormSchema = formSchema.safeExtend({
  expression: z
    .string()
    .min(1, "Expression must not be empty")
    .refine((v) => v.includes("return"), {
      message: "Expression must contain a return statement",
    }),
});

const channelToFormValues = (ch: channel.Channel) => ({
  ...ch.payload,
  dataType: ch.dataType.toString(),
});

export type RetrieveQuery = {
  key: channel.Key;
  rangeKey?: ranger.Key;
};

export const ZERO_FORM_VALUES: z.infer<
  typeof formSchema | typeof calculatedFormSchema
> = {
  key: 0,
  name: "",
  index: 0,
  dataType: DataType.FLOAT32.toString(),
  internal: false,
  isIndex: false,
  leaseholder: 0,
  virtual: false,
  expression: "",
  concurrency: control.Concurrency.exclusive,
  operations: [
    {
      type: "none",
      resetChannel: 0,
      duration: TimeSpan.ZERO,
    },
  ],
};

export const { useRetrieve, useRetrieveStateful, useRetrieveObservable } =
  Flux.createRetrieve<RetrieveQuery, channel.Channel>({
    name: RESOURCE_NAME,
    retrieve: async ({ client, query: { key, rangeKey } }) =>
      await client.channels.retrieve(key, { rangeKey }),
    subscribe: ({ client, query }, handler) => client.channels.onChange(query, handler),
    getCached: ({ client, query }) => client.channels.getCached(query),
  });

export type RetrieveMultipleQuery = channel.RetrieveOptions & {
  keys: channel.Key[];
};

export const { useRetrieve: useRetrieveMultiple } = Flux.createRetrieve<
  RetrieveMultipleQuery,
  channel.Channel[]
>({
  name: PLURAL_RESOURCE_NAME,
  retrieve: async ({ client, query }) => await client.channels.retrieve(query),
  subscribe: ({ client, query }, handler) => client.channels.onChange(query, handler),
  getCached: ({ client, query }) => client.channels.getCached(query),
});

const retrieveInitialFormValues = async ({
  query: { key, rangeKey },
  client,
  reset,
}: Flux.FormRetrieveParams<
  FormQuery,
  typeof formSchema | typeof calculatedFormSchema
>) => {
  if (key == null) return;
  reset(channelToFormValues(await client.channels.retrieve(key, { rangeKey })));
};

const updateForm = async ({
  client,
  set,
  value,
}: Flux.FormUpdateParams<typeof formSchema | typeof calculatedFormSchema>) => {
  const values = value();
  if (values.requires != null) delete values.requires;
  const ch = await client.channels.create(values);
  set("key", ch.key);
};

export type FormQuery = optional.Optional<RetrieveQuery, "key">;

const formMountListeners: Flux.CreateFormParams<
  FormQuery,
  typeof formSchema | typeof calculatedFormSchema
>["mountListeners"] = ({ client, query: { key, rangeKey }, reset }) => {
  if (key == null) return [];
  return client.channels.onChange({ key, rangeKey }, (result) => {
    if (query.isLive(result)) reset(channelToFormValues(result));
  });
};

export const useForm = Flux.createForm<FormQuery, typeof formSchema>({
  name: RESOURCE_NAME,
  schema: formSchema,
  initialValues: ZERO_FORM_VALUES,
  retrieve: retrieveInitialFormValues,
  update: updateForm,
  mountListeners: formMountListeners,
});

export const useCalculatedForm = Flux.createForm<
  FormQuery,
  typeof calculatedFormSchema
>({
  name: "calculated channel",
  schema: calculatedFormSchema,
  initialValues: ZERO_FORM_VALUES,
  retrieve: retrieveInitialFormValues,
  update: updateForm,
  mountListeners: formMountListeners,
});

export type ListQuery = channel.RetrieveOptions & {
  searchTerm?: string;
  rangeKey?: string;
  internal?: boolean;
  offset?: number;
  limit?: number;
};

const DEFAULT_LIST_PARAMS: ListQuery = {
  internal: false,
};

export const useList = Flux.createList<ListQuery, channel.Key, channel.Channel>({
  name: PLURAL_RESOURCE_NAME,
  retrieve: async ({ client, query }) =>
    await client.channels.retrieve({ ...DEFAULT_LIST_PARAMS, ...query }),
  retrieveByKey: async ({ client, key, query: { rangeKey } }) =>
    await client.channels.retrieve(key, { rangeKey }),
  subscribe: ({ client, query }, handler) =>
    client.channels.onChange({ ...DEFAULT_LIST_PARAMS, ...query }, handler),
  getCached: ({ client, query }) =>
    client.channels.getCached({ ...DEFAULT_LIST_PARAMS, ...query }),
});

export interface RenameParams extends Pick<channel.Payload, "key" | "name"> {}

export const { useUpdate: useRename } = Flux.createUpdate<RenameParams>({
  name: RESOURCE_NAME,
  verbs: Flux.RENAME_VERBS,
  update: async ({ client, data, onOptimisticComplete }) => {
    const { key, name } = data;
    await client.channels.rename(key, name, {
      onOptimistic: async () => await onOptimisticComplete(data),
    });
    return data;
  },
});

const ALIAS_RESOURCE_NAME = "channel alias";

export interface UpdateAliasParams extends optional.Optional<
  ranger.alias.Alias,
  "range" | "channel"
> {
  alias: string;
}

export const { useUpdate: useUpdateAlias } = Flux.createUpdate<UpdateAliasParams>({
  name: ALIAS_RESOURCE_NAME,
  verbs: Flux.UPDATE_VERBS,
  update: async ({ client, data }) => {
    const { range, channel, alias } = data;
    if (range == null || channel == null) return false;
    await client.ranges.setAlias(range, channel, alias);
    return data;
  },
});

export type DeleteParams = channel.Key | channel.Key[];

export const { useUpdate: useDelete } = Flux.createUpdate<DeleteParams>({
  name: RESOURCE_NAME,
  verbs: Flux.DELETE_VERBS,
  update: async ({ client, data, onOptimisticComplete }) => {
    await client.channels.delete(data, {
      onOptimistic: async () => await onOptimisticComplete(data),
    });
    return data;
  },
});

export interface DeleteAliasParams {
  range?: ranger.Key;
  channels?: channel.Key | channel.Key[];
}

export const { useUpdate: useDeleteAlias } = Flux.createUpdate<DeleteAliasParams>({
  name: ALIAS_RESOURCE_NAME,
  verbs: Flux.DELETE_VERBS,
  update: async ({ client, data }) => {
    const { range, channels } = data;
    if (range == null || channels == null) return false;
    await client.ranges.deleteAlias(range, array.toArray(channels));
    return data;
  },
});

type RetrieveGroupQuery = Record<string, never>;

export const { useRetrieve: useRetrieveGroup } = Flux.createRetrieve<
  RetrieveGroupQuery,
  group.Group
>({
  name: "Channel Group",
  retrieve: async ({ client }) => await client.channels.retrieveGroup(),
});
