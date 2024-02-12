// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useCallback, type ReactElement, useState } from "react";

import { Icon } from "@synnaxlabs/media";
import { type Key, type KeyedRenderableRecord } from "@synnaxlabs/x";

import { Align } from "@/align";
import { Button as CoreButton } from "@/button";
import { CSS } from "@/css";
import { Dropdown } from "@/dropdown";
import { type UseSelectProps, useSelect } from "@/hooks/useSelect";
import { type Input } from "@/input";
import { List as CoreList } from "@/list";
import { List } from "@/select/List";
import { componentRenderProp, type RenderProp } from "@/util/renderProp";

import "@/select/Button.css";

export interface ButtonOptionProps<
  K extends Key = Key,
  E extends KeyedRenderableRecord<K, E> = KeyedRenderableRecord<K>,
> extends Pick<CoreButton.ButtonProps, "onClick"> {
  key: K;
  selected: boolean;
  entry: E;
  title: E[keyof E];
}

export interface ButtonProps<
  K extends Key = Key,
  E extends KeyedRenderableRecord<K, E> = KeyedRenderableRecord<K>,
> extends Input.Control<K>,
    Omit<Align.PackProps, "children" | "onChange">,
    Pick<UseSelectProps, "allowNone" | "allowMultiple"> {
  children?: RenderProp<ButtonOptionProps<K, E>>;
  entryRenderKey?: keyof E;
  data: E[];
}

export const Button = <
  K extends Key = Key,
  E extends KeyedRenderableRecord<K, E> = KeyedRenderableRecord<K>,
>({
  children = defaultSelectButtonOption,
  value,
  onChange,
  entryRenderKey = "key",
  allowNone = false,
  allowMultiple = false,
  data,
  ...props
}: ButtonProps<K, E>): JSX.Element => {
  const { onSelect } = useSelect({
    allowMultiple,
    allowNone,
    data,
    value: [value],
    onChange: ([v]) => onChange(v),
  });

  return (
    <Align.Pack {...props}>
      {data.map((e) => {
        return children({
          key: e.key,
          onClick: () => onSelect(e.key),
          selected: e.key === value,
          entry: e,
          title: e[entryRenderKey],
        });
      })}
    </Align.Pack>
  );
};

const defaultSelectButtonOption = <
  K extends Key = Key,
  E extends KeyedRenderableRecord<K, E> = KeyedRenderableRecord<K>,
>({
  key,
  onClick,
  selected,
  title,
}: ButtonOptionProps<K, E>): JSX.Element => (
  <CoreButton.Button onClick={onClick} variant={selected ? "filled" : "outlined"}>
    {title}
  </CoreButton.Button>
);

export interface DropdownButtonButtonProps<
  K extends Key,
  E extends KeyedRenderableRecord<K, E>,
> extends CoreButton.ButtonProps {
  selected: E | null;
  renderKey: keyof E;
  toggle: () => void;
  visible: boolean;
}

export interface DropdownButtonProps<
  K extends Key,
  E extends KeyedRenderableRecord<K, E>,
> extends Omit<Dropdown.DialogProps, "onChange" | "visible" | "children">,
    Input.Control<K>,
    Omit<CoreList.ListProps<K, E>, "children">,
    Pick<CoreButton.ButtonProps, "disabled"> {
  columns?: Array<CoreList.ColumnSpec<K, E>>;
  children?: RenderProp<DropdownButtonButtonProps<K, E>>;
  renderKey?: keyof E;
  allowNone?: boolean;
  hideColumnHeader?: boolean;
  disabled?: boolean;
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
    className={CSS.B("select-button")}
    onClick={toggle}
    variant="outlined"
    endIcon={<Icon.Caret.Up className={CSS.BE("select-button", "indicator")} />}
    {...props}
  >
    {children ?? selected?.[renderKey]}
  </CoreButton.Button>
);

export const defaultButton: RenderProp<DropdownButtonButtonProps<any, any>> =
  componentRenderProp(BaseButton);

export const DropdownButton = <
  K extends Key = Key,
  E extends KeyedRenderableRecord<K, E> = KeyedRenderableRecord<K>,
>({
  data,
  value,
  columns = [],
  children = defaultButton,
  renderKey = "key",
  allowNone = false,
  onChange,
  disabled,
  hideColumnHeader = true,
}: DropdownButtonProps<K, E>): ReactElement => {
  const { ref, visible, toggle, close } = Dropdown.use();
  const [selected, setSelected] = useState<E | null>(data.find((e) => e.key === value));

  const handleChange: UseSelectProps<K, E>["onChange"] = useCallback(
    ([next]: K[], e): void => {
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

  return (
    <CoreList.List data={data}>
      <Dropdown.Dialog visible={visible} ref={ref} matchTriggerWidth>
        {children({
          selected,
          renderKey,
          toggle,
          visible,
          disabled,
        })}
        <List<K, E>
          visible={visible}
          value={[value]}
          onChange={handleChange}
          allowMultiple={false}
          allowNone={allowNone}
          columns={columns}
          hideColumnHeader={hideColumnHeader}
        />
      </Dropdown.Dialog>
    </CoreList.List>
  );
};
