// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type cache,
  label,
  type ontology,
  status,
  TimeStamp,
} from "@synnaxlabs/client";
import { primitive, uuid } from "@synnaxlabs/x";
import { useEffect } from "react";
import type z from "zod";

import { Flux } from "@/flux";
import { Label } from "@/label";
import { Synnax } from "@/synnax";

const RESOURCE_NAME = "status";
const PLURAL_RESOURCE_NAME = "statuses";

export type ListParams = status.MultiRetrieveParams;

export const useList = Flux.createList<ListParams, status.Key, status.Status>({
  name: PLURAL_RESOURCE_NAME,
  retrieve: async ({ client, query }) => await client.statuses.retrieve(query),
  retrieveByKey: async ({ client, key }) => await client.statuses.retrieve({ key }),
  subscribe: ({ client, query }, handler) => client.statuses.onChange(query, handler),
  getCached: ({ client, query }) => client.statuses.getCached(query),
});

export const { useUpdate: useDelete } = Flux.createUpdate<status.Key | status.Key[]>({
  name: RESOURCE_NAME,
  verbs: Flux.DELETE_VERBS,
  update: async ({ client, data }) => {
    await client.statuses.delete(data);
    return data;
  },
});

export interface SetParams {
  statuses: status.New | status.New[];
  parent?: ontology.ID;
}

export const { useUpdate: useSet } = Flux.createUpdate<SetParams>({
  name: RESOURCE_NAME,
  verbs: Flux.SET_VERBS,
  update: async ({ client, data, data: { statuses, parent } }) => {
    if (Array.isArray(statuses)) await client.statuses.set(statuses, { parent });
    else await client.statuses.set(statuses, { parent });
    return data;
  },
});

export type RetrieveQuery = status.SingleRetrieveParams;

const BASE_QUERY: Pick<RetrieveQuery, "includeLabels"> = {
  includeLabels: true,
};

// Cached answers are untyped; a details schema only validates the fetch, so
// schema-typed reads cast the shared cache entries.
export const createRetrieve = <DetailsSchema extends z.ZodType = z.ZodNever>(
  detailsSchema?: DetailsSchema,
) =>
  Flux.createRetrieve<RetrieveQuery, status.Status<DetailsSchema>>({
    name: RESOURCE_NAME,
    retrieve: async ({ client, query }) =>
      await client.statuses.retrieve({ ...BASE_QUERY, ...query, detailsSchema }),
    subscribe: ({ client, query }, handler) =>
      client.statuses.onChange(query, handler as cache.ChangeHandler<status.Status>),
    getCached: ({ client, query }) =>
      client.statuses.getCached(query) as
        cache.Cached<status.Status<DetailsSchema>> | undefined,
  });

export type RetrieveMultipleQuery = {
  keys: status.Key[];
};

export const { useRetrieve: useRetrieveMultiple } = Flux.createRetrieve<
  RetrieveMultipleQuery,
  status.Status[]
>({
  name: PLURAL_RESOURCE_NAME,
  retrieve: async ({ client, query: { keys } }) => {
    if (keys.length === 0) return [];
    return await client.statuses.retrieve({ keys });
  },
  subscribe: ({ client, query: { keys } }, handler) => {
    if (keys.length === 0) return () => {};
    return client.statuses.onChange({ keys }, handler);
  },
  getCached: ({ client, query: { keys } }) => {
    if (keys.length === 0) return undefined;
    return client.statuses.getCached({ keys });
  },
});

export const { useRetrieve } = createRetrieve();

export const useSetSynchronizer = (onSet: (status: status.Status) => void): void => {
  const client = Synnax.use();
  useEffect(() => client?.statuses.onSet(onSet), [client, onSet]);
};

export const useDeleteSynchronizer = (onDelete: (key: status.Key) => void): void => {
  const client = Synnax.use();
  useEffect(() => client?.statuses.onDelete(onDelete), [client, onDelete]);
};

export const formSchema = status.statusZ().omit({ labels: true }).safeExtend({
  labels: label.keyZ.array().optional(),
});

const INITIAL_VALUES: z.infer<typeof formSchema> = {
  key: "",
  variant: "success",
  message: "",
  time: TimeStamp.now(),
  name: "",
  description: "",
  labels: [],
};

export const useForm = Flux.createForm<Partial<RetrieveQuery>, typeof formSchema>({
  name: RESOURCE_NAME,
  schema: formSchema,
  initialValues: INITIAL_VALUES,
  retrieve: async ({ client, query: { key }, reset }) => {
    if (primitive.isZero(key)) return;
    const stat = await client.statuses.retrieve({ ...BASE_QUERY, key });
    const labels = await client.labels.retrieve({ for: status.ontologyID(stat.key) });
    reset({ ...stat, labels: labels.map((l) => l.key) });
  },
  update: async ({ client, value, set }) => {
    set("time", TimeStamp.now());
    const v = value();
    if (primitive.isZero(v.key)) v.key = uuid.create();
    const { labels: labelKeys, ...rest } = v;
    const res = await client.statuses.set(rest);
    if (labelKeys != null)
      await Label.setLabelsFor({
        client,
        data: { id: status.ontologyID(res.key), labels: labelKeys },
      });
    set("key", res.key);
  },
  mountListeners: ({ client, query: { key }, reset }) => {
    if (primitive.isZero(key)) return [];
    return client.statuses.onChange({ key }, (result) => {
      if (result?.variant !== "changed") return;
      const { labels, ...rest } = result.data;
      reset({ ...rest, labels: labels?.map((l) => l.key) ?? [] });
    });
  },
});

export interface RenameParams extends Pick<status.Status, "key" | "name"> {}

export const { useUpdate: useRename } = Flux.createUpdate<RenameParams>({
  name: RESOURCE_NAME,
  verbs: Flux.RENAME_VERBS,
  update: async ({ client, data, onOptimisticComplete }) => {
    const { key, name } = data;
    await client.statuses.rename(key, name, {
      onOptimistic: async () => await onOptimisticComplete(data),
    });
    return data;
  },
});
