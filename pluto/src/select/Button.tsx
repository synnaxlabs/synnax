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
import { Provider, type ProviderProps, useItemState } from "@/select/Provider";

export interface ButtonsProps<K extends record.Key = record.Key>
  extends Omit<Align.PackProps, "onSelect">,
    ProviderProps<K> {}

export const Buttons = <K extends record.Key = record.Key>({
  value,
  onSelect,
  clear,
  ...rest
}: ButtonsProps<K>): ReactElement => (
  <Provider value={value} onSelect={onSelect} clear={clear}>
    <Align.Pack {...rest} />
  </Provider>
);

export const Button = <K extends record.Key = record.Key>({
  itemKey,
  ...rest
}: Omit<CoreButton.ToggleProps, "onChange" | "value"> & {
  itemKey: K;
}): ReactElement => {
  const [selected, handleSelect] = useItemState<K>(itemKey);
  return <CoreButton.Toggle {...rest} onChange={handleSelect} value={selected} />;
};

export const ButtonIcon = <K extends record.Key = record.Key>({
  itemKey,
  ...rest
}: Omit<CoreButton.ToggleIconProps, "onChange" | "value"> & {
  itemKey: K;
}): ReactElement => {
  const [selected, handleSelect] = useItemState<K>(itemKey);
  return <CoreButton.ToggleIcon {...rest} onChange={handleSelect} value={selected} />;
};
