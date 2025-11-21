// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { access, ontology } from "@synnaxlabs/client";
import { array, uuid } from "@synnaxlabs/x";

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

export interface ListParams extends List.PagerParams {}

export const useList = Flux.createList<
  ListParams,
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
    const ids = keys.map((key) => access.role.ontologyID(key));
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

export const formSchema = access.role.newZ;

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
  },
  retrieve: async ({ client, query, store }) => {
    if (query.key == null) return;
    const role = await retrieveSingle({ client, query: { key: query.key }, store });
    store.roles.set(query.key, role);
  },
  update: async ({ client, value, store, set, rollbacks }) => {
    const v = value();
    let r: access.role.Role = { key: uuid.create(), ...v };
    rollbacks.push(store.roles.set(r.key, r));
    r = await client.access.roles.create(r);
    set("key", r.key);
  },
});
