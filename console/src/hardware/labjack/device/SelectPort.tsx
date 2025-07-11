// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Align, Component, Dialog, List, Select, Text } from "@synnaxlabs/pluto";
import { type ReactNode } from "react";

import {
  type Model,
  type Port,
  PORTS,
  type PortType,
} from "@/hardware/labjack/device/types";

export interface SelectPortProps
  extends Select.SingleProps<string, Port | undefined>,
    Omit<List.UseStaticDataArgs<string, Port>, "data"> {
  model: Model;
  portType: PortType;
  children?: ReactNode;
}

const listItem = Component.renderProp((props: List.ItemProps<string>) => {
  const port = List.useItem<string, Port>(props.itemKey);
  if (port == null) return null;
  const { alias, key } = port;
  return (
    <List.Item {...props}>
      <Text.Text level="p" shade={11}>
        {alias ?? key}
      </Text.Text>
      {alias != null && (
        <Text.Text level="small" shade={10}>
          {key}
        </Text.Text>
      )}
    </List.Item>
  );
});

export const SelectPort = ({
  model,
  portType,
  value,
  onChange,
  children,
  allowNone,
  emptyContent,
  filter,
  ...rest
}: SelectPortProps) => {
  const { data, useListItem, retrieve } = List.useStaticData<string, Port>({
    data: PORTS[model][portType],
    filter,
  });
  const selected = useListItem(value);
  return (
    <Dialog.Frame {...rest}>
      <Select.Frame data={data} useListItem={useListItem} onChange={onChange}>
        <Align.Pack x>
          <Dialog.Trigger>{selected?.alias ?? selected?.key}</Dialog.Trigger>
          {children}
        </Align.Pack>
        <Select.Dialog<string, List.RetrieveParams>
          onSearch={retrieve}
          searchPlaceholder="Search Ports..."
          emptyContent={emptyContent}
        >
          {listItem}
        </Select.Dialog>
      </Select.Frame>
    </Dialog.Frame>
  );
};
