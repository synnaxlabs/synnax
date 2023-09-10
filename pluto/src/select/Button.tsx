// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Key, type KeyedRenderableRecord } from "@synnaxlabs/x";

import { Align } from "@/align";
import { Button as CoreButton } from "@/button";
import {
  type UseSelectMultipleProps,
  useSelectMultiple,
} from "@/hooks/useSelectMultiple";
import { type Input } from "@/input";
import { type RenderProp } from "@/util/renderProp";

export interface ButtonOptionProps<
  K extends Key = Key,
  E extends KeyedRenderableRecord<K, E> = KeyedRenderableRecord<K>,
> extends Pick<CoreButton.ButtonProps, "onClick"> {
  key: K;
  selected: boolean;
  entry: E;
  title: E[keyof E];
}

export interface ButtonProps<
  K extends Key = Key,
  E extends KeyedRenderableRecord<K, E> = KeyedRenderableRecord<K>,
> extends Input.Control<K>,
    Omit<Align.PackProps, "children" | "onChange">,
    Pick<UseSelectMultipleProps, "allowNone" | "allowMultiple"> {
  children?: RenderProp<ButtonOptionProps<K, E>>;
  entryRenderKey?: keyof E;
  data: E[];
}

export const Button = <
  K extends Key = Key,
  E extends KeyedRenderableRecord<K, E> = KeyedRenderableRecord<K>,
>({
  children = defaultSelectButtonOption,
  value,
  onChange,
  entryRenderKey = "key",
  allowNone = false,
  allowMultiple = false,
  data,
  ...props
}: ButtonProps<K, E>): JSX.Element => {
  const { onSelect } = useSelectMultiple({
    allowMultiple,
    allowNone,
    data,
    value: [value],
    onChange: ([v]) => onChange(v),
  });

  return (
    <Align.Pack {...props}>
      {data.map((e) => {
        return children({
          key: e.key,
          onClick: () => onSelect(e.key),
          selected: e.key === value,
          entry: e,
          title: e[entryRenderKey],
        });
      })}
    </Align.Pack>
  );
};

const defaultSelectButtonOption = <
  K extends Key = Key,
  E extends KeyedRenderableRecord<K, E> = KeyedRenderableRecord<K>,
>({
  key,
  onClick,
  selected,
  title,
}: ButtonOptionProps<K, E>): JSX.Element => {
  return (
    <CoreButton.Button
      key={key}
      onClick={onClick}
      variant={selected ? "filled" : "outlined"}
    >
      {title}
    </CoreButton.Button>
  );
};
