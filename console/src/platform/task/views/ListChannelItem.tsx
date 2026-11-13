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
import { deep } from "@synnaxlabs/x";
import { type PropsWithChildren, type ReactElement, type ReactNode } from "react";

import { CSS } from "@/platform/css";
import { ChannelName } from "@/platform/task/ChannelName";
import { EnabledCheckbox } from "@/platform/task/EnabledCheckbox";
import { getChannelNameID } from "@/platform/task/getChannelNameID";
import { useHoveredRef, useIsHovered } from "@/platform/task/port/Hovered";
import { TareButton } from "@/platform/task/TareButton";
import { type Channel } from "@/platform/task/types";

/** The channel column that follows a task's own columns. */
export const LIST_CHANNEL_COLUMNS: List.Column[] = [
  { name: "Channel", width: "minmax(24rem, 1fr)" },
];

/** The command and state columns that follow a write task's own columns. */
export const LIST_WRITE_CHANNEL_COLUMNS: List.Column[] = [
  { name: "Command", width: "minmax(24rem, 1fr)" },
  { name: "State", width: "minmax(24rem, 1fr)" },
];

interface ItemChannel extends Channel {
  name?: string;
  cmdChannelName?: string;
  stateChannelName?: string;
}

export interface ListChannelItemProps extends List.ItemProps<string> {
  /** The channel's form path. */
  path: deep.Target<ItemChannel>;
  /** The bound channel, or 0 when none exists yet. A write task's command channel. */
  channel: channel.Key;
  /** The name deploying the task gives a new channel. */
  defaultName?: string;
  /** A write task's state channel, or 0 when none exists yet. */
  stateChannel?: channel.Key;
  /** The name deploying the task gives a new state channel. */
  stateDefaultName?: string;
  /** Deploying the task skips the channel. */
  skipped?: boolean;
  /** Shows a tare button for a float read channel. */
  tareable?: boolean;
  /** One {@link List.Cell} for each of the task's own columns. */
  children: ReactNode;
}

const ChannelCell = ({ children }: PropsWithChildren) => (
  <Flex.Box
    x
    align="center"
    justify="between"
    gap="large"
    className={CSS.BE("list-channel-item", "channel")}
  >
    {children}
  </Flex.Box>
);

/**
 * A channel of a task's record table: a checkbox, the task's fields, an arrow, then
 * the channel's name and enabled switch. A write channel shows its command and state
 * names. Only the checkbox and modifier clicks select it, so editing a field never
 * does. Under a list with a map, it shows as hovered while the pointer is on its port.
 */
export const ListChannelItem = ({
  path,
  channel,
  defaultName,
  stateChannel,
  stateDefaultName,
  skipped,
  tareable = false,
  children,
  className,
  ...rest
}: ListChannelItemProps): ReactElement => {
  const item = deep.widen(path);
  const disabled = Form.useFieldValue(item.disabled, { optional: true });
  const linked = useIsHovered(rest.itemKey);
  const hoveredRef = useHoveredRef(rest.itemKey);
  const flow = <Icon.Arrow.Right className={CSS.BE("list-channel-item", "flow")} />;
  const enabled = <EnabledCheckbox path={item.disabled} />;
  let channels: ReactNode;
  if (stateChannel == null)
    channels = (
      <ChannelCell>
        <ChannelName
          channel={channel}
          namePath={item.name}
          defaultName={defaultName}
          skipped={skipped}
          id={getChannelNameID(rest.itemKey)}
          icon={flow}
        />
        <Flex.Box x align="center" gap="tiny">
          {tareable && <TareButton channel={channel} disabled={disabled ?? false} />}
          {enabled}
        </Flex.Box>
      </ChannelCell>
    );
  else {
    const cmd = (
      <ChannelName
        channel={channel}
        namePath={item.cmdChannelName}
        defaultName={defaultName}
        skipped={skipped}
        id={getChannelNameID(rest.itemKey, "cmd")}
        icon={flow}
      />
    );
    const state = (
      <ChannelName
        channel={stateChannel}
        namePath={item.stateChannelName}
        defaultName={stateDefaultName}
        skipped={skipped}
        id={getChannelNameID(rest.itemKey, "state")}
        icon={flow}
      />
    );
    channels = (
      <>
        <ChannelCell>{cmd}</ChannelCell>
        <ChannelCell>
          {state}
          {enabled}
        </ChannelCell>
      </>
    );
  }
  return (
    <Select.Item
      {...rest}
      // Only a hovered port overrides the item's own hover, such as the keyboard's.
      {...(linked && { hovered: true })}
      ref={hoveredRef}
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
        {channels}
      </List.Cells>
    </Select.Item>
  );
};
