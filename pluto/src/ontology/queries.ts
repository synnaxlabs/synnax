// Copyright 2026 Synnax Labs, Inc.
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
import { type FluxSubStore } from "@/ontology/aether/queries";

const RESOURCE_RESOURCE_NAME = "resource";
const PLURAL_RESOURCE_RESOURCE_NAME = "resources";
const PLURAL_CHILDREN_RESOURCE_NAME = "children";

export const useResourceSetSynchronizer = (
  onSet: (resource: ontology.Resource) => void,
): void => {
  const store = Flux.useStore<FluxSubStore>();
  useEffect(() => store.resources.onSet(onSet), [store.resources]);
};

export const useResourceDeleteSynchronizer = (
  onDelete: (id: ontology.ID) => void,
): void => {
  const store = Flux.useStore<FluxSubStore>();
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
  const store = Flux.useStore<FluxSubStore>();
  useEffect(() => store.relationships.onSet(onSet), [store.relationships]);
};

export const useRelationshipDeleteSynchronizer = (
  onDelete: (relationship: ontology.Relationship) => void,
): void => {
  const store = Flux.useStore<FluxSubStore>();
  useEffect(
    () =>
      store.relationships.onDelete((changed) =>
        onDelete(ontology.relationshipZ.parse(changed)),
      ),
    [store.relationships, onDelete],
  );
};

interface DependentQuery extends List.PagerParams {
  id?: ontology.ID;
}

export const createDependentsListHook = (
  direction: ontology.RelationshipDirection,
  name: string,
) =>
  Flux.createList<DependentQuery, string, ontology.Resource, FluxSubStore>({
    name,
    retrieve: async ({ client, query: { id } }) => {
      if (id == null) return [];
      return await client.ontology.retrieve([id], {
        children: direction === "to",
        parents: direction === "from",
      });
    },
    retrieveByKey: async ({ client, key }) =>
      await client.ontology.retrieve(ontology.idZ.parse(key)),
    mountListeners: ({ store, onChange, onDelete, client, query: { id } }) => [
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

export const useListChildren = createDependentsListHook(
  "to",
  PLURAL_CHILDREN_RESOURCE_NAME,
);

export interface ListQuery extends ontology.RetrieveRequest {}

export const useResourceList = Flux.createList<
  ListQuery,
  string,
  ontology.Resource,
  FluxSubStore
>({
  name: PLURAL_RESOURCE_RESOURCE_NAME,
  retrieveCached: ({ store }) => store.resources.list(),
  retrieve: async ({ client, query, store }) => {
    const res = await client.ontology.retrieve(query);
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

export const retrieveCachedParentID = (
  store: FluxSubStore,
  id: ontology.ID,
): ontology.ID | null => {
  const res = store.relationships.get((r) =>
    ontology.matchRelationship(r, {
      type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
      to: id,
    }),
  );
  if (res.length === 0) return null;
  return res[0].from;
};

export interface RetrieveParentIDQuery {
  id: ontology.ID;
}

export const retrieveParentID = async ({
  client,
  query: { id },
  store,
}: Flux.RetrieveParams<RetrieveParentIDQuery, FluxSubStore>): Promise<ontology.ID> => {
  const cached = retrieveCachedParentID(store, id);
  if (cached != null) return cached;
  const res = await client.ontology.retrieveParents(id);
  store.resources.set(res);
  const rel: ontology.Relationship = {
    from: res[0].id,
    type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
    to: id,
  };
  store.relationships.set(ontology.relationshipToString(rel), rel);
  return res[0].id;
};

export const filterRelationshipsThatHaveIDs =
  (resources: ontology.ID[]) => (rel: ontology.Relationship) =>
    resources.some(
      (resource) =>
        ontology.idsEqual(rel.to, resource) || ontology.idsEqual(rel.from, resource),
    );

export interface MoveChildrenParams {
  source: ontology.ID;
  destination: ontology.ID;
  ids: ontology.ID[];
}

const MOVE_VERBS: Flux.Verbs = {
  present: "move",
  participle: "moving",
  past: "moved",
};

export const { useUpdate: useMoveChildren } = Flux.createUpdate<
  MoveChildrenParams,
  FluxSubStore
>({
  name: PLURAL_CHILDREN_RESOURCE_NAME,
  verbs: MOVE_VERBS,
  update: async ({ client, data, store, rollbacks }) => {
    const { source, destination, ids } = data;
    rollbacks.push(
      store.relationships.delete((rel) =>
        ids.some((id) =>
          ontology.matchRelationship(rel, {
            type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
            from: source,
            to: id,
          }),
        ),
      ),
    );
    ids.forEach((id) => {
      const rel = {
        from: destination,
        type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
        to: id,
      };
      rollbacks.push(store.relationships.set(ontology.relationshipToString(rel), rel));
    });
    await client.ontology.moveChildren(source, destination, ...ids);
    return data;
  },
});

export const renameFluxResource = (
  store: FluxSubStore,
  id: ontology.ID,
  name: string,
) => Flux.partialUpdate(store.resources, ontology.idToString(id), { name });

export interface RetrieveChildrenQuery {
  id: ontology.ID;
}

export const { useRetrieveObservable: useRetrieveObservableChildren } =
  Flux.createRetrieve<RetrieveChildrenQuery, ontology.Resource[], FluxSubStore>({
    name: RESOURCE_RESOURCE_NAME,
    retrieve: async ({ client, query, store }) => {
      const children = await client.ontology.retrieveChildren(query.id);
      store.resources.set(children);
      return children;
    },
  });

interface RetrieveResourceQuery {
  ids: ontology.ID[];
}

export const {
  useRetrieve: useRetrieveResource,
  useRetrieveObservable: useRetrieveObservableResource,
} = Flux.createRetrieve<RetrieveResourceQuery, ontology.Resource[], FluxSubStore>({
  name: RESOURCE_RESOURCE_NAME,
  retrieve: async ({ client, query: { ids }, store }) => {
    const cached = store.resources.get(ontology.idToString(ids));
    if (cached.length === ids.length) return cached;
    const resource = await client.ontology.retrieve(ids);
    store.resources.set(resource);
    return resource;
  },
});
