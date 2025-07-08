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

export interface SelectPortProps extends Select.SingleProps<string> {
  model: Model;
  portType: PortType;
  children: ReactNode;
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
  ...rest
}: SelectPortProps) => {
  const { data, useItem } = List.useStaticData<string, Port>(PORTS[model][portType]);
  const listProps = List.use({ data });
  const selectProps = Select.useSingle({
    data,
    allowNone: false,
    value,
    onChange,
  });
  const selected = useItem(value);
  return (
    <Select.Dialog
      {...selectProps}
      {...listProps}
      data={data}
      useItem={useItem}
      {...rest}
    >
      <Align.Pack x>
        <Dialog.Trigger>{selected?.alias ?? selected?.key}</Dialog.Trigger>
        {children}
      </Align.Pack>
      <Dialog.Content>
        <List.Items<string, Port> {...listProps}>{listItem}</List.Items>
      </Dialog.Content>
    </Select.Dialog>
  );
};
