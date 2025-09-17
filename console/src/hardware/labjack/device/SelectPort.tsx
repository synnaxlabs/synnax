// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Component, Dialog, Flex, List, Select, Text } from "@synnaxlabs/pluto";
import { type ReactNode } from "react";

import {
  type Model,
  type Port,
  PORTS,
  type PortType,
} from "@/hardware/labjack/device/types";

export interface SelectPortProps
  extends Omit<
      Select.SingleProps<string, Port | undefined>,
      "resourceName" | "data" | "children"
    >,
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
    <Select.ListItem {...props} align="center">
      <Text.Text style={{ width: 50 }}>{alias ?? key}</Text.Text>
      {alias != null && (
        <Text.Text level="small" color={10}>
          {key}
        </Text.Text>
      )}
    </Select.ListItem>
  );
});

export const SelectPort = ({
  model,
  portType,
  value,
  onChange,
  children,
  emptyContent,
  filter,
  triggerProps,
  variant,
  dialogProps,
  ...rest
}: SelectPortProps) => {
  const { data, getItem, retrieve } = List.useStaticData<string, Port>({
    data: PORTS[model][portType],
    filter,
  });
  const selected = getItem(value ?? "");
  const dialogVariant = variant === "preview" ? "connected" : variant;
  const triggerVariant = variant === "preview" ? "preview" : undefined;
  return (
    <Dialog.Frame variant={dialogVariant} {...rest}>
      <Select.Frame
        data={data}
        getItem={getItem}
        onChange={onChange}
        value={value}
        closeDialogOnSelect
      >
        <Flex.Box pack x>
          <Dialog.Trigger variant={triggerVariant} {...triggerProps}>
            {selected?.alias ?? selected?.key}
          </Dialog.Trigger>
          {children}
        </Flex.Box>
        <Select.Dialog<string>
          onSearch={(term) => retrieve({ searchTerm: term })}
          emptyContent={emptyContent}
          resourceName="Port"
          {...dialogProps}
        >
          {listItem}
        </Select.Dialog>
      </Select.Frame>
    </Dialog.Frame>
  );
};
