// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NotFoundError, ontology } from "@synnaxlabs/client";

import { Flux } from "@/flux";

export const useResourceSetSynchronizer = (onSet: (id: ontology.ID) => void): void =>
  Flux.useListener({
    channel: ontology.RESOURCE_SET_CHANNEL_NAME,
    onChange: Flux.parsedHandler(ontology.idZ, async ({ changed }) => onSet(changed)),
  });

export const useResourceDeleteSynchronizer = (
  onDelete: (id: ontology.ID) => void,
): void =>
  Flux.useListener({
    channel: ontology.RESOURCE_DELETE_CHANNEL_NAME,
    onChange: Flux.parsedHandler(ontology.idZ, async ({ changed }) => {
      onDelete(changed);
    }),
  });

export const useRelationshipSetSynchronizer = (
  onSet: (relationship: ontology.Relationship) => void,
): void =>
  Flux.useListener({
    channel: ontology.RELATIONSHIP_SET_CHANNEL_NAME,
    onChange: Flux.parsedHandler(ontology.relationshipZ, async ({ changed }) => {
      onSet(changed);
    }),
  });

export const useRelationshipDeleteSynchronizer = (
  onDelete: (relationship: ontology.Relationship) => void,
): void =>
  Flux.useListener({
    channel: ontology.RELATIONSHIP_DELETE_CHANNEL_NAME,
    onChange: Flux.parsedHandler(ontology.relationshipZ, async ({ changed }) => {
      onDelete(changed);
    }),
  });

export const matchRelationshipAndID = (
  relationship: ontology.Relationship,
  direction: ontology.RelationshipDirection,
  id: ontology.ID,
) =>
  relationship.type === ontology.PARENT_OF_RELATIONSHIP_TYPE &&
  ontology.idsEqual(
    relationship[ontology.oppositeRelationshipDirection(direction)],
    id,
  );

interface UseDependentQueryParams {
  id: ontology.ID;
}

export const createDependentsListHook = (direction: ontology.RelationshipDirection) =>
  Flux.createList<UseDependentQueryParams, string, ontology.Resource>({
    name: "useDependents",
    retrieve: async ({ client, params: { id } }) =>
      await client.ontology.retrieve([id], {
        children: direction === "to",
        parents: direction === "from",
      }),
    retrieveByKey: async ({ client, key }) =>
      await client.ontology.retrieve(ontology.idZ.parse(key)),
    listeners: [
      {
        channel: ontology.RELATIONSHIP_SET_CHANNEL_NAME,
        onChange: Flux.parsedHandler(
          ontology.relationshipZ,
          async ({ client, changed, params, onChange }) => {
            if (!("id" in params)) return;
            const { id } = params;
            if (!matchRelationshipAndID(changed, direction, id)) return;
            const dependent = await client.ontology.retrieve(changed[direction]);
            onChange(dependent.key, dependent);
          },
        ),
      },
      {
        channel: ontology.RELATIONSHIP_DELETE_CHANNEL_NAME,
        onChange: Flux.parsedHandler(
          ontology.relationshipZ,
          async ({ changed, params, onDelete }) => {
            if (!("id" in params)) return;
            const { id } = params;
            if (!matchRelationshipAndID(changed, direction, id)) return;
            onDelete(ontology.idToString(changed[direction]));
          },
        ),
      },
      {
        channel: ontology.RESOURCE_SET_CHANNEL_NAME,
        onChange: Flux.parsedHandler(
          ontology.idZ,
          async ({ client, changed, params, onChange }) => {
            if (!("id" in params)) return;
            const { id } = params;
            if (!ontology.idsEqual(id, changed)) return;
            const nextDependent = await client.ontology.retrieve(changed);
            onChange(nextDependent.key, nextDependent);
          },
        ),
      },
    ],
  });

export const useChildren = createDependentsListHook("to");
export const useParents = createDependentsListHook("from");

export interface UseResourceQueryParams {
  id: ontology.ID;
}

export const useResource = Flux.createRetrieve<
  UseResourceQueryParams,
  ontology.Resource
>({
  name: "useResource",
  retrieve: async ({ client, params: { id } }) => client.ontology.retrieve(id),
  listeners: [
    {
      channel: ontology.RESOURCE_SET_CHANNEL_NAME,
      onChange: Flux.parsedHandler(
        ontology.idZ,
        async ({ client, changed, params: { id }, onChange }) => {
          if (!ontology.idsEqual(id, changed)) return;
          const nextDependent = await client.ontology.retrieve(changed);
          onChange(nextDependent);
        },
      ),
    },
    {
      channel: ontology.RESOURCE_DELETE_CHANNEL_NAME,
      onChange: Flux.parsedHandler(ontology.idZ, async ({ params: { id } }) => {
        throw new NotFoundError(
          `Resource with ID ${ontology.idToString(id)} not found`,
        );
      }),
    },
  ],
});

export interface ListParams {
  offset?: number;
  limit?: number;
  term?: string;
}

export const useResourceList = Flux.createList<ListParams, string, ontology.Resource>({
  name: "useResourceList",
  retrieve: async ({ client, params }) => await client.ontology.retrieve(params),
  retrieveByKey: async ({ client, key }) =>
    await client.ontology.retrieve(ontology.idZ.parse(key)),
});

export const retrieveParentID = Flux.createRetrieve<
  { id: ontology.ID; type?: ontology.ResourceType },
  ontology.ID | null
>({
  name: "useParentID",
  retrieve: async ({ client, params: { id, type } }) => {
    const res = await client.ontology.retrieveParents(id);
    if (type == null) return res[0].id;
    const parent = res.find(({ id }) => id.type === type);
    if (parent == null) return null;
    return parent.id;
  },
  listeners: [
    {
      channel: ontology.RELATIONSHIP_SET_CHANNEL_NAME,
      onChange: Flux.parsedHandler(
        ontology.relationshipZ,
        async ({ changed, params: { id, type }, onChange }) => {
          if (
            changed.type === ontology.PARENT_OF_RELATIONSHIP_TYPE &&
            (type == null || changed.from.type === type) &&
            ontology.idsEqual(changed.to, id)
          )
            onChange(changed.to);
        },
      ),
    },
    {
      channel: ontology.RELATIONSHIP_DELETE_CHANNEL_NAME,
      onChange: Flux.parsedHandler(
        ontology.relationshipZ,
        async ({ changed, params: { id, type }, onChange }) => {
          if (
            changed.type === ontology.PARENT_OF_RELATIONSHIP_TYPE &&
            (type == null || changed.from.type === type) &&
            ontology.idsEqual(changed.to, id)
          )
            onChange(null);
        },
      ),
    },
  ],
});
