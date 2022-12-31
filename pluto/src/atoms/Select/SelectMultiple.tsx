// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useEffect, useRef, useState } from "react";

import clsx from "clsx";
import { AiOutlineClose } from "react-icons/ai";

import { Theming } from "../../theming";
import { Dropdown, DropdownProps } from "../Dropdown";
import { Group } from "../Group/Group";
import { ListColumn } from "../List";

import { SelectList } from "./SelectList";

import { Button } from "@/atoms/Button";
import { Input, InputControlProps, InputProps } from "@/atoms/Input";
import { List } from "@/atoms/List";
import { Space } from "@/atoms/Space";
import { Tag } from "@/atoms/Tag";
import { visibleCls } from "@/util/css";
import { RenderableRecord } from "@/util/record";
import { render } from "@/util/renderable";

import "./SelectMultiple.css";

export interface SelectMultipleProps<E extends RenderableRecord<E>>
  extends Omit<
      DropdownProps,
      "visible" | "setVisible" | "ref" | "children" | "onChange"
    >,
    InputControlProps<readonly string[]> {
  data?: E[];
  tagKey?: keyof E;
  columns: Array<ListColumn<E>>;
}

export const SelectMultiple = <E extends RenderableRecord<E>>({
  data = [],
  columns = [],
  tagKey = "key",
  value,
  onChange,
  ...props
}: SelectMultipleProps<E>): JSX.Element => {
  const { ref, visible, onFocus } = Dropdown.use();
  return (
    <List data={data}>
      <Dropdown {...props} ref={ref} visible={visible}>
        <List.Search>
          {({ onChange }) => (
            <SelectMultipleInput<E>
              tagKey={tagKey}
              selected={value}
              onChange={onChange}
              onFocus={onFocus}
              visible={visible}
            />
          )}
        </List.Search>
        <SelectList value={value} onChange={onChange} columns={columns} allowMultiple />
      </Dropdown>
    </List>
  );
};

interface SelectMultipleInputProps<E extends RenderableRecord<E>>
  extends Pick<InputProps, "onChange" | "onFocus"> {
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
    <Group align="stretch" grow>
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
    </Group>
  );
};
