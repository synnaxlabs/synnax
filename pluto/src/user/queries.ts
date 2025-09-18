// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ontology, user } from "@synnaxlabs/client";
import { array } from "@synnaxlabs/x";
import { z } from "zod";

import { Flux } from "@/flux";
import { Ontology } from "@/ontology";

export type UseDeleteArgs = user.Key | user.Key[];

export interface FluxStore extends Flux.UnaryStore<user.Key, user.User> {}

export const FLUX_STORE_KEY = "user";

export const FLUX_STORE_CONFIG: Flux.UnaryStoreConfig<
  FluxSubStore,
  user.Key,
  user.User
> = {
  listeners: [],
};

export interface FluxSubStore extends Flux.Store {
  [Ontology.RELATIONSHIPS_FLUX_STORE_KEY]: Ontology.RelationshipFluxStore;
  [Ontology.RESOURCES_FLUX_STORE_KEY]: Ontology.ResourceFluxStore;
  [FLUX_STORE_KEY]: FluxStore;
}

export const { useUpdate: useDelete } = Flux.createUpdate<UseDeleteArgs, FluxSubStore>({
  name: "User",
  update: async ({ client, value, store, rollbacks }) => {
    const keys = array.toArray(value);
    const ids = keys.map((k) => user.ontologyID(k));
    const relFilter = Ontology.filterRelationshipsThatHaveIDs(ids);
    rollbacks.add(store.relationships.delete(relFilter));
    rollbacks.add(store.resources.delete(ontology.idToString(ids)));
    await client.users.delete(keys);
    return value;
  },
});

export const retrieveSingle = async ({
  client,
  params,
  store,
}: Flux.RetrieveArgs<{ key: string }, FluxSubStore>) => {
  const { key } = params;
  const cached = store.users.get(key);
  if (cached != null) return cached;
  const user = await client.users.retrieve(params);
  store.users.set(user.key, user);
  return user;
};

export interface UseRenameArgs {
  key: user.Key;
  name: string;
}

export const { useUpdate: useRename } = Flux.createUpdate<UseRenameArgs, FluxSubStore>({
  name: "User",
  update: async ({ client, value, rollbacks, store }) => {
    const { key, name } = value;
    await client.users.changeUsername(key, name);
    const id = user.ontologyID(key);
    rollbacks.add(
      store.resources.set(ontology.idToString(id), (r) =>
        r == null ? undefined : { ...r, name },
      ),
    );
    return value;
  },
});

export interface UseRetrieveGroupArgs {}

export const { useRetrieve: useRetrieveGroupID } = Flux.createRetrieve<
  UseRetrieveGroupArgs,
  ontology.ID | undefined,
  FluxSubStore
>({
  name: "User Group",
  retrieve: async ({ client, store }) => {
    const rels = store.relationships.get((rel) =>
      ontology.matchRelationship(rel, {
        from: ontology.ROOT_ID,
        type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
      }),
    );
    const groups = store.resources.get(rels.map((rel) => ontology.idToString(rel.to)));
    const cachedRes = groups.find((group) => group.name === "Users");
    if (cachedRes != null) return cachedRes.id;
    const res = await client.ontology.retrieveChildren(ontology.ROOT_ID);
    store.resources.set(res);
    return res.find((r) => r.name === "Users")?.id;
  },
});

export const formSchema = user.newZ.extend({
  password: z.string().min(1, "Password is required"),
  firstName: z.string().min(1, "First name is required"),
  lastName: z.string().min(1, "Last name is required"),
});

export interface UseFormParams {
  key?: user.Key;
}

const ZERO_FORM_VALUES: z.infer<typeof formSchema> = {
  key: "",
  username: "",
  firstName: "",
  lastName: "",
  password: "",
};

export const useForm = Flux.createForm<UseFormParams, typeof formSchema, FluxSubStore>({
  name: "User",
  schema: formSchema,
  initialValues: ZERO_FORM_VALUES,
  retrieve: async ({ client, params: { key }, reset, store }) => {
    if (key == null) return;
    const user = await retrieveSingle({
      client,
      params: { key },
      store,
    });
    reset(user);
  },
  update: async ({ client, value }) => {
    await client.users.create(value());
  },
});
