// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useEffect, useRef, useState } from "react";




import { Button } from "@/core/Button";

import clsx from "clsx";

import { Dropdown, DropdownProps } from "@/core/Dropdown";

import { AiOutlineClose } from "react-icons/ai";

import { Input, InputControlProps, InputProps } from "@/core/Input";
import { ListColumn, List } from "@/core/List";
import { Pack, PackProps } from "@/core/Pack";
import { Space } from "@/core/Space";
import { Tag } from "@/core/Tag";

import { Theming } from "../../theming";

import { visibleCls } from "@/util/css";
import { RenderableRecord } from "@/util/record";
import { render } from "@/util/renderable";

import { SelectList } from "./SelectList";

import "./SelectMultiple.css";

type SelectMultipleInputGroupProps = Omit<
  PackProps,
  "onChange" | "onFocus" | "children"
>;

export interface SelectMultipleProps<E extends RenderableRecord<E>>
  extends Pick<DropdownProps, "location">,
    InputControlProps<readonly string[]>,
    SelectMultipleInputGroupProps {
  data?: E[];
  columns?: Array<ListColumn<E>>;
  tagKey?: keyof E;
}

export const SelectMultiple = <E extends RenderableRecord<E>>({
  data = [],
  columns = [],
  tagKey = "key",
  value,
  onChange,
  location,
  ...props
}: SelectMultipleProps<E>): JSX.Element => {
  const { ref, visible, onFocus } = Dropdown.use();
  return (
    <List data={data}>
      <Dropdown ref={ref} visible={visible} location={location}>
        <List.Search>
          {({ onChange }) => (
            <SelectMultipleInput<E>
              tagKey={tagKey}
              selected={value}
              onChange={onChange}
              onFocus={onFocus}
              visible={visible}
              {...props}
            />
          )}
        </List.Search>
        <SelectList value={value} onChange={onChange} columns={columns} allowMultiple />
      </Dropdown>
    </List>
  );
};

interface SelectMultipleInputProps<E extends RenderableRecord<E>>
  extends Pick<InputProps, "onChange" | "onFocus">,
    SelectMultipleInputGroupProps {
  selected: readonly string[];
  tagKey: keyof E;
  visible: boolean;
}

const SelectMultipleInput = <E extends RenderableRecord<E>>({
  selected,
  onChange,
  onFocus,
  visible,
  tagKey,
  ...props
}: SelectMultipleInputProps<E>): JSX.Element => {
  const {
    sourceData,
    select: { onSelect, clear },
  } = List.useContext<E>();
  const [value, setValue] = useState("");

  const { theme } = Theming.useContext();

  const ref = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (visible) ref.current?.focus();
    else setValue("");
  });

  const handleChange = (v: string): void => {
    setValue(v);
    onChange(v);
  };

  return (
    <Pack align="stretch" {...props} grow>
      <Input
        ref={ref}
        className={clsx(
          "pluto-select-multiple__input",
          `pluto-select__input--${visibleCls(visible)}`
        )}
        placeholder="Search"
        value={value}
        onChange={handleChange}
        onFocus={onFocus}
      />
      <Space
        direction="horizontal"
        className="pluto-select-multiple__tags"
        align="center"
        grow
      >
        {selected
          ?.map((k) => sourceData.find((v) => v.key === k))
          .map((e, i) => {
            if (e == null) return null;
            return (
              <Tag
                key={e.key}
                color={theme.colors.visualization.palettes.default[i]}
                onClose={() => onSelect?.(e.key)}
                size="small"
                variant="outlined"
              >
                {render(e[tagKey])}
              </Tag>
            );
          })}
      </Space>
      <Button.IconOnly
        className={clsx(
          "pluto-select-multiple__clear",
          `pluto-select-multiple__clear--${visibleCls(visible)}`
        )}
        variant="outlined"
        onClick={clear}
      >
        <AiOutlineClose aria-label="clear" />
      </Button.IconOnly>
    </Pack>
  );
};
