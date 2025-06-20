// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type label, ontology } from "@synnaxlabs/client";
import { useCallback, useState } from "react";

import { NULL_CLIENT_ERROR } from "@/errors";
import { useAsyncEffect } from "@/hooks";
import { useDeleteSynchronizer, useSetSynchronizer } from "@/label/synchronizers";
import { Ontology } from "@/ontology";
import { Status } from "@/status";
import { Synnax } from "@/synnax";

export const use = (id: ontology.CrudeID): label.Label[] => {
  const client = Synnax.use();
  const [labels, setLabels] = useState<label.Label[]>([]);
  const idStr = new ontology.ID(id).toString();
  useAsyncEffect(async () => {
    if (client == null) {
      setLabels([]);
      return;
    }
    const labels = await client.labels.retrieveFor(idStr);
    setLabels(labels);
  }, [client, idStr]);

  const handleRelationshipDelete = useCallback(
    (relationship: ontology.Relationship) => {
      if (relationship.type == "labeled_by" && relationship.from.equals(idStr))
        setLabels((prevLabels) =>
          prevLabels.filter((l) => l.key !== relationship.to.key),
        );
    },
    [labels, idStr],
  );
  Ontology.useRelationshipDeleteSynchronizer(handleRelationshipDelete);

  const handleError = Status.useErrorHandler();

  const handleRelationshipSet = useCallback(
    (relationship: ontology.Relationship) => {
      if (relationship.type !== "labeled_by" || !relationship.from.equals(idStr))
        return;
      const { key } = relationship.to;
      if (labels.some((l) => l.key === key)) return;
      handleError(async () => {
        if (client == null) throw NULL_CLIENT_ERROR;
        const label = await client.labels.retrieve(key);
        setLabels((prevLabels) => {
          if (prevLabels.some((l) => l.key === key)) return prevLabels;
          return [...prevLabels, label];
        });
      }, `Failed to process new label for ${idStr}`);
    },
    [client, idStr, labels],
  );
  Ontology.useRelationshipSetSynchronizer(handleRelationshipSet);

  const handleDeleteLabels = useCallback(
    (key: label.Key) => {
      if (labels.some((l) => l.key === key))
        setLabels((prevLabels) => prevLabels.filter((l) => l.key !== key));
    },
    [labels],
  );
  useDeleteSynchronizer(handleDeleteLabels);

  const handleSetLabels = useCallback(
    (label: label.Label) => {
      if (!labels.some((l) => l.key === label.key)) return;
      setLabels((prevLabels) =>
        prevLabels.map((l) => (l.key === label.key ? label : l)),
      );
    },
    [labels],
  );
  useSetSynchronizer(handleSetLabels);

  return labels;
};
