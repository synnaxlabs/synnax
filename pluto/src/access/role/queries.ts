// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { access, ontology, user } from "@synnaxlabs/client";
import { array, uuid } from "@synnaxlabs/x";
import { z } from "zod";

import { type role } from "@/access/role/aether";
import { Flux } from "@/flux";
import { type List } from "@/list";
import { Ontology } from "@/ontology";

export const RESOURCE_NAME = "Role";
export const PLURAL_RESOURCE_NAME = "Roles";

export interface RetrieveQuery {
  key: string;
}

const retrieveSingle = async ({
  client,
  query: { key },
  store,
}: Flux.RetrieveParams<
  RetrieveQuery,
  role.FluxSubStore
>): Promise<access.role.Role> => {
  let r = store.roles.get(key);
  if (r != null) return r;
  r = await client.access.roles.retrieve({ key });
  store.roles.set(key, r);
  return r;
};

export const { useRetrieve } = Flux.createRetrieve<
  RetrieveQuery,
  access.role.Role,
  role.FluxSubStore
>({
  name: RESOURCE_NAME,
  retrieve: retrieveSingle,
  mountListeners: ({ store, query: { key }, onChange }) => [
    store.roles.onSet(onChange, key),
  ],
});

export interface ListQuery extends List.PagerParams {}

export const useList = Flux.createList<
  ListQuery,
  access.role.Key,
  access.role.Role,
  role.FluxSubStore
>({
  name: PLURAL_RESOURCE_NAME,
  retrieveCached: ({ store }) => store.roles.list(),
  retrieve: async ({ client, query }) => await client.access.roles.retrieve(query),
  retrieveByKey: async ({ key, ...rest }) =>
    await retrieveSingle({ ...rest, query: { key } }),
  mountListeners: ({ store, onChange, onDelete }) => [
    store.roles.onSet((role) => onChange(role.key, role)),
    store.roles.onDelete(onDelete),
  ],
});

export type DeleteParams = access.role.Key | access.role.Key[];

export const { useUpdate: useDelete } = Flux.createUpdate<
  DeleteParams,
  role.FluxSubStore
>({
  name: RESOURCE_NAME,
  verbs: Flux.DELETE_VERBS,
  update: async ({ client, data, store, rollbacks }) => {
    const keys = array.toArray(data);
    const ids = access.role.ontologyID(keys);
    const relFilter = Ontology.filterRelationshipsThatHaveIDs(ids);
    rollbacks.push(store.relationships.delete(relFilter));
    rollbacks.push(store.resources.delete(ontology.idToString(ids)));
    rollbacks.push(store.roles.delete(keys));
    await client.access.roles.delete(keys);
    return data;
  },
});

export interface RenameParams {
  key: access.role.Key;
  name: string;
}

export const { useUpdate: useRename } = Flux.createUpdate<
  RenameParams,
  role.FluxSubStore
>({
  name: RESOURCE_NAME,
  verbs: Flux.RENAME_VERBS,
  update: async ({ client, data, rollbacks, store }) => {
    const { key, name } = data;
    const existing = await retrieveSingle({ client, store, query: { key } });
    const updated = { ...existing, name };
    await client.access.roles.create(updated);
    rollbacks.push(Flux.partialUpdate(store.roles, key, { name }));
    rollbacks.push(
      Ontology.renameFluxResource(store, access.role.ontologyID(key), name),
    );
    return data;
  },
});

export interface ChangeRoleFormQuery {
  key: user.Key;
}

export const changeRoleFormSchema = z.object({
  key: user.keyZ,
  role: access.role.keyZ,
});

const retrieveUserRole = async ({
  client,
  store,
  query: { key: userKey },
}: Flux.RetrieveParams<RetrieveQuery, role.FluxSubStore>): Promise<
  access.role.Key | undefined
