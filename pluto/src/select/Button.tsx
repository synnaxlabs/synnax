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

import { Align } from "@/align";
import { Button as CoreButton } from "@/button";
import { List } from "@/list";
import { Select } from "@/select";
import { type SingleFrameProps, useItemState } from "@/select/Frame";

export interface ButtonsProps<
  K extends record.Key = record.Key,
  E extends record.Keyed<K> | undefined = record.Keyed<K>,
> extends Omit<Align.PackProps, "onSelect" | "onChange">,
    Omit<SingleFrameProps<K, E>, "getItem" | "subscribe" | "data"> {
  keys: K[] | readonly K[];
}

export const Buttons = <K extends record.Key = record.Key>({
  value,
  onChange,
  keys,
  ...rest
}: ButtonsProps<K>): ReactElement => {
  const listProps = List.useKeysData<K>(keys);
  return (
    <Select.Frame<K, record.Keyed<K>>
      value={value}
      onChange={onChange}
      multiple={false}
      closeDialogOnSelect={false}
      {...listProps}
    >
      <Align.Pack {...rest} />
    </Select.Frame>
  );
};

export const Button = <K extends record.Key = record.Key>({
  itemKey,
  ...rest
}: Omit<CoreButton.ToggleProps, "onChange" | "value"> & {
  itemKey: K;
}): ReactElement | null => {
  const { selected, onSelect } = useItemState<K>(itemKey);
  return <CoreButton.Toggle {...rest} onChange={onSelect} value={selected} />;
};

export const ButtonIcon = <K extends record.Key = record.Key>({
  itemKey,
  ...rest
}: Omit<CoreButton.ToggleIconProps, "onChange" | "value"> & {
  itemKey: K;
}): ReactElement | null => {
  const { selected, onSelect } = useItemState<K>(itemKey);
  return <CoreButton.ToggleIcon {...rest} value={selected} onChange={onSelect} />;
};
