// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { actions, channel, DataType, query } from "@synnaxlabs/client";
import { control, deep, primitive, TimeSpan, verbs } from "@synnaxlabs/x";
import { z } from "zod";

import {
  PLURAL_RESOURCE_NAME,
  RESOURCE_NAME,
  retrieveDefinition,
  retrieveMultipleDefinition,
  type RetrieveMultipleQuery,
  type RetrieveQuery,
} from "@/channel/aether/queries";
import { resolveIcon } from "@/channel/resolveIcon";
import { Flux } from "@/flux";
import { flux } from "@/flux/aether";

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

export const formPath = deep.path<z.infer<typeof formSchema>>();

export const calculatedFormSchema = formSchema.safeExtend({
  expression: z
    .string()
    .min(1, "Expression must not be empty")
    .refine((v) => v.includes("return"), {
      message: "Expression must contain a return statement",
    }),
});

export const calculatedFormPath = deep.path<z.infer<typeof calculatedFormSchema>>();

const ZERO_OPERATION: channel.Operation = {
  type: "none",
  resetChannel: 0,
  duration: TimeSpan.ZERO,
};

// The form edits operations[0], so a channel without operations loads as "none".
const channelToFormValues = (ch: channel.Channel) => ({
  ...ch.payload,
  dataType: ch.dataType.toString(),
  operations: ch.operations.length > 0 ? ch.operations : [ZERO_OPERATION],
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
  operations: [ZERO_OPERATION],
};

export const { use, useResult, useEnsure, createSelector, createResultSelector } =
  Flux.createRetrieve<RetrieveQuery, channel.Channel>(retrieveDefinition);

export const useResultName = createResultSelector(({ name }) => name);

export const useResultIsIndex = createResultSelector(({ isIndex }) => isIndex);

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

/**
 * The icon of the channel, and whether it is internal or virtual. Compared by value: a
 * status heartbeat is silenced.
 */
export const useResultKind = createResultSelector(
  (ch) => ({
    icon: resolveIcon(ch.payload),
    internal: ch.internal,
    virtual: ch.virtual,
  }),
  (a, b) => a.icon === b.icon && a.internal === b.internal && a.virtual === b.virtual,
);

export const { use: useMultiple, useResult: useResultMultiple } = Flux.createRetrieve<
  RetrieveMultipleQuery,
  channel.Channel[]
>(retrieveMultipleDefinition);

const retrieveInitialFormValues = async ({
  query: { key },
  client,
}: Flux.RetrieveParams<FormQuery>) =>
  channelToFormValues(await client.channels.retrieve(key));

const getCachedFormValues = ({
  client,
  query: { key },
}: Flux.RetrieveParams<FormQuery>) => {
  const cached = client.channels.watch({ key }).getCached();
  return query.isLive(cached) ? channelToFormValues(cached) : undefined;
};

const updateForm = async ({
  client,
  set,
  value,
  stored,
}: Flux.FormUpdateParams<typeof formSchema | typeof calculatedFormSchema>) => {
  const values = value();
  if (values.requires != null) delete values.requires;
  if (primitive.isZero(values.key)) {
    const ch = await client.channels.create(values);
    set(formPath.key, ch.key);
    return;
  }
  const edits: channel.Action[] = actions.diffFields(channel.payloadZ, stored, values, [
    "expression",
    "operations",
  ]);
  if (stored?.name !== values.name)
    edits.unshift(channel.rename({ name: values.name }));
  if (edits.length > 0) await client.channels.dispatch(values.key, edits);
};

export type FormQuery = RetrieveQuery;

const formMountListeners: Flux.CreateFormParams<
  FormQuery,
  typeof formSchema | typeof calculatedFormSchema
>["mountListeners"] = ({ client, query: { key }, reset }) =>
  client.channels.watch({ key }).onChange((result) => {
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

export type ListQuery = channel.RetrieveMultipleParams;

const normalizeListQuery = <Q extends ListQuery>(query: Q): Q => ({
  ...query,
  where: { internal: { eq: false }, ...query.where },
});

/** Lists the channels that are not internal, unless the query sets `where.internal`. */
export const useList = ({
  query,
  ...rest
}: Omit<
  Flux.UseListParams<ListQuery, channel.Channel>,
  "itemQuery" | "name"
> = {}): Flux.UseListReturn<channel.Key, channel.Channel> =>
  Flux.useList((client) => client.channels, {
    ...rest,
    name: PLURAL_RESOURCE_NAME,
    query: query === null ? null : normalizeListQuery(query ?? {}),
    itemQuery: (key) => ({ key }),
  });

export const { useResult: useResultList } = Flux.createRetrieve<
  ListQuery,
  channel.Channel[]
>({
  name: PLURAL_RESOURCE_NAME,
  normalizeQuery: normalizeListQuery,
  ...flux.listReads(({ client, query }) => client.channels.watch(query)),
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
