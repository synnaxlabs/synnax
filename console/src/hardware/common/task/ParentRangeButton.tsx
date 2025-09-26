// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ranger, task } from "@synnaxlabs/client";
import { Button, Flex, type Flux, Icon, Ranger, Text } from "@synnaxlabs/pluto";
import { useCallback, useState } from "react";

import { useKey } from "@/hardware/common/task/Form";
import { Layout } from "@/layout";
import { OVERVIEW_LAYOUT } from "@/range/overview/layout";

export const ParentRangeButton = () => {
  const taskKey = useKey();
  const [parent, setParent] = useState<ranger.Payload | null>(null);
  Ranger.useRetrieveParentEffect({
    query: taskKey != null ? { id: task.ontologyID(taskKey) } : undefined,
    onChange: useCallback(
      (p: Flux.Result<ranger.Range | null>) => setParent(p.data ?? null),
      [],
    ),
  });
  const placeLayout = Layout.usePlacer();
  if (parent == null) return null;
  const { key, name } = parent;
  const handleClick = () => placeLayout({ ...OVERVIEW_LAYOUT, key, name });
  return (
    <Flex.Box x align="center" gap="small">
      <Text.Text>Snapshotted to</Text.Text>
      <Button.Button
        gap="small"
        onClick={handleClick}
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
