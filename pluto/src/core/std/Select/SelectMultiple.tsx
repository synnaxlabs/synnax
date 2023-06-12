// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement, useEffect, useRef, useState } from "react";

import { Icon } from "@synnaxlabs/media";
import { convertRenderV, Key, KeyedRenderableRecord } from "@synnaxlabs/x";

import { Color } from "@/core/color";
import { CSS } from "@/core/css";
import { Button } from "@/core/std/Button";
import { Dropdown, DropdownProps } from "@/core/std/Dropdown";
import { Input, InputControl, InputProps } from "@/core/std/Input";
import { ListColumn, List, ListProps } from "@/core/std/List";
import { Pack } from "@/core/std/Pack";
import { SelectList } from "@/core/std/Select/SelectList";
import { Space } from "@/core/std/Space";
import { Tag } from "@/core/std/Tag";
import { Theming } from "@/core/theming";

import "@/core/std/Select/SelectMultiple.css";

export interface SelectMultipleProps<
  K extends Key = Key,
  E extends KeyedRenderableRecord<K, E> = KeyedRenderableRecord<K>
> extends Omit<DropdownProps, "visible" | "onChange" | "children">,
    InputControl<readonly K[]>,
    Omit<ListProps<K, E>, "children"> {
  columns?: Array<ListColumn<K, E>>;
  tagKey?: keyof E;
}

export const SelectMultiple = <
  K extends Key = Key,
  E extends KeyedRenderableRecord<K, E> = KeyedRenderableRecord<K>
>({
  onChange,
  value,
  location,
  data = [],
  columns = [],
  tagKey = "key",
  emptyContent,
  ...props
}: SelectMultipleProps<K, E>): ReactElement => {
  const { ref, visible, open } = Dropdown.use();
  return (
    <List data={data} emptyContent={emptyContent}>
      <Dropdown ref={ref} visible={visible} location={location} {...props}>
        <List.Search>
          {({ onChange, value: searchV }) => (
            <SelectMultipleInput<K, E>
              onChange={onChange}
              value={searchV}
              selected={value}
              onFocus={open}
              tagKey={tagKey}
              visible={visible}
            />
          )}
        </List.Search>
        <SelectList value={value} onChange={onChange} columns={columns} allowMultiple />
      </Dropdown>
    </List>
  );
};

interface SelectMultipleInputProps<K extends Key, E extends KeyedRenderableRecord<K, E>>
  extends Pick<InputProps, "onChange" | "onFocus" | "value"> {
  selected: readonly K[];
  tagKey: keyof E;
  visible: boolean;
}

const SelectMultipleInput = <K extends Key, E extends KeyedRenderableRecord<K, E>>({
  selected,
  onChange,
  onFocus,
  visible,
  tagKey,
  ...props
}: SelectMultipleInputProps<K, E>): ReactElement => {
  const {
    sourceData,
    select: { onSelect, clear },
  } = List.useContext<K, E>();
  const [value, setValue] = useState("");

  const { theme } = Theming.useContext();

  const ref = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (visible) ref.current?.focus();
    else setValue("");
  }, [visible, selected]);

  const handleChange = (v: string): void => {
    setValue(v);
    onChange(v);
  };

  return (
    <Pack align="stretch" {...props} grow>
      <Input
        ref={ref}
        className={CSS(CSS.BE("select-multiple", "input"), CSS.visible(visible))}
        placeholder="Search"
        value={value}
        onChange={handleChange}
        onFocus={onFocus}
      />
      <Space
        direction="x"
        className={CSS.BE("select-multiple", "tags")}
        align="center"
        grow
      >
        {selected
          ?.map((k) => sourceData.find((v) => v.key === k))
          .map((e, i) => {
            if (e == null) return null;
            return (
              <Tag
                key={e.key}
                color={new Color(theme.colors.visualization.palettes.default[i]).hex}
                onClose={() => onSelect?.(e.key)}
                size="small"
                variant="outlined"
              >
                {convertRenderV(e[tagKey])}
              </Tag>
            );
          })}
      </Space>
      <Button.Icon
        className={CSS.BE("select-multiple", "clear")}
        variant="outlined"
        onClick={clear}
      >
        <Icon.Close aria-label="clear" />
      </Button.Icon>
    </Pack>
  );
};
