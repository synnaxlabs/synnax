// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { access, type Synnax, user } from "@synnaxlabs/client";
import { z } from "zod";

import { Flux } from "@/flux";
import { type List } from "@/list";

export const RESOURCE_NAME = "Role";
export const PLURAL_RESOURCE_NAME = "Roles";

export type RetrieveQuery = {
  key: string;
};

export const { useRetrieve } = Flux.createRetrieve<RetrieveQuery, access.role.Role>({
  name: RESOURCE_NAME,
  retrieve: async ({ client, query: { key } }) =>
    await client.access.roles.retrieve(key),
  subscribe: ({ client, query: { key } }, handler) =>
    client.access.roles.onChange(key, handler),
  getCached: ({ client, query: { key } }) => client.access.roles.getCached(key),
});

export type ListQuery = List.PagerParams;

export const useList = Flux.createList<ListQuery, access.role.Key, access.role.Role>({
  name: PLURAL_RESOURCE_NAME,
  retrieve: async ({ client, query }) => await client.access.roles.retrieve(query),
  retrieveByKey: async ({ client, key }) => await client.access.roles.retrieve(key),
  subscribe: ({ client, query }, handler) =>
    client.access.roles.onChange(query, handler),
  subscribeByKey: ({ client, key }, handler) =>
    client.access.roles.onChange(key, handler),
  getCached: ({ client, query }) => client.access.roles.getCached(query),
});

export type DeleteParams = access.role.Key | access.role.Key[];

export const { useUpdate: useDelete } = Flux.createUpdate<DeleteParams>({
  name: RESOURCE_NAME,
  verbs: Flux.DELETE_VERBS,
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
  verbs: Flux.RENAME_VERBS,
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

const retrieveUserRole = async (
  client: Synnax,
  userKey: user.Key,
): Promise<access.role.Key | undefined> => {
  const parents = await client.ontology.parents.retrieve({
    ids: user.ontologyID(userKey),
    types: ["role"],
  });
  return parents.at(0)?.id.key;
};

export const useChangeRoleForm = Flux.createForm<
  ChangeRoleFormQuery,
  typeof changeRoleFormSchema
>({
  name: RESOURCE_NAME,
  schema: changeRoleFormSchema,
  initialValues: { key: "", role: "" },
  retrieve: async ({ client, query: { key: userKey }, reset }) => {
    const roleKey = await retrieveUserRole(client, userKey);
    reset({ key: userKey, role: roleKey ?? "" });
  },
  update: async ({ client, value }) => {
    const { key: userKey, role: newRoleKey } = value();
    const oldRoleKey = await retrieveUserRole(client, userKey);
    if (oldRoleKey === newRoleKey) return;
    if (oldRoleKey != null)
      await client.access.roles.unassign({ user: userKey, role: oldRoleKey });
    await client.access.roles.assign({ user: userKey, role: newRoleKey });
  },
});

export const formSchema = access.role.roleZ
  .extend({
    policies: access.policy.keyZ.array(),
  })
  .partial({ key: true });

export const useForm = Flux.createForm<Partial<RetrieveQuery>, typeof formSchema>({
  name: RESOURCE_NAME,
  schema: formSchema,
  initialValues: {
    name: "",
    description: "",
    internal: false,
    policies: [],
  },
  retrieve: async ({ client, query }) => {
    if (query.key == null) return;
    await client.access.roles.retrieve(query.key);
  },
  update: async ({ client, value, set }) => {
    const v = value();
    let r: access.role.Role = access.role.roleZ.parse(v);
    if (v.policies.length > 0)
      await client.ontology.addChildren(
        access.role.ontologyID(r.key),
        ...v.policies.map((p) => access.policy.ontologyID(p)),
      );
    r = await client.access.roles.create(r);
    set("key", r.key);
  },
});
