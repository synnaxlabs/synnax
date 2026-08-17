// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, DataType, type group, query, type ranger } from "@synnaxlabs/client";
import {
  array,
  control,
  type optional,
  primitive,
  TimeSpan,
  verbs,
} from "@synnaxlabs/x";
import { z } from "zod";

import {
  PLURAL_RESOURCE_NAME,
  RESOURCE_NAME,
  retrieveDefinition,
  retrieveMultipleDefinition,
  type RetrieveMultipleQuery,
  type RetrieveQuery,
} from "@/channel/aether/queries";
import { Flux } from "@/flux";

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

export {
  type RetrieveMultipleQuery,
  type RetrieveQuery,
} from "@/channel/aether/queries";

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

export const { use, useResult, useEnsure, createSelector, createResultSelector } =
  Flux.createRetrieve<RetrieveQuery, channel.Channel>(retrieveDefinition);

/** The channel's range-scoped alias when one is set, its name otherwise. */
export const useAlias = createSelector(({ alias, name }) =>
  primitive.isNonZero(alias) ? alias : name,
);

/** {@link useAlias} with the result contract: fetch on cold, never throw. */
export const useResultAlias = createResultSelector(({ alias, name }) =>
  primitive.isNonZero(alias) ? alias : name,
);

export const useResultName = createResultSelector(({ name }) => name);

/** Compared by variant and message: a heartbeat that changes neither is silenced. */
export const useResultStatus: Flux.UseResult<
  RetrieveQuery,
  channel.Status | undefined
> = createResultSelector(
  ({ status }) => status,
  (a, b) => a?.variant === b?.variant && a?.message === b?.message,
);

export const useResultDataType = createResultSelector(
  ({ dataType }) => dataType,
  (a, b) => a.equals(b),
);

/** The stored range-scoped alias (undefined when unset) alongside the name. */
export const useResultAliasAndName = createResultSelector(
  ({ alias, name }) => ({ alias, name }),
  (a, b) => a.alias === b.alias && a.name === b.name,
);

export const { use: useMultiple, useResult: useResultMultiple } = Flux.createRetrieve<
  RetrieveMultipleQuery,
  channel.Channel[]
>(retrieveMultipleDefinition);

const retrieveInitialFormValues = async ({
  query: { key, rangeKey },
  client,
}: Flux.RetrieveParams<FormQuery>) =>
  channelToFormValues(await client.channels.retrieve(key, { rangeKey }));

const getCachedFormValues = ({
  client,
  query: { key, rangeKey },
}: Flux.RetrieveParams<FormQuery>) => {
  const cached = client.channels.getCached({ key, rangeKey });
  return query.isLive(cached) ? channelToFormValues(cached) : undefined;
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

export type FormQuery = RetrieveQuery;

const formMountListeners: Flux.CreateFormParams<
  FormQuery,
  typeof formSchema | typeof calculatedFormSchema
>["mountListeners"] = ({ client, query: { key, rangeKey }, reset }) =>
  client.channels.onChange({ key, rangeKey }, (result) => {
    if (query.isLive(result)) reset(channelToFormValues(result));
  });

export const useForm = Flux.createForm<FormQuery, typeof formSchema>({
  name: RESOURCE_NAME,
  schema: formSchema,
  initialValues: ZERO_FORM_VALUES,
  retrieve: retrieveInitialFormValues,
  getCached: getCachedFormValues,
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
  getCached: getCachedFormValues,
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
  normalizeQuery: (query) => ({ ...DEFAULT_LIST_PARAMS, ...query }),
  retrieve: async ({ client, query }) => await client.channels.retrieve(query),
  retrieveByKey: async ({ client, key, query: { rangeKey } }) =>
    await client.channels.retrieve(key, { rangeKey }),
  onChange: ({ client, query }, handler) => client.channels.onChange(query, handler),
  getCached: ({ client, query }) => client.channels.getCached(query),
});

export interface RenameParams extends Pick<channel.Payload, "key" | "name"> {}

export const { useUpdate: useRename } = Flux.createUpdate<RenameParams>({
  name: RESOURCE_NAME,
  verbs: verbs.RENAME,
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
  verbs: verbs.UPDATE,
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
  verbs: verbs.DELETE,
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
  verbs: verbs.DELETE,
  update: async ({ client, data }) => {
    const { range, channels } = data;
    if (range == null || channels == null) return false;
    await client.ranges.deleteAlias(range, array.toArray(channels));
    return data;
  },
});

type RetrieveGroupQuery = Record<string, never>;

export const { use: useGroup } = Flux.createRetrieve<RetrieveGroupQuery, group.Group>({
  name: "channel group",
  retrieve: async ({ client }) => await client.channels.retrieveGroup(),
});
