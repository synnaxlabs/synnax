// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  label,
  NotFoundError,
  type ontology,
  ranger,
  type Synnax as Client,
} from "@synnaxlabs/client";
import { type optional, primitive } from "@synnaxlabs/x";
import { useEffect } from "react";
import { z } from "zod";

import { Flux } from "@/flux";
import { Label } from "@/label";
import { type List } from "@/list";
import { Synnax } from "@/synnax";

const RESOURCE_NAME = "range";
const PLURAL_RESOURCE_NAME = "ranges";
const KV_RESOURCE_NAME = "metadata";
const PLURAL_KV_RESOURCE_NAME = "metadata";
const PLURAL_CHILDREN_RESOURCE_NAME = "child ranges";
const PARENT_RESOURCE_NAME = "parent range";

export type RetrieveQuery = Pick<
  ranger.RetrieveRequest,
  "includeLabels" | "includeParent"
> & {
  key: ranger.Key;
};

export const useSetSynchronizer = (onSet: (range: ranger.Payload) => void): void => {
  const client = Synnax.use();
  useEffect(() => client?.ranges.onSet((range) => onSet(range.payload)), [client]);
};

export const useDeleteSynchronizer = (onDelete: (key: ranger.Key) => void): void => {
  const client = Synnax.use();
  useEffect(() => client?.ranges.onDelete((key) => onDelete(key)), [client]);
};

export type ListChildrenQuery = List.PagerParams & {
  key?: ranger.Key;
};

export const useListChildren = Flux.createList<
  ListChildrenQuery,
  ranger.Key,
  ranger.Range
>({
  name: PLURAL_CHILDREN_RESOURCE_NAME,
  retrieve: async ({ client, query: { key } }) => {
    if (key == null) return [];
    return await client.ranges.children.retrieve(key);
  },
  retrieveByKey: async ({ client, key }) => await client.ranges.retrieve(key),
  subscribe: ({ client, query: { key } }, handler) => {
    if (key == null) return () => {};
    return client.ranges.children.onChange(key, handler);
  },
  getCached: ({ client, query: { key } }) => {
    if (key == null) return undefined;
    return client.ranges.children.getCached(key);
  },
});

export type RetrieveParentQuery = {
  id: ontology.ID;
};

export const {
  useRetrieve: useRetrieveParent,
  useRetrieveEffect: useRetrieveParentEffect,
} = Flux.createRetrieve<RetrieveParentQuery, ranger.Range | null>({
  name: PARENT_RESOURCE_NAME,
  retrieve: async ({ client, query: { id } }) =>
    await client.ranges.parent.retrieve(id),
  subscribe: ({ client, query: { id } }, handler) =>
    client.ranges.parent.onChange(id, handler),
  getCached: ({ client, query: { id } }) => client.ranges.parent.getCached(id),
});

export const {
  useRetrieve,
  useRetrieveObservable,
  useRetrieveSuspended: useRetrieveSuspense,
  useEnsureRetrieved,
} = Flux.createRetrieve<RetrieveQuery, ranger.Range>({
  name: RESOURCE_NAME,
  retrieve: async ({ client, query: { key } }) => await client.ranges.retrieve(key),
  subscribe: ({ client, query: { key } }, handler) =>
    client.ranges.onChange(key, handler),
  getCached: ({ client, query: { key } }) => client.ranges.getCached(key),
});

export type RetrieveMultipleQuery = {
  keys: ranger.Key[];
};

export const {
  useRetrieve: useRetrieveMultiple,
  useRetrieveObservable: useRetrieveObservableMultiple,
} = Flux.createRetrieve<RetrieveMultipleQuery, ranger.Range[]>({
  name: PLURAL_RESOURCE_NAME,
  retrieve: async ({ client, query: { keys } }) => await client.ranges.retrieve(keys),
  subscribe: ({ client, query: { keys } }, handler) =>
    client.ranges.onChange(keys, handler),
  getCached: ({ client, query: { keys } }) => client.ranges.getCached(keys),
});

export const formSchema = z.object({
  ...ranger.payloadZ.omit({ timeRange: true }).partial({ key: true }).shape,
  labels: z.array(label.keyZ),
  parent: z.string().optional(),
  timeRange: z
    .object({ start: z.number(), end: z.number() })
    .refine(({ start, end }) => end >= start, {
      error: "End time must be after start time",
      path: ["end"],
    }),
});

export const toFormValues = (range: ranger.Range): z.infer<typeof formSchema> => ({
  ...range.payload,
  timeRange: range.timeRange.numeric,
  parent: range.parent?.key,
  labels: range.labels?.map((l) => l.key) ?? [],
});

export type FormQuery = optional.Optional<RetrieveQuery, "key">;

const ZERO_FORM_VALUES: z.infer<typeof formSchema> = {
  name: "",
  labels: [],
  parent: "",
  timeRange: { start: 0, end: 0 },
};

export const useForm = Flux.createForm<FormQuery, typeof formSchema>({
  name: RESOURCE_NAME,
  schema: formSchema,
  initialValues: ZERO_FORM_VALUES,
  retrieve: async ({ client, query: { key }, reset }) => {
    if (key == null) return;
    reset(toFormValues(await client.ranges.retrieve(key)));
  },
  update: async ({ client, value: getValue, reset }) => {
    const value = getValue();
    const parentKey = value.parent;
    const rng = await client.ranges.create({
      ...value,
      parent: primitive.isNonZero(parentKey) ? { key: parentKey } : undefined,
    });
    await Label.setLabelsFor({
      client,
      data: { id: rng.ontologyID, labels: value.labels },
    });
    reset({
      ...value,
      ...rng.payload,
      timeRange: rng.timeRange.numeric,
      labels: value.labels,
      parent: value.parent,
    });
  },
  mountListeners: ({ client, query: { key }, reset }) => {
    if (key == null) return [];
    return client.ranges.onChange(key, (result) => {
      if (result?.variant === "changed") reset(toFormValues(result.data));
    });
  },
});

