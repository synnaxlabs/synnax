// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type icd, task } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Text } from "@synnaxlabs/lyra/text";
import { ICD } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { CSS } from "@/platform/css";
import { Panel } from "@/platform/panel";

export interface TasksProps {
  icdKey: icd.Key;
}

/** Lists the tasks that use the ICD. Selecting one opens it. */
export const Tasks = ({ icdKey }: TasksProps): ReactElement => {
  const { data, variant, status } = ICD.useTasksResult({ key: icdKey });
  const openTab = Panel.useOpenTab();
  let content: ReactElement;
  if (variant === "error")
    content = (
      <Text.Text level="small" status="error">
        {status.message}
      </Text.Text>
    );
  else if (data == null || data.length === 0)
    content = (
      <Text.Text level="small" color={9}>
        {data == null ? "Loading tasks" : "No tasks use this ICD"}
      </Text.Text>
    );
  else
    content = (
      <Flex.Box x wrap gap="small">
        {data.map(({ key, name }) => (
          <Button.Button
            key={key}
            variant="text"
            size="small"
            onClick={() =>
              openTab({ variant: "resource", resource: task.resourceID(key) })
            }
          >
            <Icon.Task />
            {name}
          </Button.Button>
        ))}
      </Flex.Box>
    );
  return (
    <Flex.Box y gap="small" className={CSS.BE("icd-editor", "tasks")}>
      <Text.Text level="small" weight={500}>
        Used by
      </Text.Text>
      {content}
    </Flex.Box>
  );
};
