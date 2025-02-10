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
import { type UnknownRecord } from "@synnaxlabs/x";
import { useState } from "react";

import { Layout } from "@/layout";
import { Range } from "@/range";

export interface ParentRangeButtonProps<
  Config extends UnknownRecord = UnknownRecord,
  Details extends {} = UnknownRecord,
  Type extends string = string,
> {
  task: task.Task<Config, Details, Type>;
}

export const ParentRangeButton = <
  Config extends UnknownRecord = UnknownRecord,
  Details extends {} = UnknownRecord,
  Type extends string = string,
>({
  task,
}: ParentRangeButtonProps<Config, Details, Type>) => {
  const client = Synnax.use();
  const handleException = Status.useExceptionHandler();
  const [parent, setParent] = useState<ontology.Resource>();
  const placeLayout = Layout.usePlacer();
  useAsyncEffect(async () => {
    try {
      if (client == null) return;
      const parent = await task.snapshottedTo();
      if (parent != null) setParent(parent);
      const tracker = await client.ontology.openDependentTracker({
        target: task.ontologyID,
        dependents: parent == null ? [] : [parent],
        relationshipDirection: ontology.TO_RELATIONSHIP_DIRECTION,
      });
      tracker.onChange((parents) => {
        const rng = parents.find((p) => p.id.matchesType(ranger.ONTOLOGY_TYPE));
        setParent(rng);
      });
      return async () => await tracker.close();
    } catch (e) {
      handleException(e, `Failed to retrieve parent ranges for ${task.name}`);
      setParent(undefined);
      return undefined;
    }
  }, [task, client?.key]);
  if (parent == null) return null;
  const handleClick = () =>
    placeLayout({ ...Range.OVERVIEW_LAYOUT, key: parent.id.key, name: parent.name });
  return (
    <Align.Space align="center" direction="x" size="small">
      <Text.Text level="p">Snapshotted to</Text.Text>
      <Button.Button
        iconSpacing="small"
        onClick={handleClick}
        shade={7}
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