export const useLabels = (
  key: ranger.Key,
  opts?: Omit<
    Flux.UseDirectRetrieveParams<Label.LabelsOfQuery, label.Label[]>,
    "query"
  >,
): Flux.UseDirectRetrieveReturn<label.Label[]> =>
  Label.useRetrieveLabelsOf({ id: ranger.ontologyID(key) }, opts);

export type ListQuery = Omit<ranger.RetrieveRequest, "names">;

export const useList = Flux.createList<ListQuery, ranger.Key, ranger.Range>({
  name: PLURAL_RESOURCE_NAME,
  retrieve: async ({ client, query }) => await client.ranges.retrieve(query),
  retrieveByKey: async ({ client, key }) => await client.ranges.retrieve(key),
  subscribe: ({ client, query }, handler) => client.ranges.onChange(query, handler),
  getCached: ({ client, query }) => client.ranges.getCached(query),
});

export const metaDataFormSchema = z.object({
  pairs: z.array(z.object({ key: z.string(), value: z.string() })),
});

export type ListMetaDataQuery = {
  rangeKey: ranger.Key;
};

export const useListMetaData = Flux.createList<
  ListMetaDataQuery,
  string,
  ranger.kv.Pair
>({
  name: PLURAL_KV_RESOURCE_NAME,
  retrieve: async ({ client, query: { rangeKey } }) =>
    await client.ranges.kv.retrieve(rangeKey),
  retrieveByKey: async ({ client, key, query: { rangeKey } }) => {
    if (rangeKey == null) return undefined;
    const kv = client.ranges.getKV(rangeKey);
    const value = await kv.get(key);
    return { key, value, range: rangeKey };
  },
  subscribe: ({ client, query: { rangeKey } }, handler) =>
    client.ranges.kv.onChange(rangeKey, handler),
  getCached: ({ client, query: { rangeKey } }) => client.ranges.kv.getCached(rangeKey),
});

export const kvPairFormSchema = ranger.kv.pairZ;

export type KVFormQuery = ListMetaDataQuery;

const ZERO_KV_PAIR_FORM_VALUES: z.infer<typeof kvPairFormSchema> = {
  key: "",
  value: "",
  range: "",
};

export const useKVPairForm = Flux.createForm<KVFormQuery, typeof kvPairFormSchema>({
  name: KV_RESOURCE_NAME,
  schema: kvPairFormSchema,
  retrieve: async () => undefined,
  initialValues: ZERO_KV_PAIR_FORM_VALUES,
  update: async ({ client, value: getPair }) => {
    const { key, value, range } = getPair();
    await client.ranges.getKV(range).set(key, value);
  },
});

export interface DeleteKVParams extends ListMetaDataQuery {
  key: string;
}

export const { useUpdate: useDeleteKV } = Flux.createUpdate<DeleteKVParams>({
  name: KV_RESOURCE_NAME,
  verbs: Flux.DELETE_VERBS,
  update: async ({ client, data }) => {
    const { key, rangeKey } = data;
    await client.ranges.getKV(rangeKey).delete(key);
    return data;
  },
});

export interface SetKVParams extends ListMetaDataQuery, ranger.kv.Pair {}

export const { useUpdate: useUpdateKV } = Flux.createUpdate<SetKVParams>({
  name: KV_RESOURCE_NAME,
  verbs: Flux.UPDATE_VERBS,
  update: async ({ client, data }) => {
    const { range, key, value } = data;
    await client.ranges.getKV(range).set(key, value);
    return data;
  },
});

export const { useUpdate: useCreate } = Flux.createUpdate<ranger.New, ranger.Payload>({
  name: RESOURCE_NAME,
  verbs: Flux.CREATE_VERBS,
  update: async ({ client, data }) => (await client.ranges.create(data)).payload,
});

export type DeleteParams = ranger.Key | ranger.Key[];

export const { useUpdate: useDelete } = Flux.createUpdate<DeleteParams>({
  name: RESOURCE_NAME,
  verbs: Flux.DELETE_VERBS,
  update: async ({ client, data }) => {
    await client.ranges.delete(data);
    return data;
  },
});

export interface RenameParams extends Pick<ranger.Payload, "key" | "name"> {}

export const { useUpdate: useRename } = Flux.createUpdate<RenameParams>({
  name: RESOURCE_NAME,
  verbs: Flux.RENAME_VERBS,
  update: async ({ client, data, onOptimisticComplete }) => {
    const { key, name } = data;
    await client.ranges.rename(key, name, {
      onOptimistic: async () => await onOptimisticComplete(data),
    });
    return data;
  },
});

const requireRange = (client: Client | null, key: ranger.Key): ranger.Range => {
  const cached = client?.ranges.getCached(key);
  if (cached == null || cached.variant === "deleted")
    throw new NotFoundError(`Range with key ${key} not found`);
  return cached.data;
};

export interface SelectKeyParams {
  key: ranger.Key;
}

export const [useSelectName, useGetName] = Flux.createSelector<SelectKeyParams, string>(
  {
    subscribe: ({ client, args: { key } }, notify) =>
      client == null ? () => {} : client.ranges.onChange(key, notify),
    select: ({ client, args: { key } }) => requireRange(client, key).name,
  },
);
