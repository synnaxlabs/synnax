// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ontology, ranger, type task } from "@synnaxlabs/client";
import {
  Align,
  Button,
  Icon,
  Status,
  Synnax,
  Text,
  useAsyncEffect,
} from "@synnaxlabs/pluto";
import { useState } from "react";

import { Layout } from "@/layout";
import { OVERVIEW_LAYOUT } from "@/range/overview/layout";

export interface ParentRangeButtonProps {
  taskKey: task.Key;
}

export const ParentRangeButton = ({ taskKey }: ParentRangeButtonProps) => {
  const client = Synnax.use();
  const handleError = Status.useErrorHandler();
  const [parent, setParent] = useState<ontology.Resource>();
  const placeLayout = Layout.usePlacer();
  useAsyncEffect(async () => {
    try {
      if (client == null) return;
      const parent = await client.hardware.tasks.retrieveSnapshottedTo(taskKey);
      if (parent == null) return;
      setParent(parent);
      const tracker = await client.ontology.openDependentTracker({
        target: parent.id,
        dependents: parent == null ? [] : [parent],
        relationshipDirection: ontology.TO_RELATIONSHIP_DIRECTION,
      });
      tracker.onChange((parents) => {
        const rng = parents.find((p) => p.id.matchesType(ranger.ONTOLOGY_TYPE));
        setParent(rng);
      });
      return async () => await tracker.close();
    } catch (e) {
      handleError(e, `Failed to retrieve parent ranges for task`);
      setParent(undefined);
      return undefined;
    }
  }, [taskKey, client?.key]);
  if (parent == null) return null;
  const handleClick = () =>
    placeLayout({ ...OVERVIEW_LAYOUT, key: parent.id.key, name: parent.name });
  return (
    <Align.Space x align="center" size="small">
      <Text.Text level="p" shade={11}>
        Snapshotted to
      </Text.Text>
      <Button.Button
        iconSpacing="small"
        onClick={handleClick}
        shade={11}
        startIcon={<Icon.Range />}
        style={{ padding: "1rem" }}
        variant="text"
        weight={400}
      >
        {parent.name}
      </Button.Button>
    </Align.Space>
  );
};
