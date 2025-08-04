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

const matchRelationshipAndID = (
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
  direction: ontology.RelationshipDirection;
}

const retrieveDependents = Flux.createRetrieve<
  UseDependentQueryParams,
  ontology.Resource[]
>({
  name: "useDependents",
  retrieve: async ({ client, params: { id, direction } }) =>
    await client.ontology.retrieve([id], {
      children: direction === "to",
      parents: direction === "from",
    }),
  listeners: [
    {
      channel: ontology.RELATIONSHIP_SET_CHANNEL_NAME,
      onChange: Flux.parsedHandler(
        ontology.relationshipZ,
        async ({ client, changed, params: { id, direction }, onChange }) => {
          if (!matchRelationshipAndID(changed, direction, id)) return;
          const dependent = await client.ontology.retrieve(changed[direction]);
          onChange((p) => [
            ...p.filter((d) => !ontology.idsEqual(d.id, dependent.id)),
            dependent,
          ]);
        },
      ),
    },
    {
      channel: ontology.RELATIONSHIP_DELETE_CHANNEL_NAME,
      onChange: Flux.parsedHandler(
        ontology.relationshipZ,
        async ({ changed, params: { id, direction }, onChange }) => {
          if (!matchRelationshipAndID(changed, direction, id)) return;
          onChange((p) =>
            p.filter((d) => !ontology.idsEqual(d.id, changed[direction])),
          );
        },
      ),
    },
    {
      channel: ontology.RESOURCE_SET_CHANNEL_NAME,
      onChange: Flux.parsedHandler(
        ontology.idZ,
        async ({ client, changed, onChange }) => {
          const nextDependent = await client.ontology.retrieve(changed);
          onChange((p) =>
            p.map((d) => (ontology.idsEqual(d.id, changed) ? nextDependent : d)),
          );
        },
      ),
    },
  ],
});

export const useChildren = (
  id: ontology.ID,
): Flux.UseDirectRetrieveReturn<ontology.Resource[]> =>
  retrieveDependents.useDirect({ params: { id, direction: "to" } });

export const useParents = (
  id: ontology.ID,
): Flux.UseDirectRetrieveReturn<ontology.Resource[]> =>
  retrieveDependents.useDirect({ params: { id, direction: "from" } });

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
