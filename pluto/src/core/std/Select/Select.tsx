// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  FocusEventHandler,
  ReactElement,
  useCallback,
  useEffect,
  useRef,
  useState,
} from "react";

import { Icon } from "@synnaxlabs/media";
import { AsyncTermSearcher, Key, KeyedRenderableRecord } from "@synnaxlabs/x";

import { CSS } from "@/core/css";
import { Button, ButtonIconProps } from "@/core/std/Button";
import { Dropdown, DropdownProps } from "@/core/std/Dropdown";
import { InputControl, Input, InputProps } from "@/core/std/Input";
import { List, ListColumn, ListProps } from "@/core/std/List";
import { Pack } from "@/core/std/Pack";
import { SelectList } from "@/core/std/Select/SelectList";

export interface SelectProps<
  K extends Key = Key,
  E extends KeyedRenderableRecord<K, E> = KeyedRenderableRecord<K>
> extends Omit<DropdownProps, "onChange" | "visible" | "children">,
    InputControl<K>,
    Omit<ListProps<K, E>, "children"> {
  tagKey?: keyof E;
  columns?: Array<ListColumn<K, E>>;
  inputProps?: Omit<InputProps, "onChange">;
  searcher?: AsyncTermSearcher<string, E>;
  allowClear?: boolean;
}

export const Select = <
  K extends Key = Key,
  E extends KeyedRenderableRecord<K, E> = KeyedRenderableRecord<K>
>({
  onChange,
  value,
  tagKey = "key",
  columns = [],
  data = [],
  emptyContent,
  inputProps,
  allowClear = true,
  searcher,
  ...props
}: SelectProps<K, E>): ReactElement => {
  const { ref, visible, open, close } = Dropdown.use();
  const [stateData, setStateData] = useState<E[]>(data);
  data = searcher != null ? stateData : data;
  const initialValue = useRef<K>(value);

  const [selected, setSelected] = useState<E | null>(() => {
    return data.find((e) => e.key === value) ?? null;
  });

  const handleChange = useCallback(
    ([v]: readonly K[]): void => {
      close();
      if (v == null) {
        if (!allowClear) return;
        setSelected(null);
        return onChange(initialValue.current);
      }
      const e = data.find((e) => e.key === v) as E;
      setSelected(e);
      onChange(v);
    },
    [data, onChange, allowClear]
  );

  const input = ({ onChange }: InputControl<string>): ReactElement => (
    <SelectInput
      onChange={onChange}
      onFocus={open}
      selected={selected}
      tagKey={tagKey}
      visible={visible}
      allowClear={allowClear}
    />
  );

  const filterOrSearch =
    searcher != null ? (
      <List.Search searcher={searcher} onChange={setStateData}>
        {input}
      </List.Search>
    ) : (
      <List.Filter>{input}</List.Filter>
    );

  return (
    <List data={data} emptyContent={emptyContent}>
      <Dropdown ref={ref} visible={visible} {...props}>
        {filterOrSearch}
        <SelectList<K, E>
          value={[value]}
          onChange={handleChange}
          allowMultiple={false}
          columns={columns}
        />
      </Dropdown>
    </List>
  );
};

export interface SelectInputProps<K extends Key, E extends KeyedRenderableRecord<K, E>>
  extends Omit<InputProps, "value"> {
  tagKey: keyof E;
  selected: E | null;
  visible: boolean;
  debounceSearch?: number;
  allowClear?: boolean;
}

const SelectInput = <K extends Key, E extends KeyedRenderableRecord<K, E>>({
  tagKey,
  selected,
  visible,
  onChange,
  onFocus,
  allowClear = true,
  debounceSearch = 250,
  ...props
}: SelectInputProps<K, E>): ReactElement => {
  const {
    select: { clear },
  } = List.useContext();
  // We maintain our own value state for two reasons:
  //
  //  1. So we can avoid executing a search when the user selects an item and hides the
  //     dropdown.
  //  2. So that we can display the previous search results when the user focuses on the
  //       while still being able to clear the input value for searching.
  //
  const [internalValue, setInternalValue] = useState("");

  // Runs to set the value of the input to the item selected from the list.
  useEffect(() => {
    if (visible) return;
    const key = selected?.key;
    const isZero =
      selected == null ||
      (typeof key === "number" && key === 0) ||
      (typeof key === "string" && key.length === 0);
    if (isZero) return setInternalValue("");
    const v = selected?.[tagKey] as string | number;
    setInternalValue?.(v.toString());
  }, [selected, visible, tagKey]);

  const handleChange = (v: string): void => {
    onChange(v);
    setInternalValue(v);
  };

  const handleFocus: FocusEventHandler<HTMLInputElement> = (e) => {
    setInternalValue("");
    onFocus?.(e);
  };

  const handleClear = (): void => {
    setInternalValue("");
    clear?.();
  };

  const input = (
    <Input
      value={internalValue}
      onChange={handleChange}
      onFocus={handleFocus}
      style={{ flexGrow: 1 }}
      {...props}
    />
  );

  if (allowClear) {
    return (
      <Pack direction="x" className={CSS.BE("select", "input")}>
        {input}
        <SelectClearButton onClick={handleClear} />
      </Pack>
    );
  }

  return input;
};

export const SelectClearButton = (
  props: Omit<ButtonIconProps, "children">
): ReactElement => (
  <Button.Icon className={CSS.BE("select", "clear")} variant="outlined" {...props}>
    <Icon.Close aria-label="clear" />
  </Button.Icon>
);
