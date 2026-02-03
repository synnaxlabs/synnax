// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { group, ontology } from "@synnaxlabs/client";

import { Flux } from "@/flux";
import { Ontology } from "@/ontology";

export const FLUX_STORE_KEY = "groups";
export const RESOURCE_NAME = "group";
export const PLURAL_RESOURCE_NAME = "groups";

export interface FluxStore extends Flux.UnaryStore<group.Key, group.Group> {}

export interface FluxSubStore extends Flux.Store {
  [FLUX_STORE_KEY]: FluxStore;
  [Ontology.RELATIONSHIPS_FLUX_STORE_KEY]: Ontology.RelationshipFluxStore;
  [Ontology.RESOURCES_FLUX_STORE_KEY]: Ontology.ResourceFluxStore;
}

const SET_GROUP_LISTENER: Flux.ChannelListener<FluxSubStore, typeof group.groupZ> = {
  channel: group.SET_CHANNEL_NAME,
  schema: group.groupZ,
  onChange: ({ store, changed }) => store.groups.set(changed.key, changed),
};

const DELETE_GROUP_LISTENER: Flux.ChannelListener<FluxSubStore, typeof group.keyZ> = {
  channel: group.DELETE_CHANNEL_NAME,
  schema: group.keyZ,
  onChange: ({ store, changed }) => store.groups.delete(changed),
};

export const FLUX_STORE_CONFIG: Flux.UnaryStoreConfig<FluxSubStore> = {
  listeners: [SET_GROUP_LISTENER, DELETE_GROUP_LISTENER],
};

export interface RetrieveQuery {
  key: group.Key;
}

export const retrieveSingle = async ({
  query: { key },
  client,
  store,
}: Flux.RetrieveParams<RetrieveQuery, FluxSubStore>) => {
  const cached = store.groups.get(key);
  if (cached != null) return cached;
  const res = await client.ontology.retrieve(group.ontologyID(key));
  store.groups.set(key, group.groupZ.parse(res.data));
  return group.groupZ.parse(res.data);
};

export interface CreateParams extends group.CreateArgs {}

export const { useUpdate: useCreate } = Flux.createUpdate<CreateParams, FluxSubStore>({
  name: RESOURCE_NAME,
  verbs: Flux.CREATE_VERBS,
  update: async ({ data, client, store }) => {
    const { parent } = data;
    const res = await client.ontology.groups.create(data);
    store.groups.set(res.key, res);
    return { ...res, parent };
  },
});

export interface ListQuery {
  parent?: ontology.ID;
  searchTerm?: string;
  offset?: number;
  limit?: number;
}

export const useList = Flux.createList<ListQuery, group.Key, group.Group, FluxSubStore>(
  {
    name: PLURAL_RESOURCE_NAME,
    retrieveCached: ({ store, query: { parent } }) => {
      if (parent == null) return [];
      const rels = store.relationships.get((r) =>
        ontology.matchRelationship(r, { from: parent, type: "parent" }),
      );
      return store.groups.get(rels.map((r) => r.to.key));
    },
    retrieve: async ({ client, query, store }) => {
      const { parent } = query;
      if (parent == null) return [];
      const res = await client.ontology.retrieveChildren(parent, {
        ...query,
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
    retrieveByKey: async ({ key, ...rest }) =>
      await retrieveSingle({ ...rest, query: { key } }),
    mountListeners: ({ store, onChange, onDelete, query: { parent }, client }) => [
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
        const group = await retrieveSingle({
          query: { key: rel.to.key },
          client,
          store,
        });
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
  },
);

export interface DeleteParams {
  key: string;
}

export const { useUpdate: useDelete } = Flux.createUpdate<DeleteParams, FluxSubStore>({
  name: RESOURCE_NAME,
  verbs: Flux.DELETE_VERBS,
  update: async ({ client, data, store }) => {
    await client.ontology.groups.delete(data.key);
    store.groups.delete(data.key);
    return data;
  },
});

export interface RenameParams extends Pick<group.Group, "key" | "name"> {}

export const { useUpdate: useRename } = Flux.createUpdate<RenameParams, FluxSubStore>({
  name: RESOURCE_NAME,
  verbs: Flux.RENAME_VERBS,
  update: async ({ client, data, store, rollbacks }) => {
    const { key, name } = data;
    rollbacks.push(Flux.partialUpdate(store.groups, key, { name }));
    rollbacks.push(Ontology.renameFluxResource(store, group.ontologyID(key), name));
    await client.ontology.groups.rename(key, name);
    return data;
  },
});
