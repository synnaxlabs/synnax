// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { label, type ontology, query, type Synnax as Client } from "@synnaxlabs/client";
import { color, verbs } from "@synnaxlabs/x";
import type z from "zod";

import { Flux } from "@/flux";

export const RESOURCE_NAME = "label";
export const PLURAL_RESOURCE_NAME = "labels";

export type RetrieveQuery = label.RetrieveSingleParams;

export type LabelsOfQuery = {
  id: ontology.ID;
};

interface SetLabelsForParams {
  client: Client;
  data: {
    id: ontology.ID;
    labels: label.Key[];
  };
}

export const setLabelsFor = async ({
  client,
  data: { id, labels },
}: SetLabelsForParams): Promise<label.Label[]> => {
  await client.labels.label(id, labels, { replace: true });
  if (labels.length === 0) return [];
  return await client.labels.retrieve({ keys: labels });
};

export const { useRetrieve: useRetrieveLabelsOf } = Flux.createRetrieve<
  LabelsOfQuery,
  label.Label[]
>({
  name: PLURAL_RESOURCE_NAME,
  retrieve: async ({ client, query: { id } }) =>
    await client.labels.retrieve({ for: id }),
  subscribe: ({ client, query: { id } }, handler) =>
    client.labels.onChange({ for: id }, handler),
  getCached: ({ client, query: { id } }) => client.labels.getCached({ for: id }),
});

export type ListQuery = label.RetrieveMultipleParams;

export const useList = Flux.createList<ListQuery, label.Key, label.Label>({
  name: PLURAL_RESOURCE_NAME,
  retrieve: async ({ client, query }) => await client.labels.retrieve(query),
  retrieveByKey: async ({ client, key }) => await client.labels.retrieve(key),
  subscribe: ({ client, query }, handler) => client.labels.onChange(query, handler),
  getCached: ({ client, query }) => client.labels.getCached(query),
});

type FormQuery = {
  key?: label.Key;
};

export const formSchema = label.labelZ.partial({ key: true });

const INITIAL_VALUES: z.infer<typeof formSchema> = {
  name: "",
  color: color.construct("#000000"),
};

export const useForm = Flux.createForm<FormQuery, typeof formSchema>({
  name: RESOURCE_NAME,
  initialValues: INITIAL_VALUES,
  schema: formSchema,
  retrieve: async ({ client, query: { key }, reset }) => {
    if (key == null) return;
    reset(await client.labels.retrieve(key));
  },
  update: async ({ client, value, reset }) => {
    const updated = await client.labels.create(value());
    reset(updated);
  },
  mountListeners: ({ client, query: { key }, reset }) => {
    if (key == null) return [];
    return client.labels.onChange(key, (result) => {
      if (query.isLive(result)) reset(result);
    });
  },
});

export type DeleteParams = label.Key | label.Key[];

export const { useUpdate: useDelete } = Flux.createUpdate<DeleteParams>({
  name: RESOURCE_NAME,
  verbs: verbs.DELETE,
  update: async ({ client, data }) => {
    await client.labels.delete(data);
    return data;
  },
});

export type RetrieveMultipleParams = {
  keys: label.Key[];
};

export const { useRetrieve: useRetrieveMultiple } = Flux.createRetrieve<
  RetrieveMultipleParams,
  label.Label[]
>({
  name: PLURAL_RESOURCE_NAME,
  retrieve: async ({ client, query }) => await client.labels.retrieve(query),
  subscribe: ({ client, query }, handler) => client.labels.onChange(query, handler),
  getCached: ({ client, query }) => client.labels.getCached(query),
});
