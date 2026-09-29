// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type library } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Text } from "@synnaxlabs/lyra/text";
import { Library } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { CSS } from "@/platform/css";
import { Panel } from "@/platform/panel";

export interface TasksProps {
  libraryKey: library.Key;
}

/** Lists the tasks that use the library. Selecting one opens it. */
export const Tasks = ({ libraryKey }: TasksProps): ReactElement => {
  const { data, variant, status } = Library.useTasksResult({ key: libraryKey });
  const open = Panel.useOpenResource();
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
        {data == null ? "Loading tasks" : "No tasks use this library"}
      </Text.Text>
    );
  else
    content = (
      <Flex.Box x wrap gap="small">
        {data.map((resource) => (
          <Button.Button
            key={resource.key}
            variant="text"
            size="small"
            onClick={() => open(resource)}
          >
            <Icon.Task />
            {resource.name}
          </Button.Button>
        ))}
      </Flex.Box>
    );
  return (
    <Flex.Box y gap="small" className={CSS.BE("library-editor", "tasks")}>
      <Text.Text level="small" weight={500}>
        Used by
      </Text.Text>
      {content}
    </Flex.Box>
  );
};
