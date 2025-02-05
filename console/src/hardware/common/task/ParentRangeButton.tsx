// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ontology, ranger, type task } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import { Align, Button, Status, Synnax, Text, useAsyncEffect } from "@synnaxlabs/pluto";
import { type ReactElement, useState } from "react";

import { Layout } from "@/layout";
import { Range } from "@/range";

export interface ParentRangeButtonProps {
  key?: task.Key;
}

export const ParentRangeButton = ({
  key,
}: ParentRangeButtonProps): ReactElement | null => {
  const client = Synnax.use();
  const handleException = Status.useExceptionHandler();
  const [parent, setParent] = useState<ontology.Resource>();
  const placeLayout = Layout.usePlacer();
  useAsyncEffect(async () => {
    try {
      if (client == null || key == null) return;
      const tsk = await client.hardware.tasks.retrieve(key);
      const parent = await tsk.snapshottedTo();
      if (parent != null) setParent(parent);
      const tracker = await client.ontology.openDependentTracker({
        target: tsk.ontologyID,
        dependents: parent == null ? [] : [parent],
        relationshipDirection: ontology.TO_RELATIONSHIP_DIRECTION,
      });
      tracker.onChange((parents) => {
        const rng = parents.find((p) => p.id.matchesType(ranger.ONTOLOGY_TYPE));
        setParent(rng);
      });
      return async () => await tracker.close();
    } catch (e) {
      handleException(e, "Failed to retrieve parent ranges");
      setParent(undefined);
      return undefined;
    }
  }, [key, client?.key]);
  if (parent == null) return null;
  return (
    <Align.Space direction="x" size="small" align="center">
      <Text.Text level="p">Snapshotted to</Text.Text>
      <Button.Button
        variant="text"
        shade={7}
        weight={400}
        startIcon={<Icon.Range />}
        iconSpacing="small"
        style={{ padding: "1rem" }}
        onClick={() =>
          placeLayout({
            ...Range.overviewLayout,
            key: parent.id.key,
            name: parent.name,
          })
        }
      >
        {parent.name}
      </Button.Button>
    </Align.Space>
  );
};
