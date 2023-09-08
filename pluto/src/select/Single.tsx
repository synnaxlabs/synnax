// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type FocusEventHandler,
  type ReactElement,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";

import {
  type AsyncTermSearcher,
  type Key,
  type KeyedRenderableRecord,
  primitiveIsZero,
} from "@synnaxlabs/x";

import { Align } from "@/align";
import { CSS } from "@/css";
import { Dropdown } from "@/dropdown";
import { useAsyncEffect } from "@/hooks";
import { type UseSelectMultipleProps } from "@/hooks/useSelectMultiple";
import { Input } from "@/input";
import { List as CoreList } from "@/list";
import { ClearButton } from "@/select/ClearButton";
import { List } from "@/select/List";

import "@/select/Single.css";

export interface SingleProps<
  K extends Key = Key,
  E extends KeyedRenderableRecord<K, E> = KeyedRenderableRecord<K>,
> extends Omit<Dropdown.DialogProps, "onChange" | "visible" | "children">,
    Input.Control<K>,
    Omit<CoreList.ListProps<K, E>, "children">,
    Pick<Input.TextProps, "variant"> {
  tagKey?: keyof E;
  columns?: Array<CoreList.ColumnSpec<K, E>>;
  inputProps?: Omit<Input.TextProps, "onChange">;
  searcher?: AsyncTermSearcher<string, K, E>;
  allowClear?: boolean;
}

/**
 * Allows a user to browse, search for, and select a value from a list of options.
 * It's important to note that Select maintains no internal selection state. The caller
 * must provide the selected value via the `value` prop and handle any changes via the
 * `onChange` prop.
 *
 * @param props - The props for the component. Any additional props will be passed to the
 * underlying input element.
 * @param props.data - The data to be used to populate the select options.
 * @param props.columns - The columns to be used to render the select options in the
 * dropdown. See the {@link ListColumn} type for more details on available options.
 * @param props.tagKey - The option field rendered when selected. Defaults to "key".
 * @param props.location - Whether to render the dropdown above or below the select
 * component. Defaults to "below".
 * @param props.onChange - The callback to be invoked when the selected value changes.
 * @param props.value - The currently selected value.
 */
export const Single = <
  K extends Key = Key,
  E extends KeyedRenderableRecord<K, E> = KeyedRenderableRecord<K>,
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
  variant,
  ...props
}: SingleProps<K, E>): ReactElement => {
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

  const handleChange: UseSelectMultipleProps<K, E>["onChange"] = useCallback(
    ([v], e): void => {
      close();
      if (v == null) {
        if (!allowClear) return;
        setSelected(null);
        return onChange(initialValue.current);
      }
      setSelected(e.entries[0]);
      onChange(v);
    },
    [onChange, allowClear],
  );

  const InputWrapper = useMemo(
    () => (searchMode ? CoreList.Search : CoreList.Filter),
    [searchMode],
  );

  return (
    <CoreList.List data={data} emptyContent={emptyContent}>
      <Dropdown.Dialog
        ref={ref}
        visible={visible}
        className={CSS.B("select")}
        matchTriggerWidth
        {...props}
      >
        {/* @ts-expect-error - searcher is undefined when List is List.Filter  */}
        <InputWrapper searcher={searcher}>
          {({ onChange }) => (
            <SingleInput
              variant={variant}
              onChange={onChange}
              onFocus={open}
              selected={selected}
              tagKey={tagKey}
              visible={visible}
              allowClear={allowClear}
              className={className}
            />
          )}
        </InputWrapper>
        <List<K, E>
          visible={visible}
          value={[value]}
          onChange={handleChange}
          allowMultiple={false}
          columns={columns}
        />
      </Dropdown.Dialog>
    </CoreList.List>
  );
};

export interface SelectInputProps<K extends Key, E extends KeyedRenderableRecord<K, E>>
  extends Omit<Input.TextProps, "value"> {
  tagKey: keyof E;
  selected: E | null;
  visible: boolean;
  debounceSearch?: number;
  allowClear?: boolean;
}

const SingleInput = <K extends Key, E extends KeyedRenderableRecord<K, E>>({
  tagKey,
  selected,
  visible,
  onChange,
  onFocus,
  allowClear = true,
  debounceSearch = 250,
  className,
  ...props
}: SelectInputProps<K, E>): ReactElement => {
  const {
    select: { clear },
  } = CoreList.useContext();
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
    <Input.Text
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
      <Align.Pack direction="x" className={CSS(className, CSS.B("select-input"))}>
        {input}
        <ClearButton onClick={handleClear} />
      </Align.Pack>
    );
  }

  return input;
};
