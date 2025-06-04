// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ranger, task } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import { Align, Button, Ontology, Text } from "@synnaxlabs/pluto";

import { Layout } from "@/layout";
import { OVERVIEW_LAYOUT } from "@/range/overview/layout";

export interface ParentRangeButtonProps {
  taskKey: task.Key;
}

export const ParentRangeButton = ({ taskKey }: ParentRangeButtonProps) => {
  const parentRange =
    Ontology.useParents(task.ontologyID(taskKey))?.find(
      ({ id: { type } }) => type === ranger.ONTOLOGY_TYPE,
    ) ?? null;
  const placeLayout = Layout.usePlacer();
  if (parentRange == null) return null;
  const handleClick = () =>
    placeLayout({
      ...OVERVIEW_LAYOUT,
      key: parentRange.id.key,
      name: parentRange.name,
    });
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
        {parentRange.name}
      </Button.Button>
    </Align.Space>
  );
};
