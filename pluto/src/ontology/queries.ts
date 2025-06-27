// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NotFoundError, ontology } from "@synnaxlabs/client";
import { type primitive } from "@synnaxlabs/x";

import { Query } from "@/query";
import { Sync } from "@/query/sync";

export const useResourceSetSynchronizer = (onSet: (id: ontology.ID) => void): void =>
  Sync.useListener({
    channel: ontology.RESOURCE_SET_CHANNEL_NAME,
    onChange: Sync.stringHandler(async ({ changed }) =>
      onSet(ontology.idZ.parse(changed)),
    ),
  });

export const useResourceDeleteSynchronizer = (
  onDelete: (id: ontology.ID) => void,
): void =>
  Sync.useListener({
    channel: ontology.RESOURCE_DELETE_CHANNEL_NAME,
    onChange: Sync.stringHandler(async ({ changed }) =>
      onDelete(ontology.idZ.parse(changed)),
    ),
  });

export const useRelationshipSetSynchronizer = (
  onSet: (relationship: ontology.Relationship) => void,
): void =>
  Sync.useListener({
    channel: ontology.RELATIONSHIP_SET_CHANNEL_NAME,
    onChange: Sync.stringHandler(async ({ changed }) =>
      onSet(ontology.relationShipZ.parse(changed)),
    ),
  });

export const useRelationshipDeleteSynchronizer = (
  onDelete: (relationship: ontology.Relationship) => void,
): void =>
  Sync.useListener({
    channel: ontology.RELATIONSHIP_DELETE_CHANNEL_NAME,
    onChange: Sync.stringHandler(async ({ changed }) =>
      onDelete(ontology.relationShipZ.parse(changed)),
    ),
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

interface UseDependentQueryParams extends Record<string, primitive.Value> {
  id: ontology.ID;
  direction: ontology.RelationshipDirection;
}

const useDependents = (
  params: UseDependentQueryParams,
): Query.UseReturn<ontology.Resource[]> =>
  Query.use<UseDependentQueryParams, ontology.Resource[]>({
    name: "useDependents",
    params,
    retrieve: async ({ client, params: { id, direction } }) =>
      await client.ontology[`retrieve${direction === "to" ? "Children" : "Parents"}`](
        id,
      ),
    listeners: [
      {
        channel: ontology.RELATIONSHIP_SET_CHANNEL_NAME,
        onChange: Sync.stringHandler(
          async ({ client, changed, params: { id, direction }, onChange }) => {
            const relationship = ontology.relationShipZ.parse(changed);
            if (!matchRelationshipAndID(relationship, direction, id)) return;
            const dependent = await client.ontology.retrieve(relationship[direction]);
            onChange((p) => [
              ...p.filter((d) => !ontology.idsEqual(d.id, dependent.id)),
              dependent,
            ]);
          },
        ),
      },
      {
        channel: ontology.RELATIONSHIP_DELETE_CHANNEL_NAME,
        onChange: Sync.stringHandler(
          async ({ changed, params: { id, direction }, onChange }) => {
            const relationship = ontology.relationShipZ.parse(changed);
            if (!matchRelationshipAndID(relationship, direction, id)) return;
            onChange((p) =>
              p.filter((d) => !ontology.idsEqual(d.id, relationship[direction])),
            );
          },
        ),
      },

      {
        channel: ontology.RESOURCE_SET_CHANNEL_NAME,
        onChange: Sync.stringHandler(async ({ client, changed, onChange }) => {
          const nextID = ontology.idZ.parse(changed);
          const nextDependent = await client.ontology.retrieve(nextID);
          onChange((p) =>
            p.map((d) => (ontology.idsEqual(d.id, nextID) ? nextDependent : d)),
          );
        }),
      },
    ],
  });

export const useChildren = (id: ontology.ID): Query.UseReturn<ontology.Resource[]> =>
  useDependents({ id, direction: "to" });

export const useParents = (id: ontology.ID): Query.UseReturn<ontology.Resource[]> =>
  useDependents({ id, direction: "from" });

export interface UseResourceQueryParams extends Record<string, primitive.Value> {
  id: ontology.ID;
}

export const useResource = (id: ontology.ID): Query.UseReturn<ontology.Resource> =>
  Query.use<UseResourceQueryParams, ontology.Resource>({
    name: "useResource",
    params: { id },
    retrieve: async ({ client, params: { id } }) => client.ontology.retrieve(id),
    listeners: [
      {
        channel: ontology.RESOURCE_SET_CHANNEL_NAME,
        onChange: Sync.stringHandler(
          async ({ client, changed, params: { id }, onChange }) => {
            const nextID = ontology.idZ.parse(changed);
            if (!ontology.idsEqual(id, nextID)) return;
            const nextDependent = await client.ontology.retrieve(nextID);
            onChange(nextDependent);
          },
        ),
      },
      {
        channel: ontology.RESOURCE_DELETE_CHANNEL_NAME,
        onChange: Sync.stringHandler(async ({ params: { id } }) => {
          throw new NotFoundError(
            `Resource with ID ${ontology.idToString(id)} not found`,
          );
        }),
      },
    ],
  });
