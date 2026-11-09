// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { access, query, type Synnax, user } from "@synnaxlabs/client";
import { deep, type destructor, verbs } from "@synnaxlabs/x";
import { z } from "zod";

import { Flux } from "@/flux";

const RESOURCE_NAME = "role";
const PLURAL_RESOURCE_NAME = "roles";

export type RetrieveQuery = {
  key: string;
};

const retrieveForUser = async (
  client: Synnax,
  key: user.Key,
): Promise<access.role.Role> => {
  const { role } = await client.users.retrieve(key);
  return await client.access.roles.retrieve(role);
};

const cachedRoleKey = (client: Synnax, key: user.Key): access.role.Key | undefined => {
  const u = client.users.watch(key).getCached();
  return query.isLive(u) ? u.role : undefined;
};

const getCachedForUser = (
  client: Synnax,
  key: user.Key,
): query.Cached<access.role.Role> | undefined => {
  const role = cachedRoleKey(client, key);
  return role == null ? undefined : client.access.roles.watch(role).getCached();
};

export const { use } = Flux.createRetrieve<RetrieveQuery, access.role.Role>({
  name: RESOURCE_NAME,
  retrieve: async ({ client, query }) => await client.access.roles.retrieve(query),
  onChange: ({ client, query }, handler) =>
    client.access.roles.watch(query).onChange(handler),
  getCached: ({ client, query }) => client.access.roles.watch(query).getCached(),
});

export type RetrieveMultipleQuery = {
  keys: access.role.Key[];
};

export const { use: useMultiple } = Flux.createRetrieve<
  RetrieveMultipleQuery,
  access.role.Role[]
>({
  name: PLURAL_RESOURCE_NAME,
  retrieve: async ({ client, query: { keys } }) =>
    await client.access.roles.retrieve(keys),
  onChange: ({ client, query: { keys } }, handler) =>
    client.access.roles.watch(keys).onChange(handler),
  getCached: ({ client, query: { keys } }) =>
    client.access.roles.watch(keys).getCached(),
});

export type ForUserQuery = {
  user: user.Key;
};

/** Retrieves the role a user holds. */
export const { useResult: useResultForUser } = Flux.createRetrieve<
  ForUserQuery,
  access.role.Role
>({
  name: RESOURCE_NAME,
  retrieve: async ({ client, query: { user: key } }) =>
    await retrieveForUser(client, key),
  onChange: ({ client, query: { user: key } }, handler) => {
    // The answer reads the held role too, so an edit to it must notify.
    let detachRole: destructor.Destructor = () => {};
    const watchRole = (): void => {
      detachRole();
      const role = cachedRoleKey(client, key);
      detachRole =
        role == null
          ? () => {}
          : client.access.roles
              .watch(role)
              .onChange(() => handler(getCachedForUser(client, key)));
    };
    watchRole();
    const detachUser = client.users.watch(key).onChange(() => {
      watchRole();
      retrieveForUser(client, key).then(handler, (error: unknown) =>
        console.error(`failed to retrieve the role of user ${key}`, error),
      );
    });
    return () => {
      detachUser();
      detachRole();
    };
  },
  getCached: ({ client, query: { user: key } }) => getCachedForUser(client, key),
});

export type DeleteParams = access.role.Key | access.role.Key[];

export const { useUpdate: useDelete } = Flux.createUpdate<DeleteParams>({
  name: RESOURCE_NAME,
  verbs: verbs.DELETE,
  update: async ({ client, data, onOptimisticComplete }) => {
    await client.access.roles.delete(data, {
      onOptimistic: async () => await onOptimisticComplete(data),
    });
    return data;
  },
});

export interface RenameParams {
  key: access.role.Key;
  name: string;
}

export const { useUpdate: useRename } = Flux.createUpdate<RenameParams>({
  name: RESOURCE_NAME,
  verbs: verbs.RENAME,
  update: async ({ client, data }) => {
    const { key, name } = data;
    await client.access.roles.rename(key, name);
    return data;
  },
});

export type ChangeRoleFormQuery = {
  key: user.Key;
};

export const changeRoleFormSchema = z.object({
  key: user.keyZ,
  role: access.role.keyZ,
});

export const changeRoleFormPath = deep.path<z.infer<typeof changeRoleFormSchema>>();

export const useChangeRoleForm = Flux.createForm<
  ChangeRoleFormQuery,
  typeof changeRoleFormSchema
>({
  name: RESOURCE_NAME,
  schema: changeRoleFormSchema,
  initialValues: { key: "", role: "" },
  retrieve: async ({ client, query: { key } }) => ({
    key,
    role: (await client.users.retrieve(key)).role,
  }),
  update: async ({ client, value }) => {
    const { key, role } = value();
    await client.users.setRole(key, role);
  },
});
