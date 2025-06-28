// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NotFoundError, ontology } from "@synnaxlabs/client";

import { Query } from "@/query";
import { Sync } from "@/query/sync";

export const useResourceSetSynchronizer = (onSet: (id: ontology.ID) => void): void =>
  Sync.useListener({
    channel: ontology.RESOURCE_SET_CHANNEL_NAME,
    onChange: Sync.parsedHandler(ontology.idZ, async ({ changed }) => onSet(changed)),
  });

export const useResourceDeleteSynchronizer = (
  onDelete: (id: ontology.ID) => void,
): void =>
  Sync.useListener({
    channel: ontology.RESOURCE_DELETE_CHANNEL_NAME,
    onChange: Sync.parsedHandler(ontology.idZ, async ({ changed }) => {
      onDelete(changed);
    }),
  });

export const useRelationshipSetSynchronizer = (
  onSet: (relationship: ontology.Relationship) => void,
): void =>
  Sync.useListener({
    channel: ontology.RELATIONSHIP_SET_CHANNEL_NAME,
    onChange: Sync.parsedHandler(ontology.relationShipZ, async ({ changed }) => {
      onSet(changed);
    }),
  });

export const useRelationshipDeleteSynchronizer = (
  onDelete: (relationship: ontology.Relationship) => void,
): void =>
  Sync.useListener({
    channel: ontology.RELATIONSHIP_DELETE_CHANNEL_NAME,
    onChange: Sync.parsedHandler(ontology.relationShipZ, async ({ changed }) => {
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

interface UseDependentQueryParams extends Query.Params {
  id: ontology.ID;
  direction: ontology.RelationshipDirection;
}

const useDependents = (
  params: UseDependentQueryParams,
): Query.CreateReturn<ontology.Resource[]> =>
  Query.useObservable<UseDependentQueryParams, ontology.Resource[]>({
    name: "useDependents",
    params,
    retrieve: async ({ client, params: { id, direction } }) =>
      await client.ontology[`retrieve${direction === "to" ? "Children" : "Parents"}`](
        id,
      ),
    listeners: [
      {
        channel: ontology.RELATIONSHIP_SET_CHANNEL_NAME,
        onChange: Sync.parsedHandler(
          ontology.relationShipZ,
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
        onChange: Sync.parsedHandler(
          ontology.relationShipZ,
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
        onChange: Sync.parsedHandler(
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

export const useChildren = (id: ontology.ID): Query.CreateReturn<ontology.Resource[]> =>
  useDependents({ id, direction: "to" });

export const useParents = (id: ontology.ID): Query.CreateReturn<ontology.Resource[]> =>
  useDependents({ id, direction: "from" });

export interface UseResourceQueryParams extends Query.Params {
  id: ontology.ID;
}

export const useResource = (id: ontology.ID): Query.CreateReturn<ontology.Resource> =>
  Query.useObservable<UseResourceQueryParams, ontology.Resource>({
    name: "useResource",
    params: { id },
    retrieve: async ({ client, params: { id } }) => client.ontology.retrieve(id),
    listeners: [
      {
        channel: ontology.RESOURCE_SET_CHANNEL_NAME,
        onChange: Sync.parsedHandler(
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
        onChange: Sync.parsedHandler(ontology.idZ, async ({ params: { id } }) => {
          throw new NotFoundError(
            `Resource with ID ${ontology.idToString(id)} not found`,
          );
        }),
      },
    ],
  });
