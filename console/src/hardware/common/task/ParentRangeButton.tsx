// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { task } from "@synnaxlabs/client";
import { Button, Flex, Icon, Ontology, Ranger, Text } from "@synnaxlabs/pluto";

import { Layout } from "@/layout";
import { OVERVIEW_LAYOUT } from "@/range/overview/layout";

export interface ParentRangeButtonProps {
  taskKey: task.Key;
}

export const ParentRangeButton = ({ taskKey }: ParentRangeButtonProps) => {
  const { data: parentRangeID } = Ontology.retrieveParentID.useDirect({
    params: { id: task.ontologyID(taskKey) },
  });
  const { data: parentRange } = Ranger.retrieveQuery.useDirect({
    params: { key: parentRangeID?.key ?? "" },
  });
  const placeLayout = Layout.usePlacer();
  if (parentRange == null) return null;
  const { key, name } = parentRange;
  const handleClick = () => placeLayout({ ...OVERVIEW_LAYOUT, key, name });
  return (
    <Flex.Box x align="center" gap="small">
      <Text.Text level="p" shade={11}>
        Snapshotted to
      </Text.Text>
      <Button.Button
        gap="small"
        onClick={handleClick}
        shade={11}
        style={{ padding: "1rem" }}
        variant="text"
        weight={400}
      >
        <Icon.Range />
        {name}
      </Button.Button>
    </Flex.Box>
  );
};
