// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ontology } from "@synnaxlabs/client";
import { useEffect } from "react";

import { Flux } from "@/flux";
import { type List } from "@/list";

export interface RelationshipFluxStore
  extends Flux.UnaryStore<string, ontology.Relationship> {}

export interface ResourceFluxStore extends Flux.UnaryStore<string, ontology.Resource> {}

export const RELATIONSHIPS_FLUX_STORE_KEY = "relationships";
export const RESOURCES_FLUX_STORE_KEY = "resources";

interface SubStore extends Flux.Store {
  [RELATIONSHIPS_FLUX_STORE_KEY]: RelationshipFluxStore;
  [RESOURCES_FLUX_STORE_KEY]: ResourceFluxStore;
}

const RELATIONSHIP_SET_LISTENER: Flux.ChannelListener<
  SubStore,
  typeof ontology.relationshipZ
> = {
  channel: ontology.RELATIONSHIP_SET_CHANNEL_NAME,
  schema: ontology.relationshipZ,
  onChange: ({ store, changed }) =>
    store.relationships.set(ontology.relationshipToString(changed), changed),
};

const RELATIONSHIP_DELETE_LISTENER: Flux.ChannelListener<
  SubStore,
  typeof ontology.relationshipZ
> = {
  channel: ontology.RELATIONSHIP_DELETE_CHANNEL_NAME,
  schema: ontology.relationshipZ,
  onChange: ({ store, changed }) =>
    store.relationships.delete(ontology.relationshipToString(changed)),
};

export const RELATIONSHIP_FLUX_STORE_CONFIG: Flux.UnaryStoreConfig<SubStore> = {
  listeners: [RELATIONSHIP_SET_LISTENER, RELATIONSHIP_DELETE_LISTENER],
};

const RESOURCE_SET_LISTENER: Flux.ChannelListener<SubStore, typeof ontology.idZ> = {
  channel: ontology.RESOURCE_SET_CHANNEL_NAME,
  schema: ontology.idZ,
  onChange: async ({ store, changed, client }) =>
    store.resources.set(
      ontology.idToString(changed),
      await client.ontology.retrieve(changed),
    ),
};

const RESOURCE_DELETE_LISTENER: Flux.ChannelListener<SubStore, typeof ontology.idZ> = {
  channel: ontology.RESOURCE_DELETE_CHANNEL_NAME,
  schema: ontology.idZ,
  onChange: ({ store, changed }) => store.resources.delete(changed.key),
};

export const RESOURCE_FLUX_STORE_CONFIG: Flux.UnaryStoreConfig<SubStore> = {
  listeners: [RESOURCE_SET_LISTENER, RESOURCE_DELETE_LISTENER],
};

export const useResourceSetSynchronizer = (
  onSet: (resource: ontology.Resource) => void,
): void => {
  const store = Flux.useStore<SubStore>();
  useEffect(() => store.resources.onSet(onSet), [store.resources]);
};

export const useResourceDeleteSynchronizer = (
  onDelete: (id: ontology.ID) => void,
): void => {
  const store = Flux.useStore<SubStore>();
  useEffect(() => {
    const destructor = store.resources.onDelete(async (changed) =>
      onDelete(ontology.idZ.parse(changed)),
    );
    return () => destructor();
  }, [store.resources]);
};

export const useRelationshipSetSynchronizer = (
  onSet: (relationship: ontology.Relationship) => void,
): void => {
  const store = Flux.useStore<SubStore>();
  useEffect(() => store.relationships.onSet(onSet), [store.relationships]);
};

export const useRelationshipDeleteSynchronizer = (
  onDelete: (relationship: ontology.Relationship) => void,
): void => {
  const store = Flux.useStore<SubStore>();
  useEffect(
    () =>
      store.relationships.onDelete((changed) =>
        onDelete(ontology.relationshipZ.parse(changed)),
      ),
    [store.relationships],
  );
};

interface UseDependentQueryParams extends List.PagerParams {
  id?: ontology.ID;
}

