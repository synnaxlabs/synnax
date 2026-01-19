// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { access, ontology } from "@synnaxlabs/client";
import { array } from "@synnaxlabs/x";
import { z } from "zod";

import { type policy } from "@/access/policy/aether";
import { Flux } from "@/flux";
import { type List } from "@/list";
import { Ontology } from "@/ontology";

const RESOURCE_NAME = "Policy";
const PLURAL_RESOURCE_NAME = "Policies";

export const formSchema = z.object({
  key: z.uuid().optional(),
  name: z.string().min(1, "Name is required"),
  objects: ontology.idZ.array(),
  actions: access.actionZ.array(),
});

export interface RetrieveQuery {
  key: access.policy.Key;
}

const retrieveSingle = async ({
  client,
  query: { key },
  store,
}: Flux.RetrieveParams<RetrieveQuery, policy.FluxSubStore>) => {
  let p = store.policies.get(key);
  if (p != null) return p;
  p = await client.access.policies.retrieve({ key });
  store.policies.set(key, p);
  return p;
};

export const { useRetrieve } = Flux.createRetrieve<
  RetrieveQuery,
  access.policy.Policy,
  policy.FluxSubStore
>({
  name: RESOURCE_NAME,
  retrieve: retrieveSingle,
  mountListeners: ({ store, query: { key }, onChange }) => [
    store.policies.onSet(onChange, key),
  ],
});

export interface ListParams extends List.PagerParams {}

export const useList = Flux.createList<
  ListParams,
  access.policy.Key,
  access.policy.Policy,
  policy.FluxSubStore
>({
  name: PLURAL_RESOURCE_NAME,
  retrieveCached: ({ store }) => store.policies.list(),
  retrieve: async ({ client, query }) => await client.access.policies.retrieve(query),
  retrieveByKey: async ({ key, ...rest }) =>
    await retrieveSingle({ ...rest, query: { key } }),
  mountListeners: ({ store, onChange, onDelete }) => [
    store.policies.onSet((p) => onChange(p.key, p)),
    store.policies.onDelete(onDelete),
  ],
});

export type DeleteParams = access.policy.Key | access.policy.Key[];

export const { useUpdate: useDelete } = Flux.createUpdate<
  DeleteParams,
  policy.FluxSubStore
>({
  name: RESOURCE_NAME,
  verbs: Flux.DELETE_VERBS,
  update: async ({ client, data, store, rollbacks }) => {
    const keys = array.toArray(data);
    const ids = access.policy.ontologyID(keys);
    const relFilter = Ontology.filterRelationshipsThatHaveIDs(ids);
    rollbacks.push(store.relationships.delete(relFilter));
    rollbacks.push(store.resources.delete(keys));
    rollbacks.push(store.policies.delete(keys));
    await client.access.policies.delete(keys);
    return data;
  },
});

export interface RenameParams {
  key: access.policy.Key;
  name: string;
}

export const { useUpdate: useRename } = Flux.createUpdate<
  RenameParams,
  policy.FluxSubStore
>({
  name: RESOURCE_NAME,
  verbs: Flux.RENAME_VERBS,
  update: async ({ client, data, store, rollbacks }) => {
    const { key, name } = data;
    const existing = await retrieveSingle({ client, query: { key }, store });
    rollbacks.push(Flux.partialUpdate(store.policies, key, { name }));
    rollbacks.push(
      Ontology.renameFluxResource(store, access.policy.ontologyID(key), name),
    );
    const updated = await client.access.policies.create({ ...existing, name });
    store.policies.set(key, updated);
    return data;
  },
});

export const useForm = Flux.createForm<
  Partial<RetrieveQuery>,
  typeof formSchema,
  policy.FluxSubStore
>({
  name: RESOURCE_NAME,
  schema: formSchema,
  initialValues: {
    key: undefined,
    name: "",
    objects: [],
    actions: [],
  },
  retrieve: async ({ client, query, store, reset }) => {
    if (query.key == null) return;
    const p = await retrieveSingle({ client, query: { key: query.key }, store });
    reset(p);
  },
  update: async ({ client, value, store, set, rollbacks }) => {
    const v = value();
    const p = await client.access.policies.create(v);
    rollbacks.push(store.policies.set(p.key, p));
    set("key", p.key);
  },
});
