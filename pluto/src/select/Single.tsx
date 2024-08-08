// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/select/Single.css";

import {
  type AsyncTermSearcher,
  type Key,
  type Keyed,
  primitiveIsZero,
} from "@synnaxlabs/x";
import {
  type FocusEventHandler,
  type ReactElement,
  ReactNode,
  useCallback,
  useEffect,
  useMemo,
  useState,
} from "react";

import { Button } from "@/button";
import { CSS } from "@/css";
import { Dropdown } from "@/dropdown";
import { useAsyncEffect } from "@/hooks";
import { Input } from "@/input";
import { List as CoreList, List } from "@/list";
import {
  selectValueIsZero,
  type UseSelectOnChangeExtra,
  type UseSelectSingleProps,
} from "@/list/useSelect";
import { ClearButton } from "@/select/ClearButton";
import { Core } from "@/select/List";
import { Triggers } from "@/triggers";

export interface SingleProps<K extends Key, E extends Keyed<K>>
  extends Omit<UseSelectSingleProps<K, E>, "data" | "allowMultiple">,
    Omit<
      Dropdown.DialogProps,
      "onChange" | "visible" | "children" | "variant" | "close"
    >,
    Omit<CoreList.ListProps<K, E>, "children">,
    Pick<Input.TextProps, "variant" | "disabled">,
    Partial<Pick<CoreList.VirtualCoreProps<K, E>, "itemHeight">> {
  entryRenderKey?: keyof E | ((e: E) => string | number);
  columns?: Array<CoreList.ColumnSpec<K, E>>;
  inputProps?: Omit<Input.TextProps, "onChange">;
  searcher?: AsyncTermSearcher<string, K, E>;
  hideColumnHeader?: boolean;
  omit?: Array<K>;
  children?: List.VirtualCoreProps<K, E>["children"];
  dropdownVariant?: Dropdown.Variant;
  dropdownZIndex?: number;
  placeholder?: ReactNode;
  inputPlaceholder?: ReactNode;
  triggerTooltip?: ReactNode;
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
 * @param props.entryRenderKey - The option field rendered when selected. Defaults to "key".
 * @param props.location - Whether to render the dropdown above or below the select
 * component. Defaults to "below".
 * @param props.onChange - The callback to be invoked when the selected value changes.
 * @param props.value - The currently selected value.
 */
export const Single = <K extends Key = Key, E extends Keyed<K> = Keyed<K>>({
  onChange,
  value,
  entryRenderKey = "key",
  columns,
  data,
  emptyContent,
  inputProps,
  allowNone = true,
  searcher,
  className,
  variant,
  hideColumnHeader = false,
  disabled,
  omit,
  children,
  dropdownVariant = "connected",
  placeholder = DEFAULT_PLACEHOLDER,
  inputPlaceholder = placeholder,
  triggerTooltip,
  dropdownZIndex,
  ...props
}: SingleProps<K, E>): ReactElement => {
  const { visible, open, close, toggle } = Dropdown.use();
  const [selected, setSelected] = useState<E | null>(null);
  const searchMode = searcher != null;

  // This hook runs to make sure we have the selected entry populated when the value
  // changes externally.
  useAsyncEffect(async () => {
    if (selectValueIsZero(value)) return setSelected(null);
    if (selected?.key === value) return;
    let nextSelected: E | null = null;
    if (searchMode)
      // Wrap this in a try-except clause just in case the searcher throws an error.
      try {
        [nextSelected] = await searcher.retrieve([value]);
      } finally {
        // It might be undefined, so coalesce it to null.
        nextSelected ??= null;
      }
    else if (data != null) nextSelected = data.find((e) => e.key === value) ?? null;
    setSelected(nextSelected);
  }, [searcher, value, data]);

  const handleChange = useCallback(
    (v: K | K[] | null, e: UseSelectOnChangeExtra<K, E>): void => {
      if (Array.isArray(v)) return;
      setSelected(v == null ? null : e.entries[0]);
      close();
      onChange(v as K, e);
    },
    [onChange, allowNone],
  );

  const InputWrapper = useMemo(
    () => (searchMode ? CoreList.Search : CoreList.Filter),
    [searchMode],
  );

  const searchInput = (
    <InputWrapper<K, E> searcher={searcher}>
      {({ onChange: handleChange }) => (
        <SingleInput<K, E>
          size="large"
          autoFocus={dropdownVariant === "modal"}
          variant={variant}
          onChange={handleChange}
          onFocus={open}
          selected={selected}
          entryRenderKey={entryRenderKey}
          visible={visible}
          allowNone={allowNone}
          className={className}
          disabled={disabled}
          placeholder={inputPlaceholder}
        />
      )}
    </InputWrapper>
  );

  const buttonTrigger = (
    <Button.Button
      tooltip={triggerTooltip}
      variant="outlined"
      onClick={toggle}
      disabled={disabled}
    >
      {selected != null ? getRenderValue(entryRenderKey, selected) : placeholder}
    </Button.Button>
  );

  return (
    <Core<K, E>
      close={close}
      zIndex={dropdownZIndex}
      open={open}
      data={data}
      omit={omit}
      emptyContent={emptyContent}
      allowMultiple={false}
      visible={visible}
      value={value}
      hideColumnHeader={hideColumnHeader}
      onChange={handleChange}
      allowNone={allowNone}
      columns={columns}
      listItem={children}
      variant={dropdownVariant}
      trigger={dropdownVariant !== "modal" ? searchInput : buttonTrigger}
      extraDialogContent={dropdownVariant === "modal" ? searchInput : undefined}
      keepMounted={false}
      {...props}
    />
  );
};

export interface SelectInputProps<K extends Key, E extends Keyed<K>>
  extends Omit<Input.TextProps, "value" | "onFocus"> {
  entryRenderKey: keyof E | ((e: E) => string | number);
  selected: E | null;
  visible: boolean;
  debounceSearch?: number;
  allowNone?: boolean;
  onFocus: () => void;
}

export const DEFAULT_PLACEHOLDER = "Select";

const getRenderValue = <K extends Key, E extends Keyed<K>>(
  entryRenderKey: keyof E | ((e: E) => string | number | ReactNode),
  selected: E | null,
): ReactNode => {
  if (selected == null) return "";
  if (typeof entryRenderKey === "function") return entryRenderKey(selected);
  return (selected[entryRenderKey] as string | number).toString();
};

const SingleInput = <K extends Key, E extends Keyed<K>>({
  entryRenderKey,
  selected,
  visible,
  onChange,
  onFocus,
  allowNone = true,
  placeholder = DEFAULT_PLACEHOLDER,
  className,
  disabled,
  ...props
}: SelectInputProps<K, E>): ReactElement => {
  const { clear } = CoreList.useSelectionUtils();
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
    if (selected == null) return;
    setInternalValue(getRenderValue(entryRenderKey, selected) as string);
  }, [selected, visible, entryRenderKey]);

  const handleChange = (v: string): void => {
    onChange(v);
    setInternalValue(v);
  };

  const handleFocus: FocusEventHandler<HTMLInputElement> = () => {
    setInternalValue("");
    onFocus?.();
  };

  const handleClick: React.MouseEventHandler<HTMLInputElement> = (e) => {
    if (visible) return;
    e.preventDefault();
    onFocus?.();
  };

  const handleClear = (): void => {
    setInternalValue("");
    clear?.();
  };

  return (
    <Input.Text
      className={CSS(CSS.BE("select", "input"), className)}
      value={internalValue}
      onChange={handleChange}
      onFocus={handleFocus}
      onKeyDown={Triggers.matchCallback([["Enter"]], (e) => {
        e.preventDefault();
        if (visible) return;
        onFocus?.();
      })}
      style={{ flexGrow: 1 }}
      onClick={handleClick}
      placeholder={placeholder}
      disabled={disabled}
      {...props}
    >
      {allowNone && <ClearButton onClick={handleClear} disabled={disabled} />}
    </Input.Text>
  );
};