interface SubStore extends Flux.Store {
  relationships: RelationshipFluxStore;
  resources: ResourceFluxStore;
}

export const createDependentsListHook = (direction: ontology.RelationshipDirection) =>
  Flux.createList<UseDependentQueryParams, string, ontology.Resource, SubStore>({
    name: "useDependents",
    retrieve: async ({ client, params: { id } }) => {
      if (id == null) return [];
      return await client.ontology.retrieve([id], {
        children: direction === "to",
        parents: direction === "from",
      });
    },
    retrieveByKey: async ({ client, key }) => await client.ontology.retrieve(key),
    mountListeners: ({ store, onChange, onDelete, client, params: { id } }) => [
      store.relationships.onSet(async (relationship) => {
        if (
          ontology.matchRelationship(relationship, {
            type: "parent",
            [ontology.oppositeRelationshipDirection(direction)]: id,
          })
        ) {
          const dependent = await client.ontology.retrieve(relationship[direction]);
          onChange(dependent.key, dependent);
        }
      }),
      store.relationships.onDelete((relationship) => {
        const rel = ontology.relationshipZ.parse(relationship);
        if (
          ontology.matchRelationship(rel, {
            type: "parent",
            [ontology.oppositeRelationshipDirection(direction)]: id,
          })
        )
          onDelete(ontology.idToString(rel[direction]));
      }),
      store.resources.onSet((resource) =>
        onChange(resource.key, (prev) => {
          // Default to null if the resource is not in the list,
          // as we don't want to add any non-children.
          if (prev == null) return null;
          return { ...prev, ...resource };
        }),
      ),
      store.resources.onDelete(async (resource) => onDelete(resource)),
    ],
  });

export const useChildren = createDependentsListHook("to");

export interface ListParams extends ontology.RetrieveRequest {}

export const useResourceList = Flux.createList<
  ListParams,
  string,
  ontology.Resource,
  SubStore
>({
  name: "useResourceList",
  retrieveCached: ({ store }) => store.resources.list(),
  retrieve: async ({ client, params, store }) => {
    const res = await client.ontology.retrieve(params);
    res.forEach((r) => store.resources.set(r.key, r));
    return res;
  },
  retrieveByKey: async ({ client, key, store }) => {
    const res = await client.ontology.retrieve(ontology.idZ.parse(key));
    store.resources.set(key, res);
    return res;
  },
  mountListeners: ({ store, onChange, onDelete }) => [
    store.resources.onSet(async (r) => onChange(r.key, r)),
    store.resources.onDelete(async (key) => onDelete(key)),
  ],
});

export interface RetrieveParentIDParams {
  id: ontology.ID;
  type?: ontology.ResourceType;
}

export const retrieveCachedParentID = (store: SubStore, id: ontology.ID) => {
  const res = store.relationships.get((r) =>
    ontology.matchRelationship(r, {
      type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
      to: id,
    }),
  );
  if (res.length === 0) return null;
  return res[0].from.key;
};

export const retrieveParentID = Flux.createRetrieve<
  RetrieveParentIDParams,
  ontology.ID | null,
  SubStore
>({
  name: "useParentID",
  retrieve: async ({ client, params }) => {
    const res = await client.ontology.retrieveParents(params.id);
    if (params.type == null) return res[0].id;
    const parent = res.find(({ id }) => id.type === params.type);
    if (parent == null) return null;
    return parent.id;
  },
  mountListeners: ({ store, onChange, client, params: { id } }) => [
    store.relationships.onSet(async (relationship) => {
      if (
        ontology.matchRelationship(relationship, {
          type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
          to: id,
        })
      ) {
        const parent = await client.ontology.retrieve(relationship.from);
        onChange(parent.id);
      } else onChange(null);
    }),
    store.relationships.onDelete(async (relationship) => {
      const rel = ontology.relationshipZ.parse(relationship);
      if (
        ontology.matchRelationship(rel, {
          type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
          to: id,
        })
      )
        onChange(null);
    }),
  ],
});
