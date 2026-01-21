// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type arc } from "@synnaxlabs/client";
import {
  Arc,
  Flex,
  Form,
  Input,
  List,
  Menu,
  Select,
  stopPropagation,
  Text,
} from "@synnaxlabs/pluto";
import { useMemo } from "react";

import { ContextMenu } from "@/arc/list/ContextMenu";

export interface ItemProps extends List.ItemProps<arc.Key> {
  showStatus?: boolean;
}

export const Item = ({ showStatus: _, ...props }: ItemProps) => {
  const { itemKey } = props;
  const { getItem } = List.useUtilContext<arc.Key, arc.Arc>();
  if (getItem == null) throw new Error("getItem is null");
  const arc = List.useItem<arc.Key, arc.Arc>(itemKey);
  const { onSelect, selected, ...selectProps } = Select.useItemState(itemKey);
  const initialValues = useMemo(() => {
    if (arc == null) return undefined;
    return {
      key: arc.key,
      name: arc.name,
      graph: arc.graph,
      text: arc.text,
      version: arc.version,
    };
  }, [arc]);

  if (initialValues == null || arc == null) return null;

  const { form } = Arc.useForm({
    query: { key: itemKey },
    initialValues,
    sync: true,
    autoSave: true,
  });
  const { name } = arc;

  const menuProps = Menu.useContextMenu();

  return (
    <List.Item
      {...props}
      {...selectProps}
      selected={selected}
      rounded={!selected}
      onSelect={onSelect}
      onContextMenu={menuProps.open}
      justify="between"
      align="center"
    >
      <Form.Form<typeof Arc.formSchema> {...form}>
        <Menu.ContextMenu
          menu={(p) => <ContextMenu {...p} getItem={getItem} />}
          onClick={stopPropagation}
          {...menuProps}
        />
        <Flex.Box x align="center">
          <Input.Checkbox
            value={selected}
            onChange={onSelect}
            onClick={stopPropagation}
          />
          <Text.Text level="p">{name}</Text.Text>
        </Flex.Box>
      </Form.Form>
    </List.Item>
  );
};