> => {
  const userID = user.ontologyID(userKey);
  const rels = store.relationships.get((r) =>
    ontology.matchRelationship(r, {
      type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
      to: userID,
      from: { type: "role" },
    }),
  );
  if (rels.length > 0) return rels[0].from.key;

  const parents = await client.ontology.retrieveParents(userID, { types: ["role"] });
  if (parents.length === 0) return undefined;

  const parent = parents[0];
  const rel: ontology.Relationship = {
    from: parent.id,
    type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
    to: userID,
  };
  store.relationships.set(ontology.relationshipToString(rel), rel);

  const r: access.role.Role = {
    key: parent.id.key,
    name: parent.name,
    ...parent.data,
  };
  store.roles.set(r.key, r);
  return r.key;
};

export const useChangeRoleForm = Flux.createForm<
  ChangeRoleFormQuery,
  typeof changeRoleFormSchema,
  role.FluxSubStore
>({
  name: RESOURCE_NAME,
  schema: changeRoleFormSchema,
  initialValues: { key: "", role: "" },
  retrieve: async ({ client, query: { key: userKey }, reset, store }) => {
    const roleKey = await retrieveUserRole({ client, store, query: { key: userKey } });
    reset({ key: userKey, role: roleKey ?? "" });
  },
  update: async ({ client, value, store, rollbacks }) => {
    const { key: userKey, role: newRoleKey } = value();
    const userID = user.ontologyID(userKey);
    const oldRoleKey = await retrieveUserRole({
      client,
      store,
      query: { key: userKey },
    });
    if (oldRoleKey === newRoleKey) return;
    if (oldRoleKey != null) {
      const oldRel: ontology.Relationship = {
        from: access.role.ontologyID(oldRoleKey),
        type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
        to: userID,
      };
      rollbacks.push(store.relationships.delete(ontology.relationshipToString(oldRel)));
    }
    const newRel: ontology.Relationship = {
      from: access.role.ontologyID(newRoleKey),
      type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
      to: userID,
    };
    rollbacks.push(
      store.relationships.set(ontology.relationshipToString(newRel), newRel),
    );
    if (oldRoleKey != null)
      await client.access.roles.unassign({ user: userKey, role: oldRoleKey });
    await client.access.roles.assign({ user: userKey, role: newRoleKey });
  },
});

export const formSchema = access.role.newZ.extend({
  policies: access.policy.keyZ.array(),
});

export const useForm = Flux.createForm<
  Partial<RetrieveQuery>,
  typeof formSchema,
  role.FluxSubStore
>({
  name: RESOURCE_NAME,
  schema: formSchema,
  initialValues: {
    key: undefined,
    name: "",
    description: "",
    policies: [],
  },
  retrieve: async ({ client, query, store }) => {
    if (query.key == null) return;
    const role = await retrieveSingle({ client, query: { key: query.key }, store });
    store.roles.set(query.key, role);
  },
  update: async ({ client, value, store, set, rollbacks }) => {
    const v = value();
    let r: access.role.Role = { key: uuid.create(), ...v };
    const otgID = access.role.ontologyID(r.key);
    const otgKey = ontology.idToString(otgID);
    rollbacks.push(
      store.resources.set(otgKey, { key: otgKey, id: otgID, name: r.name, data: r }),
    );
    rollbacks.push(store.roles.set(r.key, r));
    if (v.policies.length > 0) {
      await client.ontology.addChildren(
        access.role.ontologyID(r.key),
        ...v.policies.map((p) => access.policy.ontologyID(p)),
      );
      const newRels = v.policies.map(
        (p): ontology.Relationship => ({
          from: access.role.ontologyID(r.key),
          to: access.policy.ontologyID(p),
          type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
        }),
      );
      newRels.forEach((rel) =>
        rollbacks.push(
          store.relationships.set(ontology.relationshipToString(rel), rel),
        ),
      );
    }
    r = await client.access.roles.create(r);
    set("key", r.key);
  },
});
