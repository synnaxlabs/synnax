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
  useMemo,
  useRef,
  useState,
} from "react";

import {
  AsyncTermSearcher,
  Key,
  KeyedRenderableRecord,
  primitiveIsZero,
} from "@synnaxlabs/x";

import { CSS } from "@/core/css";
import { useAsyncEffect } from "@/core/hooks";
import { Dropdown, DropdownProps } from "@/core/std/Dropdown";
import { InputControl, Input, InputProps } from "@/core/std/Input";
import { List, ListColumn, ListProps } from "@/core/std/List";
import { Pack } from "@/core/std/Pack";
import { SelectClearButton } from "@/core/std/Select/SelectClearButton";
import { SelectList } from "@/core/std/Select/SelectList";

import "@/core/std/Select/Select.css";

export interface SelectProps<
  K extends Key = Key,
  E extends KeyedRenderableRecord<K, E> = KeyedRenderableRecord<K>
> extends Omit<DropdownProps, "onChange" | "visible" | "children">,
    InputControl<K>,
    Omit<ListProps<K, E>, "children"> {
  tagKey?: keyof E;
  columns?: Array<ListColumn<K, E>>;
  inputProps?: Omit<InputProps, "onChange">;
  searcher?: AsyncTermSearcher<string, K, E>;
  allowClear?: boolean;
}

const { Filter, Search } = List;

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
  className,
  ...props
}: SelectProps<K, E>): ReactElement => {
  const { ref, visible, open, close } = Dropdown.use();
  const initialValue = useRef<K>(value);
  const [selected, setSelected] = useState<E | null>(null);
  const searchMode = searcher != null;

  useAsyncEffect(async () => {
    if (searcher == null || selected?.key === value || primitiveIsZero(value)) return;
    const [e] = await searcher.retrieve([value]);
    setSelected(e ?? null);
  }, [searcher, value]);

  useEffect(() => {
    if (selected?.key === value) return;
    setSelected(data.find((e) => e.key === value) ?? null);
  }, [value]);

  const handleChange = useCallback(
    ([v]: readonly K[], [e]: E[]): void => {
      close();
      if (v == null) {
        if (!allowClear) return;
        setSelected(null);
        return onChange(initialValue.current);
      }
      setSelected(e);
      onChange(v);
    },
    [onChange, allowClear]
  );

  const InputWrapper = useMemo(() => (searchMode ? Search : Filter), [searchMode]);

  return (
    <List data={data} emptyContent={emptyContent}>
      <Dropdown
        ref={ref}
        visible={visible}
        className={CSS(className, CSS.B("select"))}
        matchTriggerWidth
        {...props}
      >
        {/* @ts-expect-error - searcher is undefined when List is List.Filter  */}
        <InputWrapper searcher={searcher}>
          {({ onChange }) => (
            <SelectInput
              onChange={onChange}
              onFocus={open}
              selected={selected}
              tagKey={tagKey}
              visible={visible}
              allowClear={allowClear}
            />
          )}
        </InputWrapper>
        <SelectList<K, E>
          visible={visible}
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
    if (primitiveIsZero(selected?.key)) return setInternalValue("");
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
      className={CSS.BE("select", "input")}
      value={internalValue}
      onChange={handleChange}
      onFocus={handleFocus}
      style={{ flexGrow: 1 }}
      {...props}
    />
  );

  if (allowClear) {
    return (
      <Pack direction="x" className={CSS.B("select-input")}>
        {input}
        <SelectClearButton onClick={handleClear} />
      </Pack>
    );
  }

  return input;
};
