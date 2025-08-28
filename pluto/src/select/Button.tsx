// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/select/Button.css";

import { type record } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Button as CoreButton } from "@/button";
import { Flex } from "@/flex";
import { List } from "@/list";
import { Select } from "@/select";
import { type FrameProps, useContext, useItemState } from "@/select/Frame";

export interface ButtonsProps<
  K extends record.Key = record.Key,
  E extends record.Keyed<K> | undefined = record.Keyed<K>,
> extends Omit<Flex.BoxProps, "onSelect" | "onChange">,
    Omit<FrameProps<K, E>, "getItem" | "subscribe" | "data"> {
  keys: K[] | readonly K[];
}

export const Buttons = <K extends record.Key = record.Key>({
  keys,
  value,
  onChange,
  allowNone,
  multiple,
  ...rest
}: ButtonsProps<K>): ReactElement => {
  const listProps = List.useKeysData<K>(keys);
  return (
    <Select.Frame<K, record.Keyed<K>>
      closeDialogOnSelect={false}
      {...listProps}
      allowNone={allowNone}
      multiple={multiple}
      value={value as any}
      onChange={onChange as any}
    >
      <Flex.Box pack {...rest} />
    </Select.Frame>
  );
};

export interface ButtonProps<K extends record.Key = record.Key>
  extends Omit<CoreButton.ToggleProps, "onChange" | "value"> {
  itemKey: K;
}

export const Button = <K extends record.Key = record.Key>({
  itemKey,
  ...rest
}: ButtonProps<K>): ReactElement | null => {
  const { setSelected } = useContext();
  const { selected, onSelect } = useItemState<K>(itemKey);
  return (
    <CoreButton.Toggle
      {...rest}
      onChange={onSelect}
      value={selected}
      onContextMenu={(e) => {
        setSelected([itemKey]);
        e.preventDefault();
      }}
    />
  );
};
