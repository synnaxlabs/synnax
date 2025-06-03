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

import { useAsyncEffect } from "@/hooks";
import {
  useRelationshipDeleteSynchronizer,
  useRelationshipSetSynchronizer,
  useResourceSetSynchronizer,
} from "@/ontology/synchronizers";
import { Status } from "@/status";
import { Synnax } from "@/synnax";

export const useDependentTracker = (
  id: ontology.CrudeID,
  direction: ontology.RelationshipDirection,
  filter?: (dependent: ontology.Resource) => boolean,
): ontology.Resource[] => {
  const [dependents, setDependents] = useState<ontology.Resource[]>([]);
  const client = Synnax.use();
  const oppositeDirection = ontology.getOppositeRelationshipDirection(direction);
  const key = new ontology.ID(id).toString();
  const matchRelationshipAndID = useCallback(
    (relationship: ontology.Relationship) =>
      relationship.type === "parent" && relationship[oppositeDirection].equals(key),
    [key, oppositeDirection],
  );
  useAsyncEffect(async () => {
    if (client == null) return;
    const dependents =
      await client.ontology[
        `retrieve${
          direction === ontology.TO_RELATIONSHIP_DIRECTION ? "Children" : "Parents"
        }`
      ](key);
    setDependents(filter ? dependents.filter(filter) : dependents);
  }, [key, direction, client, filter]);
  const handleError = Status.useErrorHandler();
  const handleRelationshipSet = useCallback(
    (relationship: ontology.Relationship) => {
      handleError(async () => {
        if (!matchRelationshipAndID(relationship)) return;
        const dependent = await client?.ontology.retrieve(relationship[direction]);
        if (dependent == null)
          throw new Error(
            `Ontology resource with key ${relationship[direction].toString()} was not found`,
          );
        if (filter != null && !filter(dependent)) return;
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
      }, "Failed to process new ontology relationships");
    },
    [client, handleError, key, direction, matchRelationshipAndID],
  );
  useRelationshipSetSynchronizer(handleRelationshipSet);

  const handleRelationshipDelete = useCallback(
    (relationship: ontology.Relationship) => {
      if (!matchRelationshipAndID(relationship)) return;
      setDependents((prevDependents) =>
        prevDependents.filter((d) => !d.id.equals(relationship[direction])),
      );
    },
    [key, matchRelationshipAndID],
  );
  useRelationshipDeleteSynchronizer(handleRelationshipDelete);

  const handleResourceSet = useCallback(
    (id: ontology.ID) => {
      if (!dependents.some((d) => d.id.equals(id))) return;
      handleError(async () => {
        const nextDependent = await client?.ontology.retrieve(id);
        if (nextDependent == null)
          throw new Error(`Ontology resource with key ${id.toString()} was not found`);
        setDependents((prevDependents) =>
          prevDependents.flatMap((d) => {
            if (!d.id.equals(id)) return [d];
            if (filter == null || filter(nextDependent)) return [nextDependent];
            return [];
          }),
        );
      }, "Failed to process ontology resource");
    },
    [client, handleError, key, dependents, filter],
  );
  useResourceSetSynchronizer(handleResourceSet);

  return dependents;
};
