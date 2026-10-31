// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel } from "@synnaxlabs/client";
import { Text } from "@synnaxlabs/lyra/text";
import { Tooltip } from "@synnaxlabs/lyra/tooltip";
import { type JSX, type ReactElement } from "react";

import { Task } from "@/platform/task";

export interface DetailsTitleProps {
  /** The channel's port and its state. Null for a channel type with no port. */
  port: { label: string; state: Task.PortState; disabled: boolean } | null;
  type: { icon: JSX.Element; name: string };
  channel: channel.Key;
  namePath: string;
  /** The name deploying the task gives a channel with no name. */
  defaultName?: string;
  model?: string;
}

/**
 * Names the selected channel in the details header: its port in the grid's state
 * color, its type, its name, and its device's model.
 */
export const DetailsTitle = ({
  port,
  type,
  channel,
  namePath,
  defaultName,
  model,
}: DetailsTitleProps): ReactElement => (
  <>
    <Task.PortBadge state={port?.state ?? "free"} disabled={port?.disabled}>
      {port?.label ?? "NA"}
    </Task.PortBadge>
    <Tooltip.Dialog>
      {type.name}
      <Text.Text level="p" color={9}>
        {type.icon}
      </Text.Text>
    </Tooltip.Dialog>
    <Task.ChannelName
      channel={channel}
      namePath={namePath}
      defaultName={defaultName}
      level="p"
      weight={500}
      overflow="fade"
    />
    {model != null && (
      <Text.Text level="small" color={9} wrap={false}>
        {model}
      </Text.Text>
    )}
  </>
);
