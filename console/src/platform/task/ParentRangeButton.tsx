// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/task/ParentRangeButton.css";

import { ranger, task } from "@synnaxlabs/client";
import { Button, Flex, Icon, Ranger, Text } from "@synnaxlabs/pluto";

import { CSS } from "@/platform/css";
import { Errors } from "@/platform/errors";
import { Panel } from "@/platform/panel";
import { useKey } from "@/platform/task/useKey";

interface InternalProps {
  taskKey: task.Key;
}

const Internal = ({ taskKey }: InternalProps) => {
  const parent = Ranger.useParent({ id: task.ontologyID(taskKey) });
  const openTab = Panel.useOpenTab();
  if (parent == null) return null;
  const { key, name } = parent;
  const handleClick = () =>
    openTab({ variant: "resource", resource: ranger.ontologyID(key) });
  return (
    <Flex.Box x align="center" gap="small">
      <Text.Text>Snapshotted to</Text.Text>
      <Button.Button
        gap="small"
        onClick={handleClick}
        className={CSS.B("task-parent-range-button")}
        variant="text"
      >
        <Icon.Range />
        {name}
      </Button.Button>
    </Flex.Box>
  );
};

export const ParentRangeButton = () => {
  const taskKey = useKey();
  if (taskKey == null) return null;
  return (
    <Errors.SuspenseBoundary loading={null}>
      <Internal taskKey={taskKey} />
    </Errors.SuspenseBoundary>
  );
};
