// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/task/ParentRangeButton.css";

import { ranger, type task } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Text } from "@synnaxlabs/lyra/text";
import { Ranger, Task as PTask } from "@synnaxlabs/pluto";

import { CSS } from "@/platform/css";
import { Errors } from "@/platform/errors";
import { Panel } from "@/platform/panel";
import { useKey } from "@/platform/task/useKey";

interface RangeLinkProps {
  rangeKey: ranger.Key;
}

const RangeLink = ({ rangeKey }: RangeLinkProps) => {
  const { name } = Ranger.use({ key: rangeKey });
  const openTab = Panel.useOpenTab();
  const handleClick = () =>
    openTab({ variant: "resource", resource: ranger.resourceID(rangeKey) });
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

interface InternalProps {
  taskKey: task.Key;
}

const Internal = ({ taskKey }: InternalProps) => {
  const query = { key: taskKey };
  PTask.useEnsure(query);
  const range = PTask.useRange(query);
  if (range == null) return null;
  return <RangeLink rangeKey={range} />;
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
