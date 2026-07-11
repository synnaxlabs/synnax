// Copyright 2026 Synnax Labs, Inc.
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

import { Button as Base } from "@/button";
import { CSS } from "@/css";
import { Flex } from "@/flex";
import { List } from "@/list";
import { CONTEXT_SELECTED, CONTEXT_TARGET } from "@/menu/types";
import { Select } from "@/select/base";
import { Frame, type FrameProps } from "@/select/Frame";

export interface ButtonsProps<
  K extends record.Key = record.Key,
  E extends record.Keyed<K> | undefined = record.Keyed<K>,
>
  extends
    Omit<Flex.BoxProps, "onSelect" | "onChange">,
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
  // Type assertion here because there are weird unions from these being widened to
  // their full types and then TS not being able to prove that they are compatible.
  const selectionProps = {
    allowNone,
    multiple,
    value,
    onChange,
  } as FrameProps<K, record.Keyed<K>>;
  return (
    <Frame<K, record.Keyed<K>>
      closeDialogOnSelect={false}
      {...listProps}
      {...selectionProps}
    >
      <Flex.Box pack {...rest} />
    </Frame>
  );
};

export interface ButtonProps<K extends record.Key = record.Key> extends Omit<
  Base.ToggleProps,
  "onChange" | "value"
> {
  itemKey: K;
}

export const Button = <K extends record.Key = record.Key>({
  itemKey,
  className,
  ...rest
}: ButtonProps<K>): ReactElement | null => {
  const { selected, onSelect } = Select.useItemState<K>(itemKey);
  return (
    <Base.Toggle
      {...rest}
      id={itemKey.toString()}
      onChange={onSelect}
      value={selected}
      className={CSS(className, selected && CONTEXT_SELECTED, CONTEXT_TARGET)}
    />
  );
};
