// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { icd, PathError, query, ValidationError } from "@synnaxlabs/client";
import { deep, errors, verbs } from "@synnaxlabs/x";
import { type z } from "zod";

import { Flux } from "@/flux";
import { markProblems, toFormPath } from "@/icd/problems";

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
    retrieve: async ({ client, query }) => await client.icds.retrieve(query),
    onChange: ({ client, query }, handler) => client.icds.onChange(query, handler),
    getCached: ({ client, query }) => client.icds.getCached(query),
    awaitCreation: true,
  });

export const useName = createSelector(({ name }) => name);

export type ListQuery = Pick<
  icd.RetrieveRequest,
  "keys" | "searchTerm" | "offset" | "limit"
>;

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

export const formSchema = icd.icdZ;

export const formPath = deep.path<z.infer<typeof formSchema>>();

const ZERO_FORM_VALUES: z.infer<typeof formSchema> = {
  key: "",
  name: "",
  messages: [],
  hash: "",
};

/**
 * Edits one ICD. A save checks the ICD first, puts each problem on the field it names,
 * and fails with a ValidationError when there are any. Otherwise it writes the whole
 * ICD with the hash the form read, so the Core replaces it and assigns keys to new
 * messages and fields. The save fails with a StaleICDError when the ICD changed since
 * the form read it.
 */
export const useForm = Flux.createForm<RetrieveQuery, typeof formSchema>({
  name: RESOURCE_NAME,
  schema: formSchema,
  initialValues: ZERO_FORM_VALUES,
  retrieve: async ({ client, query: { key } }) => await client.icds.retrieve({ key }),
  getCached: ({ client, query: { key } }) => {
    const cached = client.icds.getCached({ key });
    return query.isLive(cached) ? cached : undefined;
  },
  update: async ({ client, value, reset, setFieldStatus }) => {
    const current = value();
    const problems = await client.icds.check(current);
    if (problems.length > 0) {
      markProblems(problems, current, setFieldStatus);
      const noun = problems.length === 1 ? "problem" : "problems";
      throw new ValidationError(`The ICD has ${problems.length} ${noun}`);
    }
    try {
      reset(await client.icds.create(current));
    } catch (e) {
      if (PathError.matchExact(e)) {
        const path = toFormPath(e.path.join("."), current);
        setFieldStatus(path, {
          key: String(path),
          variant: "error",
          message: e.error.message,
        });
      }
      throw errors.fromUnknown(e);
    }
  },
  // The editor has no name field, so a rename made elsewhere must reach the form before
  // a save writes the old name back.
  mountListeners: ({ client, query: { key }, set }) =>
    client.icds.onChange({ key }, (result) => {
      if (query.isLive(result)) set(formPath.name, result.name);
    }),
});
