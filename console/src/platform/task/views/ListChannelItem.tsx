// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/task/views/ListChannelItem.css";

import { type channel } from "@synnaxlabs/client";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { List } from "@synnaxlabs/lyra/list";
import { Select } from "@synnaxlabs/lyra/select";
import { type ReactElement, type ReactNode } from "react";

import { CSS } from "@/platform/css";
import { ChannelName } from "@/platform/task/ChannelName";
import { EnabledCheckbox } from "@/platform/task/EnabledCheckbox";
import { getChannelNameID } from "@/platform/task/getChannelNameID";

/** The arrow and channel columns that follow a task's own columns. */
export const LIST_CHANNEL_COLUMNS: List.Column[] = [
  { width: "2.5rem" },
  { name: "Channel", width: "1fr" },
];

export interface ListChannelItemProps extends List.ItemProps<string> {
  /** The channel's form path. */
  path: string;
  /** The bound channel, or 0 when none exists yet. */
  channel: channel.Key;
  /** The name a new channel takes when the task first runs. */
  defaultName?: string;
  /** One {@link List.Cell} for each of the task's own columns. */
  children: ReactNode;
}

/**
 * A channel of a task's record table: a checkbox, the task's fields, an arrow, then
 * the channel's name and enabled switch. Only the checkbox and modifier clicks select
 * it, so editing a field never does.
 */
export const ListChannelItem = ({
  path,
  channel,
  defaultName,
  children,
  className,
  ...rest
}: ListChannelItemProps): ReactElement => {
  const disabled = Form.useFieldValue<boolean>(`${path}.disabled`, { optional: true });
  return (
    <Select.Item
      {...rest}
      checkable
      align="center"
      className={CSS.cls(
        CSS.B("list-channel-item"),
        disabled === true && CSS.BM("list-channel-item", "disabled"),
        className,
      )}
    >
      <List.Cells className={CSS.BE("list-channel-item", "cells")}>
        {children}
        <Icon.Arrow.Right className={CSS.BE("list-channel-item", "flow")} />
        <Flex.Box x align="center" justify="between" gap="large">
          <ChannelName
            channel={channel}
            namePath={`${path}.name`}
            defaultName={defaultName}
            id={getChannelNameID(rest.itemKey)}
          />
          <EnabledCheckbox path={`${path}.disabled`} />
        </Flex.Box>
      </List.Cells>
    </Select.Item>
  );
};
