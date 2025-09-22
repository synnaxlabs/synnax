// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type status } from "@synnaxlabs/client";
import {
  Flex,
  Form,
  Icon,
  Input,
  List,
  Menu,
  Select,
  Status,
  stopPropagation,
  Text,
} from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { ContextMenu } from "@/status/list/ContextMenu";

export interface ItemProps extends List.ItemProps<status.Key> {}

export const Item = (props: ItemProps): ReactElement | null => {
  const { itemKey } = props;
  const item = List.useItem<status.Key, status.Status>(itemKey);
  const { form } = Status.useForm({
    params: {},
    initialValues: item,
    autoSave: true,
    sync: true,
  });
  const { selected, onSelect } = Select.useItemState(itemKey);
  const { getItem } = List.useUtilContext<status.Key, status.Status>();
  if (getItem == null) throw new Error("getItem is null");
  const menuProps = Menu.useContextMenu();

  if (item == null) return null;
  const { name, time, variant, message } = item;

  return (
    <List.Item<status.Key>
      {...props}
      justify="between"
      selected={selected}
      rounded={!selected}
      onContextMenu={menuProps.open}
    >
      <Form.Form<typeof Status.formSchema> {...form}>
        <Menu.ContextMenu
          menu={(p) => <ContextMenu {...p} getItem={getItem} />}
          onClick={stopPropagation}
          {...menuProps}
        />
        <Flex.Box x empty>
          <Input.Checkbox
            value={selected}
            onChange={onSelect}
            size="medium"
            onClick={stopPropagation}
            ghost={!selected}
          />
          <Text.Text level="p" status={variant}>
            <Status.Indicator variant={variant} />
            {name}
          </Text.Text>
        </Flex.Box>
        <Flex.Box x align="center">
          <Text.Text level="small" status={variant}>
            {message}
          </Text.Text>
          <Text.DateTime level="p" color="gray" format="dateTime">
            {time}
          </Text.DateTime>
        </Flex.Box>
      </Form.Form>
    </List.Item>
  );
};
