// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/hardware/ni/task/DigitalChannelList.css";

import { Align, Form, List, type RenderProp, Text } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { CSS } from "@/css";
import { Common } from "@/hardware/common";
import { type DigitalChannel } from "@/hardware/ni/task/types";

export interface NameProps<C extends DigitalChannel>
  extends Common.Task.ChannelListItemProps<C> {}

interface ListItemProps<C extends DigitalChannel>
  extends Common.Task.ChannelListItemProps<C> {
  name: RenderProp<NameProps<C>>;
}

const ListItem = <C extends DigitalChannel>({
  name,
  path,
  isSnapshot,
  ...rest
}: ListItemProps<C>) => (
  <List.ItemFrame
    {...rest}
    align="center"
    direction="x"
    justify="spaceBetween"
    style={{ width: "100%" }}
  >
    <Align.Space align="center" direction="x" justify="spaceEvenly">
      <Align.Pack
        align="center"
        className="port-line-input"
        direction="x"
        style={{ maxWidth: "50rem" }}
      >
        <Form.NumericField
          inputProps={{ showDragHandle: false }}
          hideIfNull
          showLabel={false}
          showHelpText={false}
          path={`${path}.port`}
        />
        <Text.Text level="p">/</Text.Text>
        <Form.NumericField
          inputProps={{ showDragHandle: false }}
          hideIfNull
          showLabel={false}
          showHelpText={false}
          path={`${path}.line`}
        />
      </Align.Pack>
      <Text.Text
        level="small"
        className={CSS.BE("port-line-input", "label")}
        shade={7}
        weight={450}
      >
        Port/Line
      </Text.Text>
    </Align.Space>
    <Align.Space direction="x" align="center" justify="spaceEvenly">
      {name({ path, isSnapshot, ...rest })}
      <Common.Task.EnableDisableButton
        path={`${path}.enabled`}
        isSnapshot={isSnapshot}
      />
    </Align.Space>
  </List.ItemFrame>
);

export interface DigitalChannelListProps<C extends DigitalChannel>
  extends Omit<Common.Task.Layouts.ListProps<C>, "listItem">,
    Pick<ListItemProps<C>, "name"> {}

export const DigitalChannelList = <C extends DigitalChannel>({
  name,
  ...rest
}: DigitalChannelListProps<C>) => {
  const listItem = useCallback(
    ({ key, ...p }: Common.Task.ChannelListItemProps<C>) => (
      <ListItem key={key} {...p} name={name} />
    ),
    [name],
  );

  return <Common.Task.Layouts.List<C> {...rest} listItem={listItem} />;
};
