// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/select/Single.css";

import { type Optional, type record } from "@synnaxlabs/x";
import { type ReactElement, type ReactNode, useEffect, useState } from "react";

import { Align } from "@/align";
import { CSS } from "@/css";
import { Dropdown } from "@/dropdown";
import { Input } from "@/input";
import { List } from "@/list";
import { Provider } from "@/select/Provider";
import { use, type UseSelectSingleProps } from "@/select/use";
import { type RenderProp } from "@/util/renderProp";

export interface TriggerProps<
  K extends record.Key,
  E extends record.Keyed<K> | undefined,
> {
  value: K | null;
  useItem: (key: K) => E;
  onClick: () => void;
}

export interface SingleProps<
  K extends record.Key,
  E extends record.Keyed<K> | undefined,
> extends Omit<UseSelectSingleProps<K>, "data" | "allowMultiple">,
    Omit<Dropdown.DialogProps, "visible" | "close" | "children" | "onChange">,
    List.UseProps<K>,
    Pick<List.ListProps<K, E>, "useItem"> {
  inputProps?: Partial<Omit<Input.TextProps, "onChange">>;
  children: [RenderProp<TriggerProps<K, E>>, List.ItemRenderProp<K, E>];
  onSearch?: (term: string) => void;
  dropdownVariant?: Dropdown.Variant;
  dropdownZIndex?: number;
  placeholder?: ReactNode;
  inputPlaceholder?: ReactNode;
  actions?: Input.ExtensionProps["children"];
}

// Point of select is to wrap together button, selector, list, dropdown, and search
// input.
// Takes two children -

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
export const Single = <
  K extends record.Key = record.Key,
  E extends record.Keyed<K> | undefined = record.Keyed<K>,
>({
  onChange,
  value,
  data,
  inputProps,
  className,
  children,
  dropdownVariant = "connected",
  placeholder = DEFAULT_PLACEHOLDER,
  inputPlaceholder = placeholder,
  dropdownZIndex,
  location,
  keepMounted,
  actions,
  onSearch,
  allowNone = true,
  useItem,
  ...rest
}: SingleProps<K, E>): ReactElement => {
  const { visible, close } = Dropdown.use();
  const { onSelect } = use<K>({
    data,
    value,
    onChange,
    allowMultiple: false,
    allowNone,
  });
  const listProps = List.use({ data });
  const [triggerRenderProp, listItemRenderProp] = children;
  return (
    <Provider value={value} onSelect={onSelect}>
      <Dropdown.Dialog
        visible={visible}
        close={close}
        location={location}
        keepMounted={keepMounted}
        variant={dropdownVariant}
        zIndex={dropdownZIndex}
        {...rest}
      >
        {triggerRenderProp({ value, useItem, onClick: open })}
        <Align.Space empty>
          {onSearch != null && (
            <SingleInput
              visible={visible}
              onChange={onSearch}
              placeholder={inputPlaceholder}
            />
          )}
          <List.List data={data} useItem={useItem} {...listProps}>
            {listItemRenderProp}
          </List.List>
        </Align.Space>
      </Dropdown.Dialog>
    </Provider>
  );
};

export interface SelectInputProps
  extends Optional<Omit<Input.TextProps, "value" | "onFocus">, "onChange"> {
  visible: boolean;
}

export const DEFAULT_PLACEHOLDER = "Select";

const SingleInput = ({
  visible,
  onChange,
  placeholder = DEFAULT_PLACEHOLDER,
  className,
  disabled,
  children,
  ...rest
}: SelectInputProps): ReactElement => {
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
    setInternalValue("");
  }, [visible]);

  const handleChange = (v: string): void => {
    onChange?.(v);
    setInternalValue(v);
  };

  return (
    <Input.Text
      className={CSS(CSS.BE("select", "input"), className)}
      value={internalValue}
      onChange={handleChange}
      style={{ flexGrow: 1 }}
      placeholder={placeholder}
      disabled={disabled}
      {...rest}
    >
      {children}
    </Input.Text>
  );
};
