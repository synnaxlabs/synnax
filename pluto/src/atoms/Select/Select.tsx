import React, { FocusEventHandler, useEffect, useState } from "react";

import clsx from "clsx";

import { Dropdown, DropdownProps } from "../Dropdown";
import { InputControlProps, Input, InputProps } from "../Input";
import { List, ListColumn } from "../List";

import { SelectList } from "./SelectList";

import { visibleCls } from "@/util/css";
import { RenderableRecord } from "@/util/record";

import "./Select.css";

export interface SelectProps<E extends RenderableRecord<E>>
  extends Pick<DropdownProps, "location">,
    InputControlProps<string> {
  data?: E[];
  tagKey?: keyof E;
  columns: Array<ListColumn<E>>;
}

export const Select = <E extends RenderableRecord<E>>({
  value,
  onChange,
  columns,
  tagKey = "key",
  data = [],
  location,
  ...props
}: SelectProps<E>): JSX.Element => {
  const { ref, visible, onFocus, setVisible } = Dropdown.use();

  const handleChange = ([key]: readonly string[]): void => {
    onChange(key);
    setVisible(false);
  };

  return (
    <List data={data}>
      <Dropdown {...props} location={location} ref={ref} visible={visible}>
        <List.Search>
          {(props: InputProps) => (
            <SelectInput
              data={data}
              selected={value}
              tagKey={tagKey}
              onFocus={onFocus}
              visible={visible}
              {...props}
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

export interface SelectInputProps<E extends RenderableRecord<E>> extends InputProps {
  tagKey: keyof E;
  selected: string;
  visible: boolean;
  data: E[];
}

const SelectInput = <E extends RenderableRecord<E>>({
  data,
  tagKey,
  selected,
  visible,
  className,
  value: propsValue,
  onChange: propsOnChange,
  onFocus: propsOnFocus,
  ...props
}: SelectInputProps<E>): JSX.Element => {
  const [value, setValue] = useState("");

  // Runs to set the value of the input to the item selected from the list.
  useEffect(() => {
    if (selected == null || selected.length === 0 || visible) return;
    const e = data.find(({ key }) => key === selected);
    const v = e?.[tagKey] ?? selected;
    setValue?.(v as string);
  }, [selected, data, visible, tagKey]);

  const handleChange = (v: string): void => {
    propsOnChange?.(v);
    setValue(v);
  };

  const handleFocus: FocusEventHandler<HTMLInputElement> = (e) => {
    setValue("");
    propsOnFocus?.(e);
  };

  return (
    <Input
      className={clsx(`pluto-select__input--${visibleCls(visible)}`, className)}
      value={value}
      onChange={handleChange}
      onFocus={handleFocus}
      {...props}
    />
  );
};
