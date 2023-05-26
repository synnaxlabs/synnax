// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { FocusEventHandler, ReactElement, useEffect, useState } from "react";

import { KeyedRenderableRecord } from "@synnaxlabs/x";

import { Dropdown, DropdownProps } from "@/core/std/Dropdown";
import { InputControl, Input, InputProps } from "@/core/std/Input";
import { List, ListColumn, ListProps } from "@/core/std/List";
import { SelectList } from "@/core/std/Select/SelectList";

export interface SelectProps<E extends KeyedRenderableRecord<E>>
  extends Omit<DropdownProps, "onChange" | "visible" | "children">,
    InputControl<string>,
    Omit<ListProps<E>, "children"> {
  tagKey?: keyof E;
  columns?: Array<ListColumn<E>>;
  inputProps?: Omit<InputProps, "onChange">;
}

export const Select = <E extends KeyedRenderableRecord<E>>({
  onChange,
  value,
  tagKey = "key",
  columns = [],
  data = [],
  emptyContent,
  inputProps,
  ...props
}: SelectProps<E>): ReactElement => {
  const { ref, visible, open, close } = Dropdown.use();

  const handleChange = ([key]: readonly string[]): void => {
    onChange(key);
    close();
  };

  return (
    <List data={data} emptyContent={emptyContent}>
      <Dropdown ref={ref} visible={visible} {...props}>
        <List.Search>
          {({ onChange }: InputProps) => (
            <SelectInput
              data={data}
              selected={value}
              tagKey={tagKey}
              onFocus={open}
              visible={visible}
              onChange={onChange}
              {...inputProps}
            />
          )}
        </List.Search>
        <SelectList
          value={[value]}
          onChange={handleChange}
          allowMultiple={false}
          columns={columns}
        />
      </Dropdown>
    </List>
  );
};

export interface SelectInputProps<E extends KeyedRenderableRecord<E>>
  extends Omit<InputProps, "value"> {
  tagKey: keyof E;
  selected: string;
  visible: boolean;
  data: E[];
  debounceSearch?: number;
}

const SelectInput = <E extends KeyedRenderableRecord<E>>({
  data,
  tagKey,
  selected,
  visible,
  onChange,
  onFocus,
  debounceSearch = 250,
  ...props
}: SelectInputProps<E>): ReactElement => {
  // We maintain our own value state for two reasons:
  //
  //  1. So we can avoid executing a search when the user selects an item and hides the
  //     dropdown.
  //  2. So that we can display the previous search results when the user focuses on the
  //       while still being able to clear the input value for searching.
  //
  const [value, setValue] = useState("");

  // Runs to set the value of the input to the item selected from the list.
  useEffect(() => {
    if (visible) return;
    if (selected == null || selected.length === 0) return setValue("");
    const e = data.find(({ key }) => key === selected);
    const v = e?.[tagKey] ?? selected;
    setValue?.(v as string);
  }, [selected, data, visible, tagKey]);

  const handleChange = (v: string): void => {
    onChange(v);
    setValue(v);
  };

  const handleFocus: FocusEventHandler<HTMLInputElement> = (e) => {
    setValue("");
    onFocus?.(e);
  };

  return (
    <Input value={value} onChange={handleChange} onFocus={handleFocus} {...props} />
  );
};
