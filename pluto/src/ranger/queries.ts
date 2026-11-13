// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { label, ranger } from "@synnaxlabs/client";
import {
  color,
  deep,
  type NumericTimeRange,
  primitive,
  unique,
  verbs,
} from "@synnaxlabs/x";
import { useEffect } from "react";
import { z } from "zod";

import { Flux } from "@/flux";
import { Label } from "@/label";
import { Synnax } from "@/synnax";

const RESOURCE_NAME = "range";
const PLURAL_RESOURCE_NAME = "ranges";
const KV_RESOURCE_NAME = "metadata";
const PLURAL_KV_RESOURCE_NAME = "metadata";
const PARENT_RESOURCE_NAME = "parent range";
const PLURAL_PARENT_RESOURCE_NAME = "parent ranges";
const PLURAL_ALIAS_RESOURCE_NAME = "channel aliases";

export type RetrieveQuery = {
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

export const { useResult: useResultParent } = Flux.createRetrieve<
  RetrieveQuery,
  ranger.Range | null
>({
  name: PARENT_RESOURCE_NAME,
  retrieve: async ({ client, query: { key } }) =>
    await client.ranges.parent.retrieve(key),
  onChange: ({ client, query: { key } }, handler) =>
    client.ranges.parent.watch(key).onChange(handler),
  getCached: ({ client, query: { key } }) =>
    client.ranges.parent.watch(key).getCached(),
});

export type AncestorsQuery = {
  keys: ranger.Key[];
};

/**
 * The parent of each of the ranges and of each range above them: a key maps to the key
 * of its parent, or to null at the top. It reads one level at a time.
 */
export const { useResult: useResultAncestors } = Flux.createRetrieve<
  AncestorsQuery,
  Record<ranger.Key, ranger.Key | null>
>({
  name: PLURAL_PARENT_RESOURCE_NAME,
  retrieve: async ({ client, query: { keys } }) => {
    const links: Record<ranger.Key, ranger.Key | null> = {};
    let level = keys;
    while (level.length > 0) {
      const ranges = await client.ranges.retrieve(level);
      ranges.forEach(({ key, parent }) => (links[key] = parent ?? null));
      level = unique.unique(
        ranges.flatMap(({ parent }) =>
          parent == null || parent in links ? [] : [parent],
        ),
      );
    }
    return links;
  },
});

export const { use, useEnsure, useTombstone, createSelector, useResult } =
  Flux.createRetrieve<RetrieveQuery, ranger.Range>({
    name: RESOURCE_NAME,
    retrieve: async ({ client, query: { key } }) => await client.ranges.retrieve(key),
    onChange: ({ client, query: { key } }, handler) =>
      client.ranges.watch(key).onChange(handler),
    getCached: ({ client, query: { key } }) => client.ranges.watch(key).getCached(),
  });

export type RetrieveMultipleQuery = {
  keys: ranger.Key[];
};

export const { use: useMultiple, useResult: useResultMultiple } = Flux.createRetrieve<
  RetrieveMultipleQuery,
  ranger.Range[]
>({
  name: PLURAL_RESOURCE_NAME,
  retrieve: async ({ client, query: { keys } }) =>
    await client.ranges.retrieve({ key: keys }),
  onChange: ({ client, query: { keys } }, handler) =>
    client.ranges.watch(keys).onChange(handler),
  getCached: ({ client, query: { keys } }) => client.ranges.watch(keys).getCached(),
});

const timeRangeZ = z
  .object({ start: z.number(), end: z.number() })
  .refine(({ start, end }) => end >= start, {
    error: "End time must be after start time",
  });

export const createFormSchema = z.object({
  ...ranger.newZ.omit({ timeRange: true }).shape,
  labels: z.array(label.keyZ),
  parent: z.string().optional(),
  timeRange: timeRangeZ,
});

export const createFormPath = deep.path<z.infer<typeof createFormSchema>>();

const ZERO_CREATE_FORM_VALUES: z.infer<typeof createFormSchema> = {
  name: "",
  labels: [],
  parent: "",
  timeRange: { start: 0, end: 0 },
};

/** A form that creates a range when the caller saves it. */
export const useCreateForm = Flux.createForm<null, typeof createFormSchema>({
  name: RESOURCE_NAME,
  schema: createFormSchema,
  initialValues: ZERO_CREATE_FORM_VALUES,
  update: async ({ client, value: getValue, reset }) => {
    const value = getValue();
    const rng = await client.ranges.create({
      ...value,
      parent: primitive.isNonZero(value.parent) ? value.parent : undefined,
    });
    reset({
      ...value,
      ...rng.payload,
      timeRange: rng.timeRange.numeric,
      parent: value.parent,
    });
  },
});

export const formSchema = z.object({
  name: z.string().min(1, "Name is required"),
  timeRange: timeRangeZ,
});

export const formPath = deep.path<z.infer<typeof formSchema>>();

/** A form over the name and time range of a cached range. */
export const useForm = Flux.createDocumentForm<
  ranger.Key,
  ranger.Range,
  typeof formSchema
>({
  name: RESOURCE_NAME,
  schema: formSchema,
  domain: (client) => client.ranges,
  useEnsure,
  toValues: ({ name, timeRange }) => ({ name, timeRange: timeRange.numeric }),
});

export const useLabels = (key: ranger.Key | null): label.Label[] | undefined => {
  const labels = useResult(key == null ? null : { key }).data?.labels;
  return Label.useResultMultiple(labels == null ? null : { keys: labels }).data;
};

export const metaDataFormSchema = z.object({
  pairs: z.array(z.object({ key: z.string(), value: z.string() })),
});

export type MetaDataQuery = {
  rangeKey: ranger.Key;
};

export const { useResult: useResultMetaData } = Flux.createRetrieve<
  MetaDataQuery,
  ranger.kv.Pair[]
>({
  name: PLURAL_KV_RESOURCE_NAME,
  retrieve: async ({ client, query: { rangeKey } }) =>
    await client.ranges.kv.retrieve(rangeKey),
  onChange: ({ client, query: { rangeKey } }, handler) =>
    client.ranges.kv.watch(rangeKey).onChange(handler),
  getCached: ({ client, query: { rangeKey } }) =>
    client.ranges.kv.watch(rangeKey).getCached(),
});

export type AliasesQuery = {
  rangeKey: ranger.Key;
};

/** The channel aliases of a range: its own and those of the ranges above it. */
export const { useResult: useResultAliases } = Flux.createRetrieve<
  AliasesQuery,
  ranger.ResolvedAliases
>({
  name: PLURAL_ALIAS_RESOURCE_NAME,
  retrieve: async ({ client, query: { rangeKey } }) =>
    await client.ranges.resolvedAliases.retrieve(rangeKey),
  onChange: ({ client, query: { rangeKey } }, handler) =>
    client.ranges.resolvedAliases.watch(rangeKey).onChange(handler),
  getCached: ({ client, query: { rangeKey } }) =>
    client.ranges.resolvedAliases.watch(rangeKey).getCached(),
});

export const kvPairFormSchema = ranger.kv.pairZ.extend({
  key: z.string().trim().min(1, "Key is required"),
});

export const kvPairFormPath = deep.path<z.infer<typeof kvPairFormSchema>>();

const ZERO_KV_PAIR_FORM_VALUES: z.infer<typeof kvPairFormSchema> = {
  key: "",
  value: "",
  range: "",
};

/** A form that sets a new metadata pair on a range when it saves. */
export const useCreateKVPairForm = Flux.createForm<null, typeof kvPairFormSchema>({
  name: KV_RESOURCE_NAME,
  schema: kvPairFormSchema,
  initialValues: ZERO_KV_PAIR_FORM_VALUES,
  update: async ({ client, value: getPair }) => {
    const { key, value, range } = getPair();
    await client.ranges.getKV(range).set(key, value);
  },
});

export interface DeleteKVParams extends MetaDataQuery {
  key: string;
}

export const { useUpdate: useDeleteKV } = Flux.createUpdate<DeleteKVParams>({
  name: KV_RESOURCE_NAME,
  verbs: verbs.DELETE,
  update: async ({ client, data, onOptimisticComplete }) => {
    const { key, rangeKey } = data;
    await client.ranges.getKV(rangeKey).delete(key, {
      onOptimistic: async () => await onOptimisticComplete(data),
    });
    return data;
  },
});

export interface SetKVParams extends MetaDataQuery, ranger.kv.Pair {}

export const { useUpdate: useUpdateKV } = Flux.createUpdate<SetKVParams>({
  name: KV_RESOURCE_NAME,
  verbs: verbs.UPDATE,
  update: async ({ client, data, onOptimisticComplete }) => {
    const { range, key, value } = data;
    await client.ranges.getKV(range).set(key, value, {
      onOptimistic: async () => await onOptimisticComplete(data),
    });
    return data;
  },
});

export const { useUpdate: useCreate } = Flux.createUpdate<ranger.New, ranger.Payload>({
  name: RESOURCE_NAME,
  verbs: verbs.CREATE,
  update: async ({ client, data }) => (await client.ranges.create(data)).payload,
});

export type DeleteParams = ranger.Key | ranger.Key[];

export const { useUpdate: useDelete } = Flux.createUpdate<DeleteParams>({
  name: RESOURCE_NAME,
  verbs: verbs.DELETE,
  update: async ({ client, data, onOptimisticComplete }) => {
    await client.ranges.delete(data, {
      onOptimistic: async () => await onOptimisticComplete(data),
    });
    return data;
  },
});

export interface RenameParams extends Pick<ranger.Payload, "key" | "name"> {}

export const { useUpdate: useRename } = Flux.createUpdate<RenameParams>({
  name: RESOURCE_NAME,
  verbs: verbs.RENAME,
  update: async ({ client, data }) => {
    const { key, name } = data;
    await client.ranges.rename(key, name);
    return data;
  },
});

export interface SetTimeRangeParams {
  key: ranger.Key;
  timeRange: NumericTimeRange;
}

export const { useUpdate: useSetTimeRange } = Flux.createUpdate<SetTimeRangeParams>({
  name: RESOURCE_NAME,
  verbs: verbs.UPDATE,
  update: async ({ client, data }) => {
    const { key, timeRange } = data;
    await client.ranges.dispatch(
      key,
      ranger.set({ path: ["timeRange"], value: timeRange }),
    );
    return data;
  },
});

export interface SetColorParams {
  key: ranger.Key;
  /** The color of the range. Undefined clears it. */
  color?: color.Color;
}

export const { useUpdate: useSetColor } = Flux.createUpdate<SetColorParams>({
  name: RESOURCE_NAME,
  verbs: verbs.UPDATE,
  update: async ({ client, data }) => {
    const { key, color: value } = data;
    await client.ranges.dispatch(
      key,
      ranger.set({ path: ["color"], value: value == null ? null : color.hex(value) }),
    );
    return data;
  },
});

export interface KeyParams {
  key: ranger.Key;
}

export const useName = createSelector(({ name }) => name);
