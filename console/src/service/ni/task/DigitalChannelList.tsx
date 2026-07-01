// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/service/ni/task/DigitalChannelList.css";

import { type Component, Flex, Form, Select, Text } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { CSS } from "@/component/css";
import { Task } from "@/component/task";
import { type DigitalChannel } from "@/service/ni/task/types";

interface ListItemProps<C extends DigitalChannel> extends Omit<
  Task.ChannelListItemProps,
  "name"
> {
  name: Component.RenderProp<DigitalNameComponentProps<C>>;
}

export type DigitalNameComponentProps<C extends DigitalChannel> = Omit<C, "key"> & {
  itemKey: string;
  path: string;
};

const ListItem = <C extends DigitalChannel>({ name, ...rest }: ListItemProps<C>) => {
  const path = `config.channels.${rest.itemKey}`;
  const channel = Form.useFieldValue<C>(path);
  if (channel == null) return null;
  return (
    <Select.ListItem {...rest} align="center" justify="between" full="x">
      <Flex.Box align="center" x justify="evenly">
        <Flex.Box
          pack
          align="center"
          className="port-line-input"
          x
          style={{ maxWidth: "50rem" }}
        >
          <Form.NumericField
            inputProps={{ showDragHandle: false }}
            hideIfNull
            showLabel={false}
            showHelpText={false}
            path={`${path}.port`}
          />
          <Text.Text color={9} weight={550}>
            /
          </Text.Text>
          <Form.NumericField
            inputProps={{ showDragHandle: false }}
            hideIfNull
            showLabel={false}
            showHelpText={false}
            path={`${path}.line`}
          />
        </Flex.Box>
        <Text.Text
          level="small"
          className={CSS.BE("port-line-input", "label")}
          weight={450}
        >
          Port/Line
        </Text.Text>
      </Flex.Box>
      <Flex.Box x align="center" justify="evenly">
        {name({ ...channel, itemKey: rest.itemKey, path })}
        <Task.EnableDisableButton path={`${path}.enabled`} />
      </Flex.Box>
    </Select.ListItem>
  );
};

export interface DigitalChannelListProps<C extends DigitalChannel>
  extends
    Omit<Task.Layouts.ListProps<C>, "listItem">,
    Pick<ListItemProps<C>, "name"> {}

export const DigitalChannelList = <C extends DigitalChannel>({
  name,
  ...rest
}: DigitalChannelListProps<C>) => {
  const listItem = useCallback(
    ({ key, ...p }: Task.ChannelListItemProps) => (
      <ListItem<C> key={key} {...p} name={name} />
    ),
    [name],
  );

  return <Task.Layouts.List<C> {...rest} listItem={listItem} />;
};
