// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { icd } from "@synnaxlabs/client";
import { deep, verbs } from "@synnaxlabs/x";
import { type z } from "zod";

import { Flux } from "@/flux";

const RESOURCE_NAME = "ICD";
const PLURAL_RESOURCE_NAME = "ICDs";

const IMPORT_VERBS: verbs.Verbs = {
  present: "import",
  participle: "importing",
  past: "imported",
};

export type RetrieveQuery = {
  key: icd.Key;
};

export const { use, useResult, useEnsure, useTombstone, createSelector } =
  Flux.createRetrieve<RetrieveQuery, icd.ICD>({
    name: RESOURCE_NAME,
    retrieve: async ({ client, query }) => await client.icds.retrieve(query.key),
    onChange: ({ client, query }, handler) => client.icds.onChange(query.key, handler),
    getCached: ({ client, query }) => client.icds.getCached(query.key),
    awaitCreation: true,
  });

export const useName = createSelector(({ name }) => name);

export type ListQuery = icd.RetrieveMultipleParams;

export const useList = Flux.createList<ListQuery, icd.Key, icd.ICD>({
  name: PLURAL_RESOURCE_NAME,
  retrieve: async ({ client, query }) => await client.icds.retrieve(query),
  retrieveByKey: async ({ client, key }) => await client.icds.retrieve({ key }),
  onChange: ({ client, query }, handler) => client.icds.onChange(query, handler),
  onChangeByKey: ({ client, key }, handler) => client.icds.onChange({ key }, handler),
  getCached: ({ client, query }) => client.icds.getCached(query),
});

export interface CreateParams extends icd.New {}

export const { useUpdate: useCreate } = Flux.createUpdate<CreateParams, icd.ICD>({
  name: RESOURCE_NAME,
  verbs: verbs.CREATE,
  update: async ({ client, data, onOptimisticComplete }) =>
    await client.icds.create(data, {
      onOptimistic: async ([optimistic]) => await onOptimisticComplete(optimistic),
    }),
});

export interface RenameParams extends Pick<icd.ICD, "key" | "name"> {}

export const { useUpdate: useRename } = Flux.createUpdate<RenameParams>({
  name: RESOURCE_NAME,
  verbs: verbs.RENAME,
  update: async ({ client, data, onOptimisticComplete }) => {
    const { key, name } = data;
    await client.icds.rename(key, name, {
      onOptimistic: async () => await onOptimisticComplete(data),
    });
    return data;
  },
});

export type DeleteParams = icd.Key | icd.Key[];

export const { useUpdate: useDelete } = Flux.createUpdate<DeleteParams>({
  name: PLURAL_RESOURCE_NAME,
  verbs: verbs.DELETE,
  update: async ({ client, data, onOptimisticComplete }) => {
    await client.icds.delete(data, {
      onOptimistic: async () => await onOptimisticComplete(data),
    });
    return data;
  },
});

export interface ImportParams {
  key: icd.Key;
  format: icd.ImportFormat;
  data: Uint8Array;
}

export const { useUpdate: useImport } = Flux.createUpdate<ImportParams, icd.ICD>({
  name: RESOURCE_NAME,
  verbs: IMPORT_VERBS,
  update: async ({ client, data: { key, format, data } }) =>
    await client.icds.import(key, data, format),
});

export const formSchema = icd.icdZ.pick({ name: true, messages: true });

export const formPath = deep.path<z.infer<typeof formSchema>>();

/**
 * A form over the name and messages of a cached ICD. Each edit reaches the Core as a
 * path action. Problems do not stop an edit: {@link useCheck} shows them.
 */
export const useForm = Flux.createDocumentForm<icd.Key, icd.ICD, typeof formSchema>({
  name: RESOURCE_NAME,
  schema: formSchema,
  domain: (client) => client.icds,
  useEnsure,
  toValues: ({ name, messages }) => ({ name, messages }),
});
