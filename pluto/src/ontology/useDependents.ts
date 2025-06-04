// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ontology } from "@synnaxlabs/client";
import { useCallback, useState } from "react";

import { NULL_CLIENT_ERROR } from "@/errors";
import { useAsyncEffect } from "@/hooks";
import {
  useRelationshipDeleteSynchronizer,
  useRelationshipSetSynchronizer,
  useResourceSetSynchronizer,
} from "@/ontology/synchronizers";
import { Status } from "@/status";
import { Synnax } from "@/synnax";

const matchRelationshipAndID = (
  relationship: ontology.Relationship,
  direction: ontology.RelationshipDirection,
  key: string,
) =>
  relationship.type === "parent" &&
  relationship[ontology.getOppositeRelationshipDirection(direction)].equals(key);

const useDependentTracker = (
  id: ontology.CrudeID,
  direction: ontology.RelationshipDirection,
): ontology.Resource[] => {
  const [dependents, setDependents] = useState<ontology.Resource[]>([]);
  const client = Synnax.use();
  const key = new ontology.ID(id).toString();
  useAsyncEffect(async () => {
    if (client == null) {
      setDependents([]);
      return;
    }
    const dependents =
      await client.ontology[`retrieve${direction === "to" ? "Children" : "Parents"}`](
        key,
      );
    setDependents(dependents);
  }, [key, direction, client]);
  const handleError = Status.useErrorHandler();

  const handleRelationshipSet = useCallback(
    (relationship: ontology.Relationship) => {
      handleError(async () => {
        if (!matchRelationshipAndID(relationship, direction, key)) return;
        if (client == null) throw NULL_CLIENT_ERROR;
        const dependent = await client.ontology.retrieve(relationship[direction]);
        setDependents((prevDependents) => {
          let changed = false;
          const nextDependents = prevDependents.map((d) => {
            if (d.id.equals(dependent.id)) {
              changed = true;
              return dependent;
            }
            return d;
          });
          if (changed) return nextDependents;
          return [...nextDependents, dependent];
        });
      }, `Failed to add new dependents for ${key}`);
    },
    [client, handleError, key, direction, matchRelationshipAndID],
  );
  useRelationshipSetSynchronizer(handleRelationshipSet);

  const handleRelationshipDelete = useCallback(
    (relationship: ontology.Relationship) => {
      if (!matchRelationshipAndID(relationship, direction, key)) return;
      setDependents((prevDependents) =>
        prevDependents.filter((d) => !d.id.equals(relationship[direction])),
      );
    },
    [key, direction, matchRelationshipAndID],
  );
  useRelationshipDeleteSynchronizer(handleRelationshipDelete);

  const handleResourceSet = useCallback(
    (id: ontology.ID) => {
      if (!dependents.some((d) => d.id.equals(id))) return;
      handleError(async () => {
        if (client == null) throw NULL_CLIENT_ERROR;
        const nextDependent = await client.ontology.retrieve(id);
        setDependents((prevDependents) =>
          prevDependents.flatMap((d) => {
            if (!d.id.equals(id)) return [d];
            return [nextDependent];
          }),
        );
      }, `Failed to update dependents for ${key}`);
    },
    [client, handleError, key, dependents],
  );
  useResourceSetSynchronizer(handleResourceSet);

  return dependents;
};

export const useChildren = (id: ontology.CrudeID): ontology.Resource[] =>
  useDependentTracker(id, "to");

export const useParents = (id: ontology.CrudeID): ontology.Resource[] =>
  useDependentTracker(id, "from");
