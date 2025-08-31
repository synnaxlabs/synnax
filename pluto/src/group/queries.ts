// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { group, ontology, type Synnax } from "@synnaxlabs/client";

import { Flux } from "@/flux";
import { type Ontology } from "@/ontology";

export const FLUX_STORE_KEY = "groups";

export interface FluxStore extends Flux.UnaryStore<group.Key, group.Payload> {}

interface SubStore extends Flux.Store {
  [FLUX_STORE_KEY]: FluxStore;
  [Ontology.RELATIONSHIPS_FLUX_STORE_KEY]: Ontology.RelationshipFluxStore;
}

const SET_GROUP_LISTENER: Flux.ChannelListener<SubStore, typeof group.groupZ> = {
  channel: group.SET_CHANNEL_NAME,
  schema: group.groupZ,
  onChange: ({ store, changed }) => store.groups.set(changed.key, changed),
};

const DELETE_GROUP_LISTENER: Flux.ChannelListener<SubStore, typeof group.keyZ> = {
  channel: group.DELETE_CHANNEL_NAME,
  schema: group.keyZ,
  onChange: ({ store, changed }) => store.groups.delete(changed),
};

export const FLUX_STORE_CONFIG: Flux.UnaryStoreConfig<SubStore> = {
  listeners: [SET_GROUP_LISTENER, DELETE_GROUP_LISTENER],
};

export const singleRetrieve = async (
  key: group.Key,
  client: Synnax,
  subStore: SubStore,
) => {
  const cached = subStore.groups.get(key);
  if (cached != null) return cached;
  const res = await client.ontology.retrieve(group.ontologyID(key));
  subStore.groups.set(key, group.groupZ.parse(res.data));
  return group.groupZ.parse(res.data);
};

export interface CreateParams {
  key?: group.Key;
}

export interface CreateValue extends group.Payload {
  parent: ontology.ID;
}

export const create = Flux.createUpdate<CreateParams, CreateValue, SubStore>({
  name: "Group",
  update: async ({ value, client, onChange, store }) => {
    const { parent } = value;
    const res = await client.ontology.groups.create(parent, value.name, value.key);
    store.groups.set(res.key, res);
    onChange({ ...res, parent });
  },
});

export interface ListParams {
  parent?: ontology.ID;
  searchTerm?: string;
  offset?: number;
  limit?: number;
}

export const useList = Flux.createList<ListParams, group.Key, group.Payload, SubStore>({
  name: "Group",
  retrieveCached: ({ store, params }) => {
    if (params.parent == null) return [];
    const rels = store.relationships.get((r) =>
      ontology.matchRelationship(r, {
        from: params.parent,
        type: "parent",
      }),
    );
    return store.groups.get(rels.map((r) => r.to.key));
  },
  retrieve: async ({ client, params, store }) => {
    const { parent } = params;
    if (parent == null) return [];
    const res = await client.ontology.retrieveChildren(parent, {
      ...params,
      types: ["group"],
    });
    const groups = res.map((r) => group.groupZ.parse(r.data));
    store.groups.set(groups);
    groups.forEach((g) => {
      const rel = {
        from: parent,
        type: "parent",
        to: group.ontologyID(g.key),
      };
      store.relationships.set(ontology.relationshipToString(rel), rel);
    });
    return groups;
  },
  retrieveByKey: async ({ client, key, store }) =>
    await singleRetrieve(key, client, store),
  mountListeners: ({ store, onChange, onDelete, params: { parent }, client }) => [
    store.groups.onSet((group) =>
      onChange(group.key, (p) => (p == null ? null : group)),
    ),
    store.groups.onDelete(onDelete),
    store.relationships.onSet(async (rel) => {
      if (
        parent == null ||
        !ontology.matchRelationship(rel, { from: parent, type: "parent" })
      )
        return;
      const group = await singleRetrieve(rel.to.key, client, store);
      onChange(group.key, group);
    }),
    store.relationships.onDelete(async (relKey) => {
      const rel = ontology.relationshipZ.parse(relKey);
      if (
        parent == null ||
        !ontology.matchRelationship(rel, { from: parent, type: "parent" })
      )
        return;
      onDelete(rel.to.key);
    }),
  ],
});

export interface RenameParams {
  key: string;
}

export const useRename = Flux.createUpdate<RenameParams, string>({
  name: "Group",
  update: async ({ client, value, params }) =>
    await client.ontology.groups.rename(params.key, value),
}).useDirect;

export interface DeleteParams {
  key: string;
}

export const useDelete = Flux.createUpdate<DeleteParams, void>({
  name: "Group",
  update: async ({ client, params }) => await client.ontology.groups.delete(params.key),
}).useDirect;
