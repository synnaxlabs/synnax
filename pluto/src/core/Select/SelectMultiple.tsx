// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useEffect, useRef, useState } from "react";

import { Icon } from "@synnaxlabs/media";
import { convertRenderV, KeyedRenderableRecord } from "@synnaxlabs/x";

import { Button } from "@/core/Button";
import { Dropdown, DropdownProps } from "@/core/Dropdown";
import { Input, InputControl, InputProps } from "@/core/Input";
import { ListColumn, List, ListProps } from "@/core/List";
import { Pack } from "@/core/Pack";
import { SelectList } from "@/core/Select/SelectList";
import "@/core/Select/SelectMultiple.css";
import { Space } from "@/core/Space";
import { Tag } from "@/core/Tag";
import { CSS } from "@/css";
import { Theming } from "@/theming";

export interface SelectMultipleProps<E extends KeyedRenderableRecord<E>>
  extends Omit<DropdownProps, "visible" | "onChange" | "children">,
    InputControl<readonly string[]>,
    Omit<ListProps<E>, "children"> {
  columns?: Array<ListColumn<E>>;
  tagKey?: keyof E;
}

export const SelectMultiple = <E extends KeyedRenderableRecord<E>>({
  onChange,
  value,
  location,
  data = [],
  columns = [],
  tagKey = "key",
  emptyContent,
  ...props
}: SelectMultipleProps<E>): JSX.Element => {
  const { ref, visible, open } = Dropdown.use();
  return (
    <List data={data} emptyContent={emptyContent}>
      <Dropdown ref={ref} visible={visible} location={location} {...props}>
        <List.Search>
          {({ onChange, value: searchV }) => (
            <SelectMultipleInput<E>
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

interface SelectMultipleInputProps<E extends KeyedRenderableRecord<E>>
  extends Pick<InputProps, "onChange" | "onFocus" | "value"> {
  selected: readonly string[];
  tagKey: keyof E;
  visible: boolean;
}

const SelectMultipleInput = <E extends KeyedRenderableRecord<E>>({
  selected,
  onChange,
  onFocus,
  visible,
  tagKey,
  ...props
}: SelectMultipleInputProps<E>): JSX.Element => {
  const {
    sourceData,
    select: { onSelect, clear },
  } = List.useContext<E>();
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
                color={theme.colors.visualization.palettes.default[i]}
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
