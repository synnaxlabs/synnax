// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type arc, query, type Synnax } from "@synnaxlabs/client";
import { verbs } from "@synnaxlabs/x";

import { Flux } from "@/flux";
import { flux } from "@/flux/aether";

export const RESOURCE_NAME = "directory";
const PLURAL_RESOURCE_NAME = "directories";

export type RetrieveRootQuery = Record<string, never>;

/** Returns the key of the directory that holds every other directory and Arc. */
export const { use: useRootKey } = Flux.createRetrieve<
  RetrieveRootQuery,
  arc.directory.Key
>({
  name: "root directory",
  retrieve: async ({ client }) => (await client.arcs.directories.retrieveRoot()).key,
});

export const { useResult: useResultList } = Flux.createRetrieve<
  arc.directory.RetrieveMultipleParams,
  arc.directory.Directory[]
>({
  name: PLURAL_RESOURCE_NAME,
  ...flux.listReads(({ client, query }) => client.arcs.directories.watch(query)),
});

export type RetrieveAncestorsQuery = { keys: arc.directory.Key[] };

const getCachedAncestry = (
  client: Synnax,
  keys: arc.directory.Key[],
): arc.directory.Directory[] | undefined => {
  const directories = client.arcs.directories.watch(keys).getCached();
  const ancestors = client.arcs.directories.getCachedAncestors(keys);
  if (!query.isLive(directories) || ancestors == null) return undefined;
  return [...directories, ...ancestors];
};

/**
 * Retrieves the directories with the keys and every directory above them, up to the
 * root. A rename or a move of one of them updates the answer.
 */
export const { useResult: useResultAncestors } = Flux.createRetrieve<
  RetrieveAncestorsQuery,
  arc.directory.Directory[]
>({
  name: "directory ancestors",
  retrieve: async ({ client, query: { keys } }) => {
    const [directories, ancestors] = await Promise.all([
      client.arcs.directories.retrieve(keys),
      client.arcs.directories.retrieveAncestors(keys),
    ]);
    return [...directories, ...ancestors];
  },
  onChange: ({ client, query: { keys } }, handler) =>
    client.arcs.directories.onAncestorsChange(keys, () =>
      handler(getCachedAncestry(client, keys)),
    ),
  getCached: ({ client, query: { keys } }) => getCachedAncestry(client, keys),
});

export interface CreateParams extends arc.directory.CreateParams {}

export const { useUpdate: useCreate } = Flux.createUpdate<CreateParams>({
  name: RESOURCE_NAME,
  verbs: verbs.CREATE,
  update: async ({ data, client }) => {
    const res = await client.arcs.directories.create(data);
    return { ...res, parent: data.parent };
  },
});

export interface RenameParams extends Pick<arc.directory.Directory, "key" | "name"> {}

export const { useUpdate: useRename } = Flux.createUpdate<RenameParams>({
  name: RESOURCE_NAME,
  verbs: verbs.RENAME,
  update: async ({ client, data }) => {
    await client.arcs.directories.rename(data.key, data.name);
    return data;
  },
});

export const { useUpdate: useDelete } = Flux.createUpdate<
  arc.directory.Key | arc.directory.Key[]
>({
  name: RESOURCE_NAME,
  verbs: verbs.DELETE,
  update: async ({ client, data }) => {
    await client.arcs.directories.delete(data);
    return data;
  },
});
