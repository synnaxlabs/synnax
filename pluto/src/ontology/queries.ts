// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type cache, ontology } from "@synnaxlabs/client";
import { useEffect } from "react";

import { Flux } from "@/flux";
import { type List } from "@/list";
import { Synnax } from "@/synnax";

const RESOURCE_RESOURCE_NAME = "resource";
const PLURAL_RESOURCE_RESOURCE_NAME = "resources";
const PLURAL_CHILDREN_RESOURCE_NAME = "children";

export const useResourceSetSynchronizer = (
  onSet: (resource: ontology.Resource) => void,
): void => {
  const client = Synnax.use();
  useEffect(() => client?.ontology.onResourceSet(onSet), [client, onSet]);
};

export const useResourceDeleteSynchronizer = (
  onDelete: (id: ontology.ID) => void,
): void => {
  const client = Synnax.use();
  useEffect(() => client?.ontology.onResourceDelete(onDelete), [client, onDelete]);
};

export const useRelationshipSetSynchronizer = (
  onSet: (relationship: ontology.Relationship) => void,
): void => {
  const client = Synnax.use();
  useEffect(() => client?.ontology.onRelationshipSet(onSet), [client, onSet]);
};

export const useRelationshipDeleteSynchronizer = (
  onDelete: (relationship: ontology.Relationship) => void,
): void => {
  const client = Synnax.use();
  useEffect(() => client?.ontology.onRelationshipDelete(onDelete), [client, onDelete]);
};

type DependentQuery = List.PagerParams & {
  id?: ontology.ID;
};

const dependentRequest = (
  direction: ontology.RelationshipDirection,
  id: ontology.ID,
): ontology.RetrieveRequest => ({
  ids: [id],
  children: direction === "to",
  parents: direction === "from",
});

export const createDependentsListHook = (
  direction: ontology.RelationshipDirection,
  name: string,
) =>
  Flux.createList<DependentQuery, string, ontology.Resource>({
    name,
    retrieve: async ({ client, query: { id } }) => {
      if (id == null) return [];
      if (direction === "to") return await client.ontology.retrieveChildren(id);
      return await client.ontology.retrieveParents(id);
    },
    retrieveByKey: async ({ client, key }) =>
      await client.ontology.retrieve(ontology.idZ.parse(key)),
    subscribe: ({ client, query: { id } }, handler) => {
      if (id == null) return () => {};
      return client.ontology.onChange(dependentRequest(direction, id), handler);
    },
    getCached: ({ client, query: { id } }) => {
      if (id == null) return undefined;
      return client.ontology.getCached(dependentRequest(direction, id));
    },
    subscribeByKey: ({ client, key }, handler) =>
      client.ontology.onChange(ontology.idZ.parse(key), handler),
  });

export const useListChildren = createDependentsListHook(
  "to",
  PLURAL_CHILDREN_RESOURCE_NAME,
);

export type ListQuery = ontology.RetrieveRequest;

export const useResourceList = Flux.createList<ListQuery, string, ontology.Resource>({
  name: PLURAL_RESOURCE_RESOURCE_NAME,
  retrieve: async ({ client, query }) => await client.ontology.retrieve(query),
  retrieveByKey: async ({ client, key }) =>
    await client.ontology.retrieve(ontology.idZ.parse(key)),
  subscribe: ({ client, query }, handler) => client.ontology.onChange(query, handler),
  getCached: ({ client, query }) => client.ontology.getCached(query),
  subscribeByKey: ({ client, key }, handler) =>
    client.ontology.onChange(ontology.idZ.parse(key), handler),
});

export interface MoveChildrenParams {
  source: ontology.ID;
  destination: ontology.ID;
  ids: ontology.ID[];
}

const MOVE_VERBS: cache.Verbs = {
  present: "move",
  participle: "moving",
  past: "moved",
};

export const { useUpdate: useMoveChildren } = Flux.createUpdate<MoveChildrenParams>({
  name: PLURAL_CHILDREN_RESOURCE_NAME,
  verbs: MOVE_VERBS,
  update: async ({ client, data, onOptimisticComplete }) => {
    const { source, destination, ids } = data;
    await onOptimisticComplete(data);
    await client.ontology.moveChildren(source, destination, ...ids);
    return data;
  },
});

export type RetrieveChildrenQuery = {
  [K in keyof ontology.RetrieveOptions]: ontology.RetrieveOptions[K];
} & {
  id: ontology.ID;
};

export const {
  useRetrieve: useRetrieveChildren,
  useRetrieveObservable: useRetrieveObservableChildren,
} = Flux.createRetrieve<RetrieveChildrenQuery, ontology.Resource[]>({
  name: RESOURCE_RESOURCE_NAME,
  retrieve: async ({ client, query: { id, ...options } }) =>
    await client.ontology.retrieveChildren(id, options),
  subscribe: ({ client, query: { id, ...options } }, handler) =>
    client.ontology.onChange({ ids: [id], ...options, children: true }, handler),
  getCached: ({ client, query: { id, ...options } }) =>
    client.ontology.getCached({ ids: [id], ...options, children: true }),
});

type RetrieveResourceQuery = {
  ids: ontology.ID[];
};

export const {
  useRetrieve: useRetrieveResource,
  useRetrieveObservable: useRetrieveObservableResource,
} = Flux.createRetrieve<RetrieveResourceQuery, ontology.Resource[]>({
  name: RESOURCE_RESOURCE_NAME,
  retrieve: async ({ client, query: { ids } }) => await client.ontology.retrieve(ids),
  subscribe: ({ client, query: { ids } }, handler) =>
    client.ontology.onChange(ids, handler),
  getCached: ({ client, query: { ids } }) => client.ontology.getCached(ids),
});
