// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { label, type ontology, type Synnax as Client } from "@synnaxlabs/client";
import { color, verbs } from "@synnaxlabs/x";
import type z from "zod";

import { Flux } from "@/flux";

const RESOURCE_NAME = "label";
const PLURAL_RESOURCE_NAME = "labels";

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

export const { useResult: useResultLabelsOf } = Flux.createRetrieve<
  LabelsOfQuery,
  label.Label[]
>({
  name: PLURAL_RESOURCE_NAME,
  retrieve: async ({ client, query: { id } }) =>
    await client.labels.retrieve({ for: id }),
  onChange: ({ client, query: { id } }, handler) =>
    client.labels.onChange({ for: id }, handler),
  getCached: ({ client, query: { id } }) => client.labels.getCached({ for: id }),
});

export interface EditForParams {
  id: ontology.ID;
  added: label.Key[];
  removed: label.Key[];
}

/**
 * Adds and removes labels on a resource. Other labels on the resource stay in place, so
 * edits from other clients at the same time are kept.
 */
export const { useUpdate: useEditFor } = Flux.createUpdate<EditForParams>({
  name: PLURAL_RESOURCE_NAME,
  verbs: verbs.UPDATE,
  update: async ({ client, data }) => {
    const { id, added, removed } = data;
    if (added.length > 0) await client.labels.label(id, added);
    if (removed.length > 0) await client.labels.remove(id, removed);
    return data;
  },
});

export type ListQuery = label.RetrieveMultipleParams;

export const useList = Flux.createList<ListQuery, label.Key, label.Label>({
  name: PLURAL_RESOURCE_NAME,
  retrieve: async ({ client, query }) => await client.labels.retrieve(query),
  retrieveByKey: async ({ client, key }) => await client.labels.retrieve(key),
  onChange: ({ client, query }, handler) => client.labels.onChange(query, handler),
  onChangeByKey: ({ client, key }, handler) => client.labels.onChange(key, handler),
  getCached: ({ client, query }) => client.labels.getCached(query),
});

const { useEnsure } = Flux.createRetrieve<RetrieveQuery, label.Label>({
  name: RESOURCE_NAME,
  retrieve: async ({ client, query: { key } }) => await client.labels.retrieve(key),
  onChange: ({ client, query: { key } }, handler) =>
    client.labels.onChange(key, handler),
  getCached: ({ client, query: { key } }) => client.labels.getCached(key),
});

export const createFormSchema = label.labelZ.partial({ key: true });

const ZERO_CREATE_FORM_VALUES: z.infer<typeof createFormSchema> = {
  name: "",
  color: color.construct("#000000"),
};

/** A form that creates a label when the caller saves it. */
export const useCreateForm = Flux.createForm<null, typeof createFormSchema>({
  name: RESOURCE_NAME,
  initialValues: ZERO_CREATE_FORM_VALUES,
  schema: createFormSchema,
  update: async ({ client, value, reset }) => {
    reset(await client.labels.create(value()));
  },
});

export const formSchema = label.labelZ.pick({ name: true, color: true });

/** A form over the name and color of a cached label. */
export const useForm = Flux.createDocumentForm<
  label.Key,
  label.Label,
  typeof formSchema
>({
  name: RESOURCE_NAME,
  schema: formSchema,
  domain: (client) => client.labels,
  useEnsure,
  toValues: ({ name, color }) => ({ name, color }),
});

export type DeleteParams = label.Key | label.Key[];

export const { useUpdate: useDelete } = Flux.createUpdate<DeleteParams>({
  name: RESOURCE_NAME,
  verbs: verbs.DELETE,
  update: async ({ client, data, onOptimisticComplete }) => {
    await client.labels.delete(data, {
      onOptimistic: async () => await onOptimisticComplete(data),
    });
    return data;
  },
});

export type RetrieveMultipleParams = {
  keys: label.Key[];
};

export const { use: useMultiple, useResult: useResultMultiple } = Flux.createRetrieve<
  RetrieveMultipleParams,
  label.Label[]
>({
  name: PLURAL_RESOURCE_NAME,
  retrieve: async ({ client, query }) => await client.labels.retrieve(query),
  onChange: ({ client, query }, handler) => client.labels.onChange(query, handler),
  getCached: ({ client, query }) => client.labels.getCached(query),
});
