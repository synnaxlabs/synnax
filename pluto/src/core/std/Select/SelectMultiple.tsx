// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement, useCallback, useLayoutEffect, useRef, useState } from "react";

import {
  convertRenderV,
  Key,
  KeyedRenderableRecord,
  AsyncTermSearcher,
} from "@synnaxlabs/x";

import { Color } from "@/core/color";
import { CSS } from "@/core/css";
import { useMount } from "@/core/hooks/useMount";
import { Dropdown, DropdownProps } from "@/core/std/Dropdown";
import { Input, InputControl, InputProps } from "@/core/std/Input";
import { ListColumn, List, ListProps } from "@/core/std/List";
import { Pack } from "@/core/std/Pack";
import { SelectClearButton } from "@/core/std/Select/Select";
import { SelectList } from "@/core/std/Select/SelectList";
import { Space } from "@/core/std/Space";
import { Tag } from "@/core/std/Tag";
import { Theming } from "@/core/theming";

import "@/core/std/Select/SelectMultiple.css";

export interface SelectMultipleProps<
  K extends Key = Key,
  E extends KeyedRenderableRecord<K, E> = KeyedRenderableRecord<K>
> extends Omit<DropdownProps, "visible" | "onChange" | "children">,
    InputControl<readonly K[]>,
    Omit<ListProps<K, E>, "children"> {
  columns?: Array<ListColumn<K, E>>;
  searcher?: AsyncTermSearcher<string, K, E>;
  tagKey?: keyof E;
}

export const SelectMultiple = <
  K extends Key = Key,
  E extends KeyedRenderableRecord<K, E> = KeyedRenderableRecord<K>
>({
  onChange,
  value,
  location,
  data = [],
  columns = [],
  tagKey = "key",
  emptyContent,
  searcher,
  ...props
}: SelectMultipleProps<K, E>): ReactElement => {
  const { ref, visible, open } = Dropdown.use();
  const [stateData, setStateData] = useState<E[]>(data);
  data = searcher != null ? stateData : data;

  const [selected, setSelected] = useState<readonly E[]>(() => {
    if (value == null) return [];
    return data.filter((e) => value.includes(e.key));
  });

  useMount(() => {
    if (searcher == null) return;
    searcher
      .retrieve(value as K[])
      .then((e) => setSelected(e))
      .catch(console.error);
  });

  const handleChange = useCallback(
    (v: readonly K[]) => {
      setSelected((p) =>
        [
          ...p.filter((e) => v.includes(e.key)),
          ...data.filter((e) => v.includes(e.key)),
        ].filter((e, i, a) => a.indexOf(e) === i)
      );
      onChange(v);
    },
    [data, onChange]
  );

  const input = ({ onChange }: InputControl<string>): ReactElement => (
    <SelectMultipleInput<K, E>
      onChange={onChange}
      selected={selected}
      onFocus={open}
      tagKey={tagKey}
      visible={visible}
    />
  );

  const filterOrSearch =
    searcher != null ? (
      <List.Search searcher={searcher} onChange={setStateData}>
        {input}
      </List.Search>
    ) : (
      <List.Filter>{input}</List.Filter>
    );

  return (
    <List data={data} emptyContent={emptyContent}>
      <Dropdown ref={ref} visible={visible} location={location} {...props}>
        {filterOrSearch}
        <SelectList
          value={value}
          onChange={handleChange}
          columns={columns}
          allowMultiple
        />
      </Dropdown>
    </List>
  );
};

interface SelectMultipleInputProps<K extends Key, E extends KeyedRenderableRecord<K, E>>
  extends Pick<InputProps, "onChange" | "onFocus"> {
  selected: readonly E[];
  tagKey: keyof E;
  visible: boolean;
}

const SelectMultipleInput = <K extends Key, E extends KeyedRenderableRecord<K, E>>({
  selected,
  onChange,
  onFocus,
  visible,
  tagKey,
  ...props
}: SelectMultipleInputProps<K, E>): ReactElement => {
  const {
    select: { onSelect, clear },
  } = List.useContext<K, E>();
  const [value, setValue] = useState("");

  const { theme } = Theming.useContext();

  const ref = useRef<HTMLInputElement>(null);

  useLayoutEffect(() => {
    if (visible) ref.current?.focus();
    // Notice how we don't call onChange with an empty value here. This is so
    // we preserve the previous search result in the list even after we clear
    // the box when a value is selected.
    else setValue("");
  }, [visible, selected]);

  const handleChange = (v: string): void => {
    setValue(v);
    onChange(v);
  };

  return (
    <Pack align="stretch" {...props} grow className={CSS.B("pluto-select-multiple")}>
      <Input
        ref={ref}
        className={CSS(CSS.BE("select-multiple", "input"), CSS.visible(visible))}
        placeholder="Search"
        value={value}
        onChange={handleChange}
        onFocus={onFocus}
      />
      <Space
        direction="x"
        className={CSS.BE("select-multiple", "tags")}
        align="center"
        grow
      >
        {selected.map((e, i) => {
          if (e == null) return null;
          return (
            <Tag
              key={e.key}
              color={new Color(theme.colors.visualization.palettes.default[i]).hex}
              onClose={() => onSelect?.(e.key)}
              size="small"
              variant="outlined"
              draggable
            >
              {convertRenderV(e[tagKey])}
            </Tag>
          );
        })}
      </Space>
      <SelectClearButton onClick={clear} />
    </Pack>
  );
};
