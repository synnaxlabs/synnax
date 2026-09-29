// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { library, type ontology, PathError, query } from "@synnaxlabs/client";
import { caseconv, deep, errors, verbs } from "@synnaxlabs/x";
import { type z } from "zod";

import { Flux } from "@/flux";

const RESOURCE_NAME = "library";
const PLURAL_RESOURCE_NAME = "libraries";

const IMPORT_VERBS: verbs.Verbs = {
  present: "import",
  participle: "importing",
  past: "imported",
};

export type RetrieveQuery = {
  key: library.Key;
};

export const { use, useEnsure, useTombstone, createSelector } = Flux.createRetrieve<
  RetrieveQuery,
  library.Library
>({
  name: RESOURCE_NAME,
  retrieve: async ({ client, query }) => await client.libraries.retrieve(query),
  onChange: ({ client, query }, handler) => client.libraries.onChange(query, handler),
  getCached: ({ client, query }) => client.libraries.getCached(query),
  awaitCreation: true,
});

export const useName = createSelector(({ name }) => name);

export type ListQuery = Pick<
  library.RetrieveRequest,
  "keys" | "searchTerm" | "offset" | "limit"
>;

export const useList = Flux.createList<ListQuery, library.Key, library.Library>({
  name: PLURAL_RESOURCE_NAME,
  retrieve: async ({ client, query }) => await client.libraries.retrieve(query),
  retrieveByKey: async ({ client, key }) => await client.libraries.retrieve({ key }),
  onChange: ({ client, query }, handler) => client.libraries.onChange(query, handler),
  onChangeByKey: ({ client, key }, handler) =>
    client.libraries.onChange({ key }, handler),
  getCached: ({ client, query }) => client.libraries.getCached(query),
});

export interface CreateParams extends library.New {}

export const { useUpdate: useCreate } = Flux.createUpdate<
  CreateParams,
  library.Library
>({
  name: RESOURCE_NAME,
  verbs: verbs.CREATE,
  update: async ({ client, data, onOptimisticComplete }) =>
    await client.libraries.create(data, {
      onOptimistic: async ([optimistic]) => await onOptimisticComplete(optimistic),
    }),
});

export interface RenameParams extends Pick<library.Library, "key" | "name"> {}

export const { useUpdate: useRename } = Flux.createUpdate<RenameParams>({
  name: RESOURCE_NAME,
  verbs: verbs.RENAME,
  update: async ({ client, data, onOptimisticComplete }) => {
    const { key, name } = data;
    await client.libraries.rename(key, name, {
      onOptimistic: async () => await onOptimisticComplete(data),
    });
    return data;
  },
});

export type DeleteParams = library.Key | library.Key[];

export const { useUpdate: useDelete } = Flux.createUpdate<DeleteParams>({
  name: PLURAL_RESOURCE_NAME,
  verbs: verbs.DELETE,
  update: async ({ client, data, onOptimisticComplete }) => {
    await client.libraries.delete(data, {
      onOptimistic: async () => await onOptimisticComplete(data),
    });
    return data;
  },
});

export interface ImportParams {
  key: library.Key;
  format: library.ImportFormat;
  data: Uint8Array;
}

export const { useUpdate: useImport } = Flux.createUpdate<
  ImportParams,
  library.Library
>({
  name: RESOURCE_NAME,
  verbs: IMPORT_VERBS,
  update: async ({ client, data: { key, format, data } }) =>
    await client.libraries.import(key, format, data),
});

export const formSchema = library.libraryZ;

const ZERO_FORM_VALUES: z.infer<typeof formSchema> = {
  key: "",
  name: "",
  entries: [],
};

/**
 * Rewrites the path of a Core validation error into the form path of the field it
 * names: camel case parts, and entry keys in place of array indexes.
 */
const toFormPath = (path: string[], value: library.Library): string =>
  deep.resolvePath(caseconv.snakeToCamel(path).join("."), value);

/**
 * Edits one library. A save writes the whole library, so the Core replaces it and
 * assigns keys to new entries and fields. A validation error from the Core lands on the
 * field it names.
 */
export const useForm = Flux.createForm<RetrieveQuery, typeof formSchema>({
  name: RESOURCE_NAME,
  schema: formSchema,
  initialValues: ZERO_FORM_VALUES,
  retrieve: async ({ client, query: { key } }) =>
    await client.libraries.retrieve({ key }),
  getCached: ({ client, query: { key } }) => {
    const cached = client.libraries.getCached({ key });
    return query.isLive(cached) ? cached : undefined;
  },
  update: async ({ client, value, reset, setFieldStatus }) => {
    const current = value();
    try {
      reset(await client.libraries.create(current));
    } catch (e) {
      if (PathError.matchExact(e)) {
        const path = toFormPath(e.path, current);
        setFieldStatus(path, { key: path, variant: "error", message: e.error.message });
      }
      throw errors.fromUnknown(e);
    }
  },
  // The editor has no name field, so a rename made elsewhere must reach the form before
  // a save writes the old name back.
  mountListeners: ({ client, query: { key }, set }) =>
    client.libraries.onChange({ key }, (result) => {
      if (query.isLive(result)) set("name", result.name);
    }),
});

const tasksParams = (key: library.Key) => ({
  ids: library.ontologyID(key),
  types: ["task" as const],
});

/** Retrieves the tasks that use a library. */
export const { use: useTasks, useResult: useTasksResult } = Flux.createRetrieve<
  RetrieveQuery,
  ontology.Resource[]
>({
  name: "library tasks",
  retrieve: async ({ client, query: { key } }) =>
    await client.ontology.users.retrieve(tasksParams(key)),
  onChange: ({ client, query: { key } }, handler) =>
    client.ontology.users.onChange(tasksParams(key), handler),
  getCached: ({ client, query: { key } }) =>
    client.ontology.users.getCached(tasksParams(key)),
});
