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
  Input,
  List,
  Select,
  Status,
  stopPropagation,
  Text,
} from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

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

  if (item == null) return null;
  const { name, time, variant, message } = item;

  return (
    <List.Item<status.Key> {...props} justify="between">
      <Form.Form<typeof Status.formSchema> {...form}>
        <Flex.Box x empty>
          <Input.Checkbox
            value={selected}
            onChange={onSelect}
            size="medium"
            onClick={stopPropagation}
            ghost={!selected}
          />
          <Text.Text level="p" weight={450}>
            {name}
          </Text.Text>
        </Flex.Box>
        <Flex.Box x align="center">
          <Text.DateTime level="small" color="gray" format="dateTime">
            {time}
          </Text.DateTime>
          <Text.Text level="p" status={variant}>
            {message}
            <Status.Indicator variant={variant} />
          </Text.Text>
        </Flex.Box>
      </Form.Form>
    </List.Item>
  );
};
