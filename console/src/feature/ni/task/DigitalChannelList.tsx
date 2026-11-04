// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/ni/task/DigitalChannelList.css";

import { type Component } from "@synnaxlabs/lyra/component";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { deep } from "@synnaxlabs/x";

import { type DigitalChannel } from "@/feature/ni/task/types";
import { CSS } from "@/platform/css";
import { Task } from "@/platform/task";

interface ListItemProps<C extends DigitalChannel> extends Omit<
  Task.ChannelListItemProps,
  "name"
> {
  path: deep.Path<C[]>;
  name: Component.RenderProp<DigitalNameComponentProps<C>>;
}

export type DigitalNameComponentProps<C extends DigitalChannel> = Omit<C, "key"> & {
  itemKey: string;
  path: deep.Path<C>;
};

const PORT_INPUT_PROPS = { showDragHandle: false, "aria-label": "Port" };

const LINE_INPUT_PROPS = { showDragHandle: false, "aria-label": "Line" };

const ListItem = <C extends DigitalChannel>({
  path,
  name,
  ...rest
}: ListItemProps<C>) => {
  const entry = path[rest.itemKey];
  const channel = Form.useFieldValue(entry);
  const { port, line, disabled } = deep.widen<DigitalChannel>(entry);
  if (channel == null) return null;
  return (
    <Select.Item {...rest} align="center" justify="between" full="x">
      <Flex.Box align="center" x justify="evenly">
        <Flex.Box pack align="center" className="port-line-input" x>
          <Form.NumericField
            inputProps={PORT_INPUT_PROPS}
            showLabel={false}
            showHelpText={false}
            path={port}
          />
          <Text.Text color={9} weight={550}>
            /
          </Text.Text>
          <Form.NumericField
            inputProps={LINE_INPUT_PROPS}
            showLabel={false}
            showHelpText={false}
            path={line}
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
        {name({ ...channel, itemKey: rest.itemKey, path: entry })}
        <Task.EnabledCheckbox path={disabled} />
      </Flex.Box>
    </Select.Item>
  );
};

export interface DigitalChannelListProps<C extends DigitalChannel>
  extends Omit<Task.Views.ListProps<C>, "listItem">, Pick<ListItemProps<C>, "name"> {}

export const DigitalChannelList = <C extends DigitalChannel>({
  path,
  name,
  ...rest
}: DigitalChannelListProps<C>) => {
  const listItem = ({ key, ...p }: Task.ChannelListItemProps) => (
    <ListItem<C> key={key} {...p} path={path} name={name} />
  );

  return <Task.Views.List<C> {...rest} path={path} listItem={listItem} />;
};
