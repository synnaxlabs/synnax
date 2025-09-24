// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Dialog, Flex, List, Select } from "@synnaxlabs/pluto";
import { type record } from "@synnaxlabs/x";
import { type ReactNode } from "react";

import {
  COIL_INPUT_TYPE,
  DISCRETE_INPUT_TYPE,
  HOLDING_REGISTER_INPUT_TYPE,
  type InputChannelType,
  REGISTER_INPUT_TYPE,
} from "@/hardware/modbus/task/types";

export type InputChannelTypeEntry = record.KeyedNamed<InputChannelType>;

const INPUT_CHANNEL_TYPES: InputChannelTypeEntry[] = [
  { key: COIL_INPUT_TYPE, name: "Coil" },
  { key: DISCRETE_INPUT_TYPE, name: "Discrete" },
  { key: HOLDING_REGISTER_INPUT_TYPE, name: "Holding Register" },
  { key: REGISTER_INPUT_TYPE, name: "Register" },
];

export interface SelectInputChannelTypeProps
  extends Omit<
    Select.SingleProps<InputChannelType, InputChannelTypeEntry>,
    "data" | "resourceName" | "children"
  > {
  children?: ReactNode;
}

export const SelectInputChannelType = ({
  value,
  onChange,
  variant,
  children,
  triggerProps,
  dialogProps,
  emptyContent,
  ...rest
}: SelectInputChannelTypeProps) => {
  const { data, getItem, retrieve } = List.useStaticData<
    InputChannelType,
    InputChannelTypeEntry
  >({
    data: INPUT_CHANNEL_TYPES,
  });
  const selected = getItem(value ?? "coil_input");
  const dialogVariant = variant === "preview" ? "connected" : variant;
  const triggerVariant = variant === "preview" ? "preview" : undefined;
  return (
    <Dialog.Frame location="bottom" variant={dialogVariant} {...rest}>
      <Select.Frame<InputChannelType, InputChannelTypeEntry>
        data={data}
        getItem={getItem}
        onChange={onChange}
        value={value}
        closeDialogOnSelect
      >
        <Flex.Box pack x>
          <Dialog.Trigger variant={triggerVariant} {...triggerProps}>
            {selected?.name}
          </Dialog.Trigger>
          {children}
        </Flex.Box>
        <Select.Dialog<string>
          onSearch={(term) => retrieve({ searchTerm: term })}
          emptyContent={emptyContent}
          resourceName="Channel Type"
          {...dialogProps}
        >
          {Select.staticListItem}
        </Select.Dialog>
      </Select.Frame>
    </Dialog.Frame>
  );
};
