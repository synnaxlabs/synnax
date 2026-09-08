// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { access, type ontology, query, type Synnax, user } from "@synnaxlabs/client";
import { verbs } from "@synnaxlabs/x";
import { z } from "zod";

import { Flux } from "@/flux";
import { type List } from "@/list";

const RESOURCE_NAME = "role";
const PLURAL_RESOURCE_NAME = "roles";

export type RetrieveQuery = {
  key: string;
};

const parentsQuery = (key: user.Key): ontology.DependentParams => ({
  ids: user.ontologyID(key),
  types: [access.role.TYPE_ONTOLOGY_ID.type],
});

const retrieveForUser = async (
  client: Synnax,
  key: user.Key,
): Promise<access.role.Role[]> => {
  const parents = await client.ontology.parents.retrieve(parentsQuery(key));
  if (parents.length === 0) return [];
  return await client.access.roles.retrieve({ keys: parents.map(({ id }) => id.key) });
};

export const { use } = Flux.createRetrieve<RetrieveQuery, access.role.Role>({
  name: RESOURCE_NAME,
  retrieve: async ({ client, query }) => await client.access.roles.retrieve(query),
  onChange: ({ client, query }, handler) =>
    client.access.roles.onChange(query, handler),
  getCached: ({ client, query }) => client.access.roles.getCached(query),
});

export type ForUserQuery = {
  user: user.Key;
};

/**
 * Retrieves the roles assigned to a user. Assignment is an ontology relationship, so
 * the roles are the user's role-typed parents.
 */
export const { useResult: useResultForUser } = Flux.createRetrieve<
  ForUserQuery,
  access.role.Role[]
>({
  name: PLURAL_RESOURCE_NAME,
  retrieve: async ({ client, query: { user: key } }) =>
    await retrieveForUser(client, key),
  onChange: ({ client, query: { user: key } }, handler) =>
    client.ontology.parents.onChange(parentsQuery(key), () => {
      void retrieveForUser(client, key).then(handler);
    }),
  getCached: ({ client, query: { user: key } }) => {
    const parents = client.ontology.parents.getCached(parentsQuery(key));
    if (!query.isLive(parents)) return undefined;
    if (parents.length === 0) return [];
    return client.access.roles.getCached({ keys: parents.map(({ id }) => id.key) });
  },
});

export type ListQuery = List.PagerParams;

export const useList = Flux.createList<ListQuery, access.role.Key, access.role.Role>({
  name: PLURAL_RESOURCE_NAME,
  retrieve: async ({ client, query }) => await client.access.roles.retrieve(query),
  retrieveByKey: async ({ client, key }) => await client.access.roles.retrieve(key),
  onChange: ({ client, query }, handler) =>
    client.access.roles.onChange(query, handler),
  onChangeByKey: ({ client, key }, handler) =>
    client.access.roles.onChange(key, handler),
  getCached: ({ client, query }) => client.access.roles.getCached(query),
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
  update: async ({ client, data, onOptimisticComplete }) => {
    const { key, name } = data;
    await client.access.roles.rename(key, name, {
      onOptimistic: async () => await onOptimisticComplete(data),
    });
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

const retrieveUserRole = async (
  client: Synnax,
  userKey: user.Key,
): Promise<access.role.Key | undefined> => {
  const parents = await client.ontology.parents.retrieve(parentsQuery(userKey));
  return parents.at(0)?.id.key;
};

export const useChangeRoleForm = Flux.createForm<
  ChangeRoleFormQuery,
  typeof changeRoleFormSchema
>({
  name: RESOURCE_NAME,
  schema: changeRoleFormSchema,
  initialValues: { key: "", role: "" },
  retrieve: async ({ client, query: { key: userKey } }) => ({
    key: userKey,
    role: (await retrieveUserRole(client, userKey)) ?? "",
  }),
  update: async ({ client, value }) => {
    const { key: userKey, role: newRoleKey } = value();
    const oldRoleKey = await retrieveUserRole(client, userKey);
    if (oldRoleKey === newRoleKey) return;
    if (oldRoleKey != null)
      await client.access.roles.unassign({ user: userKey, role: oldRoleKey });
    await client.access.roles.assign({ user: userKey, role: newRoleKey });
  },
});
