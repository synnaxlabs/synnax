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
    onChange: Sync.stringHandler(async ({ changed }) =>
      onSet(new ontology.ID(changed)),
    ),
  });

export const useResourceDeleteSynchronizer = (
  onDelete: (id: ontology.ID) => void,
): void =>
  Sync.useListener({
    channel: ontology.RESOURCE_DELETE_CHANNEL_NAME,
    onChange: Sync.stringHandler(async ({ changed }) =>
      onDelete(new ontology.ID(changed)),
    ),
  });

export const useRelationshipSetSynchronizer = (
  onSet: (relationship: ontology.Relationship) => void,
): void =>
  Sync.useListener({
    channel: ontology.RELATIONSHIP_SET_CHANNEL_NAME,
    onChange: Sync.stringHandler(async ({ changed }) =>
      onSet(ontology.parseRelationship(changed)),
    ),
  });

export const useRelationshipDeleteSynchronizer = (
  onDelete: (relationship: ontology.Relationship) => void,
): void =>
  Sync.useListener({
    channel: ontology.RELATIONSHIP_DELETE_CHANNEL_NAME,
    onChange: Sync.stringHandler(async ({ changed }) =>
      onDelete(ontology.parseRelationship(changed)),
    ),
  });

const matchRelationshipAndID = (
  relationship: ontology.Relationship,
  direction: ontology.RelationshipDirection,
  key: string,
) =>
  relationship.type === ontology.PARENT_OF_RELATIONSHIP_TYPE &&
  relationship[ontology.oppositeRelationshipDirection(direction)].equals(key);

interface UseDependentsArgs {
  id: ontology.CrudeID;
  direction: ontology.RelationshipDirection;
}

const useDependents = (args: UseDependentsArgs): Query.UseReturn<ontology.Resource[]> =>
  Query.use({
    name: "useDependents",
    params: args,
    retrieve: async ({ client, params: { id, direction } }) =>
      await client.ontology[`retrieve${direction === "to" ? "Children" : "Parents"}`](
        new ontology.ID(id).toString(),
      ),
    listeners: [
      {
        channel: ontology.RELATIONSHIP_SET_CHANNEL_NAME,
        onChange: Sync.stringHandler(
          async ({ client, changed, params: { id, direction }, onChange }) => {
            const key = new ontology.ID(id).toString();
            const relationship = ontology.parseRelationship(changed);
            if (!matchRelationshipAndID(relationship, direction, key)) return;
            const dependent = await client.ontology.retrieve(relationship[direction]);
            onChange((p) => [
              ...p.filter((d) => !d.id.equals(dependent.id)),
              dependent,
            ]);
          },
        ),
      },
      {
        channel: ontology.RELATIONSHIP_DELETE_CHANNEL_NAME,
        onChange: Sync.stringHandler(
          async ({ changed, params: { id, direction }, onChange }) => {
            const key = new ontology.ID(id).toString();
            const relationship = ontology.parseRelationship(changed);
            if (!matchRelationshipAndID(relationship, direction, key)) return;
            onChange((p) => p.filter((d) => !d.id.equals(relationship[direction])));
          },
        ),
      },

      {
        channel: ontology.RESOURCE_SET_CHANNEL_NAME,
        onChange: Sync.stringHandler(async ({ client, changed, onChange }) => {
          const dependentID = new ontology.ID(changed);
          const nextDependent = await client.ontology.retrieve(dependentID);
          onChange((p) => p.map((d) => (d.id.equals(dependentID) ? nextDependent : d)));
        }),
      },
    ],
  });

export const useChildren = (
  id: ontology.CrudeID,
): Query.UseReturn<ontology.Resource[]> => useDependents({ id, direction: "to" });

export const useParents = (
  id: ontology.CrudeID,
): Query.UseReturn<ontology.Resource[]> => useDependents({ id, direction: "from" });

export const useResource = (id: ontology.ID): Query.UseReturn<ontology.Resource> =>
  Query.use({
    name: "useResource",
    params: id,
    retrieve: async ({ client, params: id }) => client.ontology.retrieve(id),
    listeners: [
      {
        channel: ontology.RESOURCE_SET_CHANNEL_NAME,
        onChange: Sync.stringHandler(
          async ({ client, changed, params: id, onChange }) => {
            const dependentID = new ontology.ID(changed);
            if (!id.equals(dependentID)) return;
            const nextDependent = await client.ontology.retrieve(dependentID);
            onChange(nextDependent);
          },
        ),
      },
      {
        channel: ontology.RESOURCE_DELETE_CHANNEL_NAME,
        onChange: Sync.stringHandler(async ({ params: id }) => {
          throw new NotFoundError(`Resource with ID ${id.toString()} not found`);
        }),
      },
    ],
  });
