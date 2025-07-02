// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/select/Single.css";

import { primitive, type record } from "@synnaxlabs/x";
import {
  type FocusEventHandler,
  type ReactElement,
  type ReactNode,
  useCallback,
  useEffect,
  useState,
} from "react";

import { Caret } from "@/caret";
import { CSS } from "@/css";
import { Dropdown } from "@/dropdown";
import { Input } from "@/input";
import { List as CoreList, List } from "@/list";
import { ClearButton } from "@/select/ClearButton";
import { Core } from "@/select/List";
import { type UseSelectOnChangeExtra, type UseSelectSingleProps } from "@/select/use";
import { Triggers } from "@/triggers";

export interface SingleProps<K extends record.Key>
  extends Omit<UseSelectSingleProps<K>, "data" | "allowMultiple">,
    Omit<
      Dropdown.DialogProps,
      "onChange" | "visible" | "children" | "variant" | "close"
    >,
    Omit<CoreList.ListProps<K>, "children">,
    Pick<Input.TextProps, "variant" | "disabled" {
  entryRenderKey?: keyof E | ((e: E) => string | number | ReactNode);
  inputProps?: Partial<Omit<Input.TextProps, "onChange">>;
  children?: List.ItemRenderProp<K>;
  dropdownVariant?: Dropdown.Variant;
  dropdownZIndex?: number;
  placeholder?: ReactNode;
  inputPlaceholder?: ReactNode;
  triggerTooltip?: ReactNode;
  actions?: Input.ExtensionProps["children"];
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
export const Single = <K extends record.Key = record.Key>({
  onChange,
  value,
  entryRenderKey = "key",
  data,
  inputProps,
  allowNone = true,
  className,
  variant = "button",
  disabled,
  children,
  dropdownVariant = "connected",
  placeholder = DEFAULT_PLACEHOLDER,
  inputPlaceholder = placeholder,
  triggerTooltip,
  dropdownZIndex,
  actions,
  ...rest
}: SingleProps<K>): ReactElement => {
  const { visible, open, close } = Dropdown.use();

  const handleChange = useCallback(
    (v: K | null, e: UseSelectOnChangeExtra<K>): void => {
      close();
      onChange(v, e);
    },
    [onChange, close],
  );

  const { onSelect } = List.use<K>({
    value,
    data,
    onChange: handleChange,
    allowNone,
    allowMultiple: false,
  });

  const searchInput = (
    <SingleInput<K, E>
      {...inputProps}
      autoFocus={dropdownVariant === "modal"}
      variant={variant}
      onChange={handleChange}
      onFocus={open}
      entryRenderKey={entryRenderKey}
      visible={visible}
      allowNone={allowNone}
      className={className}
      dropdownVariant={dropdownVariant}
      disabled={disabled}
      placeholder={inputPlaceholder}
    >
      {actions}
    </SingleInput>
  );

  return (
    <Core<K, E>
      close={close}
      zIndex={dropdownZIndex}
      data={data}
      allowMultiple={false}
      visible={visible}
      value={value}
      onChange={handleChange}
      allowNone={allowNone}
      listItem={children}
      variant={dropdownVariant}
      trigger={dropdownVariant !== "modal" ? searchInput : buttonTrigger}
      extraDialogContent={dropdownVariant === "modal" ? searchInput : undefined}
      keepMounted={false}
      {...rest}
    />
  );
};

export interface SelectInputProps<K extends record.Key, E extends record.Keyed<K>>
  extends Omit<Input.TextProps, "value" | "onFocus"> {
  entryRenderKey: keyof E | ((e: E) => string | number | ReactNode);
  selected: E | null;
  visible: boolean;
  debounceSearch?: number;
  allowNone?: boolean;
  onFocus: () => void;
  dropdownVariant?: Dropdown.Variant;
  zIndex?: number;
}

export const DEFAULT_PLACEHOLDER = "Select";

const getRenderValue = <K extends record.Key, E extends record.Keyed<K>>(
  entryRenderKey: keyof E | ((e: E) => string | number | ReactNode),
  selected: E | null,
): ReactNode => {
  if (selected == null) return "";
  if (typeof entryRenderKey === "function") return entryRenderKey(selected);
  return (selected[entryRenderKey] as string | number).toString();
};

const SingleInput = <K extends record.Key, E extends record.Keyed<K>>({
  entryRenderKey,
  selected,
  visible,
  onChange,
  onFocus,
  allowNone = true,
  placeholder = DEFAULT_PLACEHOLDER,
  className,
  disabled,
  dropdownVariant,
  children,
  ...rest
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
    if (primitive.isZero(selected?.key)) return setInternalValue("");
    if (selected == null) return;
    setInternalValue(getRenderValue(entryRenderKey, selected) as string);
  }, [selected, visible, entryRenderKey]);

  const handleChange = (v: string): void => {
    onChange(v);
    setInternalValue(v);
  };

  const handleFocus: FocusEventHandler<HTMLInputElement> = () => {
    // Trigger an onChange to make sure the parent component is aware of the focus event.
    if (internalValue === "") onChange("");
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

  let endContent: ReactElement | undefined;
  if (dropdownVariant !== "modal")
    endContent = (
      <Caret.Animated enabledLoc="bottom" disabledLoc="left" enabled={visible} />
    );

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
      endContent={endContent}
      style={{ flexGrow: 1 }}
      onClick={handleClick}
      placeholder={placeholder}
      disabled={disabled}
      {...rest}
    >
      {children}
      {allowNone && <ClearButton onClick={handleClear} disabled={disabled} />}
    </Input.Text>
  );
};
