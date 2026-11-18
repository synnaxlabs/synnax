// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { enumeration, type resource } from "@synnaxlabs/client";
import { deep, verbs } from "@synnaxlabs/x";
import type z from "zod";

import { Flux } from "@/flux";
import { flux } from "@/flux/aether";

const RESOURCE_NAME = "enumeration";
const PLURAL_RESOURCE_NAME = "enumerations";

export type RetrieveQuery = enumeration.RetrieveSingleParams;

export const { use: useRetrieve, useResult } = Flux.createRetrieve<
  RetrieveQuery,
  enumeration.Enumeration
>({
  name: RESOURCE_NAME,
  retrieve: async ({ client, query: { key } }) =>
    await client.enumerations.retrieve(key),
  onChange: ({ client, query: { key } }, handler) =>
    client.enumerations.watch(key).onChange(handler),
  getCached: ({ client, query: { key } }) => client.enumerations.watch(key).getCached(),
});

export type RetrieveMultipleQuery = {
  keys: enumeration.Key[];
};

export const { use: useRetrieveMultiple, useResult: useResultMultiple } =
  Flux.createRetrieve<RetrieveMultipleQuery, enumeration.Enumeration[]>({
    name: PLURAL_RESOURCE_NAME,
    retrieve: async ({ client, query: { keys } }) =>
      await client.enumerations.retrieve(keys),
    onChange: ({ client, query: { keys } }, handler) =>
      client.enumerations.watch(keys).onChange(handler),
    getCached: ({ client, query: { keys } }) =>
      client.enumerations.watch(keys).getCached(),
  });

const { useEnsure } = Flux.createRetrieve<RetrieveQuery, enumeration.Enumeration>({
  name: RESOURCE_NAME,
  retrieve: async ({ client, query: { key } }) =>
    await client.enumerations.retrieve(key),
  onChange: ({ client, query: { key } }, handler) =>
    client.enumerations.watch(key).onChange(handler),
  getCached: ({ client, query: { key } }) => client.enumerations.watch(key).getCached(),
});

export type ListQuery = enumeration.RetrieveMultipleParams;

/** Every enumeration that the query matches, kept live. */
export const { useResult: useResultList } = Flux.createRetrieve<
  ListQuery,
  enumeration.Enumeration[]
>({
  name: PLURAL_RESOURCE_NAME,
  ...flux.listReads(({ client, query }) => client.enumerations.watch(query)),
});

/** The enumerations that the Core ships, with their shipped values. */
export const { useResult: useResultBuiltins } = Flux.createRetrieve<
  Record<string, never>,
  enumeration.Enumeration[]
>({
  name: "built-in enumerations",
  retrieve: async ({ client }) => await client.enumerations.retrieveBuiltins(),
});

/** The records that use an enumeration and that the caller can retrieve. */
export const { useResult: useResultDependents } = Flux.createRetrieve<
  RetrieveQuery,
  resource.ID[]
>({
  name: "enumeration users",
  retrieve: async ({ client, query: { key } }) =>
    await client.enumerations.retrieveDependents(key),
});

export const { useUpdate: useCreate } = Flux.createUpdate<
  enumeration.New,
  enumeration.Enumeration
>({
  name: RESOURCE_NAME,
  verbs: verbs.CREATE,
  update: async ({ client, data }) => await client.enumerations.create(data),
});

export const formSchema = enumeration.enumerationZ.pick({ name: true, entries: true });

export const formPath = deep.path<z.infer<typeof formSchema>>();

/** A form over the name and entries of a cached enumeration. */
export const useForm = Flux.createDocumentForm<
  enumeration.Key,
  enumeration.Enumeration,
  typeof formSchema
>({
  name: RESOURCE_NAME,
  schema: formSchema,
  domain: (client) => client.enumerations,
  useEnsure,
  toValues: ({ name, entries }) => ({ name, entries }),
});

/** Writes the shipped name and entries back onto a built-in enumeration. */
export const { useUpdate: useRestore } = Flux.createUpdate<enumeration.Enumeration>({
  name: RESOURCE_NAME,
  verbs: verbs.UPDATE,
  update: async ({ client, data }) => {
    const { key, name, entries } = data;
    await client.enumerations.retrieve(key);
    await client.enumerations.dispatch(key, [
      enumeration.set({ path: ["name"], value: name }),
      enumeration.set({ path: ["entries"], value: entries }),
    ]);
    return data;
  },
});

export interface RenameParams extends Pick<enumeration.Enumeration, "key" | "name"> {}

export const { useUpdate: useRename } = Flux.createUpdate<RenameParams>({
  name: RESOURCE_NAME,
  verbs: verbs.RENAME,
  update: async ({ client, data }) => {
    const { key, name } = data;
    // A dispatch reaches the Core only for a cached enumeration.
    await client.enumerations.retrieve(key);
    await client.enumerations.dispatch(
      key,
      enumeration.set({ path: ["name"], value: name }),
    );
    return data;
  },
});

export interface ReplaceParams {
  /** The enumeration to delete. */
  from: enumeration.Key;
  /** The enumeration that each user of from moves to. */
  to: enumeration.Key;
}

export const { useUpdate: useReplace } = Flux.createUpdate<ReplaceParams>({
  name: RESOURCE_NAME,
  verbs: verbs.REPLACE,
  update: async ({ client, data, onOptimisticComplete }) => {
    await client.enumerations.replace(data.from, data.to, {
      onOptimistic: async () => await onOptimisticComplete(data),
    });
    return data;
  },
});

export type DeleteParams = enumeration.Key | enumeration.Key[];

export const { useUpdate: useDelete } = Flux.createUpdate<DeleteParams>({
  name: RESOURCE_NAME,
  verbs: verbs.DELETE,
  update: async ({ client, data, onOptimisticComplete }) => {
    await client.enumerations.delete(data, {
      onOptimistic: async () => await onOptimisticComplete(data),
    });
    return data;
  },
});
