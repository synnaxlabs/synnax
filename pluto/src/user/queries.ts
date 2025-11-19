// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ontology, UnexpectedError, user } from "@synnaxlabs/client";
import { array } from "@synnaxlabs/x";
import { z } from "zod";

import { Flux } from "@/flux";
import { type RetrieveParams } from "@/flux/retrieve";
import { Ontology } from "@/ontology";
import { state } from "@/state";

export type UseDeleteArgs = user.Key | user.Key[];

export interface FluxStore extends Flux.UnaryStore<user.Key, user.User> {}

export const FLUX_STORE_KEY = "users";
const RESOURCE_NAME = "User";

const SET_USER_LISTENER: Flux.ChannelListener<FluxSubStore, typeof user.userZ> = {
  channel: user.SET_CHANNEL_NAME,
  schema: user.userZ,
  onChange: ({ store, changed }) => store.users.set(changed.key, changed),
};

const DELETE_USER_LISTENER: Flux.ChannelListener<FluxSubStore, typeof user.keyZ> = {
  channel: user.DELETE_CHANNEL_NAME,
  schema: user.keyZ,
  onChange: ({ store, changed }) => store.users.delete(changed),
};

export const FLUX_STORE_CONFIG: Flux.UnaryStoreConfig<
  FluxSubStore,
  user.Key,
  user.User
> = {
  listeners: [SET_USER_LISTENER, DELETE_USER_LISTENER],
};

export interface FluxSubStore extends Flux.Store {
  [FLUX_STORE_KEY]: FluxStore;
  [Ontology.RELATIONSHIPS_FLUX_STORE_KEY]: Ontology.RelationshipFluxStore;
  [Ontology.RESOURCES_FLUX_STORE_KEY]: Ontology.ResourceFluxStore;
}

export const { useUpdate: useDelete } = Flux.createUpdate<UseDeleteArgs, FluxSubStore>({
  name: RESOURCE_NAME,
  verbs: Flux.DELETE_VERBS,
  update: async ({ client, data, store, rollbacks }) => {
    const keys = array.toArray(data);
    const ids = keys.map((k) => user.ontologyID(k));
    const relFilter = Ontology.filterRelationshipsThatHaveIDs(ids);
    rollbacks.push(store.relationships.delete(relFilter));
    rollbacks.push(store.resources.delete(ontology.idToString(ids)));
    rollbacks.push(store.users.delete(keys));
    await client.users.delete(keys);
    return data;
  },
});

export interface RetrieveQuery {
  key: user.Key;
}

export const retrieveSingle = async ({
  client,
  query,
  store,
}: RetrieveParams<RetrieveQuery, FluxSubStore>) => {
  const { key } = query;
  const cached = store.users.get(key);
  if (cached != null) return cached;
  const user = await client.users.retrieve(query);
  store.users.set(user.key, user);
  return user;
};

export interface ChangeUsernameParams extends Pick<user.User, "key" | "username"> {}

export const { useUpdate: useRename } = Flux.createUpdate<
  ChangeUsernameParams,
  FluxSubStore
>({
  name: RESOURCE_NAME,
  verbs: Flux.RENAME_VERBS,
  update: async ({ client, data, rollbacks, store }) => {
    const { key, username } = data;
    await client.users.changeUsername(key, username);
    const id = user.ontologyID(key);
    rollbacks.push(
      store.resources.set(
        ontology.idToString(id),
        state.skipNull((r) => ({ ...r, username })),
      ),
    );
    rollbacks.push(
      store.users.set(
        key,
        state.skipNull((u) => ({ ...u, username })),
      ),
    );
    return data;
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
  username: "",
  firstName: "",
  lastName: "",
  password: "",
};

export const useForm = Flux.createForm<UseFormParams, typeof formSchema, FluxSubStore>({
  name: "User",
  schema: formSchema,
  initialValues: ZERO_FORM_VALUES,
  retrieve: async ({ client, query: { key }, reset, store }) => {
    if (key == null) return;
    const user = await retrieveSingle({ client, query: { key }, store });
    reset({ ...user, password: "" });
  },
  update: async ({ client, value, store }) => {
    const created = await client.users.create(value());
    store.users.set(created.key, created);
  },
});

export const { useRetrieve } = Flux.createRetrieve<
  Partial<RetrieveQuery>,
  user.User,
  FluxSubStore
>({
  name: RESOURCE_NAME,
  retrieve: async ({ client, query, store }) => {
    const { key } = query;
    if (key == null) {
      const user = client.auth?.user;
      if (user == null) {
        const res = await client.connectivity.check();
        if (res.error != null) throw res.error;
      }
      if (client?.auth?.user == null)
        throw new UnexpectedError(
          "Expected user to be available after successfully connecting to cluster",
        );
      return client.auth.user;
    }
    return await retrieveSingle({ client, query: { key }, store });
  },
});

export interface ListQuery {
  keys?: user.Key[];
}

export const useList = Flux.createList<ListQuery, user.Key, user.User, FluxSubStore>({
  name: "Users",
  retrieveCached: ({ query, store }) =>
    store.users.get((u) => {
      if (query.keys != null && query.keys.length > 0)
        return query.keys.includes(u.key);
      return true;
    }),
  retrieve: async ({ client, query, store }) => {
    const users = await client.users.retrieve({
      keys: query.keys ?? [],
    });
    store.users.set(users);
    return users;
  },
  retrieveByKey: async ({ client, key, store }) =>
    await retrieveSingle({ client, query: { key }, store }),
  mountListeners: ({ store, onChange, onDelete, query: { keys } }) => {
    const keysSet = keys ? new Set(keys) : undefined;
    return [
      store.users.onSet((user) => {
        if (keysSet != null && !keysSet.has(user.key)) return;
        onChange(user.key, user);
      }),
      store.users.onDelete(onDelete),
    ];
  },
});

export const { useRetrieve } = Flux.createRetrieve<
  Partial<RetrieveQuery>,
  user.User,
  FluxSubStore
>({
  name: RESOURCE_NAME,
  retrieve: async ({ client, query, store }) => {
    const { key } = query;
    if (key == null) {
      const user = client.auth?.user;
      if (user == null) {
        const res = await client.connectivity.check();
        if (res.error != null) throw res.error;
      }
      if (client.auth?.user == null)
        throw new UnexpectedError(
          "Expected user to be available after successfully connecting to cluster",
        );
      return client.auth.user;
    }
    return await retrieveSingle({ client, query: { key }, store });
  },
});
