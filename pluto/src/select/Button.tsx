// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/select/Button.css";

import { type Key, type Keyed } from "@synnaxlabs/x";
import {
  type ReactElement,
  type ReactNode,
  useCallback,
  useEffect,
  useState,
} from "react";

import { Align } from "@/align";
import { Button as CoreButton } from "@/button";
import { Caret } from "@/caret";
import { CSS } from "@/css";
import { Dropdown } from "@/dropdown";
import { type Input } from "@/input";
import { type List as CoreList } from "@/list";
import {
  useSelect,
  type UseSelectOnChangeExtra,
  type UseSelectProps,
} from "@/list/useSelect";
import { Core } from "@/select/List";
import { type ComponentSize } from "@/util/component";
import { componentRenderProp, type RenderProp } from "@/util/renderProp";

export interface ButtonOptionProps<K extends Key = Key, E extends Keyed<K> = Keyed<K>>
  extends Pick<CoreButton.ButtonProps, "onClick" | "size"> {
  key: K;
  selected: boolean;
  entry: E;
  title: E[keyof E];
}

export type ButtonProps<K extends Key = Key, E extends Keyed<K> = Keyed<K>> = Omit<
  UseSelectProps<K, E>,
  "data"
> &
  Omit<Align.PackProps, "children" | "onChange" | "size"> & {
    data?: E[];
    children?: RenderProp<ButtonOptionProps<K, E>>;
    entryRenderKey?: keyof E;
    size?: ComponentSize;
  };

export const Button = <K extends Key = Key, E extends Keyed<K> = Keyed<K>>({
  children = defaultSelectButtonOption,
  value,
  onChange,
  entryRenderKey = "key",
  allowNone = false,
  allowMultiple = false,
  data,
  replaceOnSingle,
  className,
  size = "small",
  ...props
}: ButtonProps<K, E>): JSX.Element => {
  const { onSelect } = useSelect<K, E>({
    allowMultiple,
    allowNone,
    replaceOnSingle,
    data,
    value,
    onChange,
  } as const as UseSelectProps<K, E>);

  return (
    <Align.Pack
      borderShade={4}
      className={CSS(CSS.B("select-button"), className)}
      size={size}
      {...props}
    >
      {data?.map((e) =>
        children({
          key: e.key,
          onClick: () => onSelect(e.key),
          size,
          selected: e.key === value,
          entry: e,
          title: e[entryRenderKey],
        }),
      )}
    </Align.Pack>
  );
};

const defaultSelectButtonOption = <K extends Key = Key, E extends Keyed<K> = Keyed<K>>({
  key,
  onClick,
  selected,
  title,
}: ButtonOptionProps<K, E>): JSX.Element => (
  <CoreButton.Button
    key={key}
    onClick={onClick}
    variant={selected ? "filled" : "outlined"}
    size="small"
  >
    {title as ReactNode}
  </CoreButton.Button>
);

export interface DropdownButtonButtonProps<K extends Key, E extends Keyed<K>>
  extends CoreButton.ButtonProps {
  selected: E | null;
  renderKey: keyof E;
  toggle: () => void;
  visible: boolean;
}

export interface DropdownButtonProps<K extends Key, E extends Keyed<K>>
  extends Omit<
      Dropdown.DialogProps,
      "onChange" | "visible" | "children" | "close" | "variant"
    >,
    Input.Control<K>,
    Omit<CoreList.ListProps<K, E>, "children">,
    Pick<CoreButton.ButtonProps, "disabled" | "variant"> {
  columns?: Array<CoreList.ColumnSpec<K, E>>;
  children?: RenderProp<DropdownButtonButtonProps<K, E>>;
  entryRenderKey?: keyof E;
  dropdownVariant?: Dropdown.Variant;
  allowNone?: boolean;
  hideColumnHeader?: boolean;
  disabled?: boolean;
  omit?: K[];
}

export const BaseButton = ({
  selected,
  renderKey,
  toggle,
  visible,
  children,
  ...props
}: DropdownButtonButtonProps<any, any>): ReactElement => (
  <CoreButton.Button
    className={CSS.B("select-dropdown-button")}
    onClick={toggle}
    variant="outlined"
    endIcon={
      <Caret.Animated enabledLoc="bottom" disabledLoc="left" enabled={visible} />
    }
    {...props}
  >
    {children ?? selected?.[renderKey]}
  </CoreButton.Button>
);

export const defaultButton: RenderProp<DropdownButtonButtonProps<any, any>> =
  componentRenderProp(BaseButton);

export const DropdownButton = <K extends Key = Key, E extends Keyed<K> = Keyed<K>>({
  data,
  value,
  columns = [],
  children = defaultButton,
  entryRenderKey = "key",
  allowNone = false,
  onChange,
  disabled,
  hideColumnHeader = true,
  variant,
  dropdownVariant,
  ...props
}: DropdownButtonProps<K, E>): ReactElement => {
  const { close, visible, toggle } = Dropdown.use();
  const [selected, setSelected] = useState<E | null>(
    data?.find((e) => e.key === value) ?? null,
  );

  useEffect(() => {
    setSelected(data?.find((e) => e.key === value) ?? null);
  }, [data, value]);

  const handleChange = useCallback(
    (next: K | K[] | null, e: UseSelectOnChangeExtra<K, E>): void => {
      if (Array.isArray(next) || next === null) return;
      close();
      if (next == null) {
        setSelected(null);
        return onChange(value);
      }
      setSelected(e.entries[0]);
      onChange(next);
    },
    [onChange, value, close, setSelected],
  );

  const childrenProps: DropdownButtonButtonProps<K, E> = {
    selected,
    renderKey: entryRenderKey,
    toggle,
    visible,
    disabled,
  };
  if (variant != null) childrenProps.variant = variant;

  return (
    <Core<K, E>
      close={close}
      data={data}
      visible={visible}
      value={value}
      onChange={handleChange}
      allowMultiple={false}
      allowNone={allowNone}
      columns={columns}
      hideColumnHeader={hideColumnHeader}
      variant={dropdownVariant}
      trigger={<>{children(childrenProps)}</>}
      {...props}
    />
  );
};
