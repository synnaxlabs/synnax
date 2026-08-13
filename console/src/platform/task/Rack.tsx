// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { task } from "@synnaxlabs/client";
import { Icon, Rack as PRack, Text, Tooltip } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { CSS } from "@/platform/css";
import { Errors } from "@/platform/errors";
import { useKey } from "@/platform/task/useKey";

interface ContentProps {
  taskKey: task.Key;
}

const Content = ({ taskKey }: ContentProps): ReactElement => {
  const query = { key: task.rackKey(taskKey) };
  PRack.useEnsure(query);
  const name = PRack.useName(query);
  return (
    <Tooltip.Dialog>
      <Text.Text level="small" color={10} weight={450}>
        Task is deployed to {name}
      </Text.Text>
      <Text.Text className={CSS.B("rack-name")} level="small" color={9} weight={350}>
        <Icon.Rack />
        {name}
      </Text.Text>
    </Tooltip.Dialog>
  );
};

export const Rack = (): ReactElement | null => {
  const taskKey = useKey();
  if (taskKey == null) return null;
  return (
    <Errors.SuspenseBoundary>
      <Content taskKey={taskKey} />
    </Errors.SuspenseBoundary>
  );
};
