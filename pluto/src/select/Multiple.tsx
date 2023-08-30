// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  ReactElement,
  useCallback,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
} from "react";

import {
  convertRenderV,
  Key,
  KeyedRenderableRecord,
  AsyncTermSearcher,
  Compare,
} from "@synnaxlabs/x";

import { Align } from "@/align";
import { Color } from "@/color";
import { CSS } from "@/css";
import { Dropdown } from "@/dropdown";
import { useAsyncEffect } from "@/hooks";
import { UseSelectMultipleProps } from "@/hooks/useSelectMultiple";
import { Input } from "@/input";
import { List as CoreList } from "@/list";
import { ClearButton } from "@/select/ClearButton";
import { List } from "@/select/List";
import { Tag } from "@/tag";
import { RenderProp, componentRenderProp } from "@/util/renderProp";

import "@/select/Multiple.css";

export interface MultipleProps<
  K extends Key = Key,
  E extends KeyedRenderableRecord<K, E> = KeyedRenderableRecord<K>
> extends Omit<Dropdown.DialogProps, "visible" | "onChange" | "children">,
    Input.Control<K[]>,
    Omit<CoreList.ListProps<K, E>, "children">,
    Pick<Input.TextProps, "placeholder"> {
  columns?: Array<CoreList.ColumnSpec<K, E>>;
  searcher?: AsyncTermSearcher<string, K, E>;
  tagKey?: keyof E;
  renderTag?: RenderProp<SelectMultipleTagProps<K, E>>;
  onTagDragStart?: (e: React.DragEvent<HTMLDivElement>, key: K) => void;
  onTagDragEnd?: (e: React.DragEvent<HTMLDivElement>, key: K) => void;
}

/**
 * Allows a user to browse, search for, and select multiple values from a list of
 * options. It's important to note that Select maintains no internal selection state.
 * The caller must provide the selected value via the `value` prop and handle any
 * changes via the `onChange` prop.
 *
 * @param props - The props for the component. Any additional props will be passed to
 * the input group containing the selection input and the selected tags.
 * @param props.data - The data to be used to populate the select options.
 * @param props.columns - The columns to be used to render the select options in the
 * dropdown. See the {@link ListColumn} type for more details on how to configure
 * columns.
 * @param props.tagKey - The option field rendered for each tag when selected in the
 * input group. Defaults to "key".
 * @param props.location - Whether to render the dropdown above or below the select
 * component. Defaults to "below".
 * @param props.onChange - The callback to be invoked when the selected value changes.
 * @param props.value - The currently selected value.
 */
export const Multiple = <
  K extends Key = Key,
  E extends KeyedRenderableRecord<K, E> = KeyedRenderableRecord<K>
>({
  onChange,
  value,
  location,
  className,
  data,
  columns = [],
  tagKey = "key",
  emptyContent,
  searcher,
  renderTag,
  placeholder,
  onTagDragStart,
  onTagDragEnd,
  style,
  ...props
}: MultipleProps<K, E>): ReactElement => {
  const { ref, visible, open } = Dropdown.use();
  const [selected, setSelected] = useState<readonly E[]>([]);
  const searchMode = searcher != null;

  useAsyncEffect(async () => {
    const selectedKeys = selected.map((v) => v.key);
    if (value.length === 0) return setSelected([]);
    if (Compare.primitiveArrays(selectedKeys, value) === Compare.EQUAL) return;
    const e = searchMode
      ? await searcher.retrieve(value)
      : data?.filter((v) => value.includes(v.key)) ?? [];
    setSelected(e);
  }, [searcher, searchMode, value, data]);

  const handleChange: UseSelectMultipleProps<K, E>["onChange"] = useCallback(
    (v, extra) => {
      setSelected(extra.entries);
      onChange(v);
    },
    [onChange]
  );

  const InputWrapper = useMemo(
    () => (searchMode ? CoreList.Search : CoreList.Filter),
    [searchMode]
  );

  return (
    <CoreList.List data={data} emptyContent={emptyContent}>
      <Dropdown.Dialog
        ref={ref}
        visible={visible}
        location={location}
        {...props}
        matchTriggerWidth
      >
        {/* @ts-expect-error - searcher is undefined when List is List.Filter  */}
        <InputWrapper searcher={searcher}>
          {({ onChange }) => (
            <MultipleInput<K, E>
              className={className}
              onChange={onChange}
              selected={selected}
              onFocus={open}
              tagKey={tagKey}
              visible={visible}
              renderTag={renderTag}
              placeholder={placeholder}
              onTagDragStart={onTagDragStart}
              onTagDragEnd={onTagDragEnd}
              style={style}
            />
          )}
        </InputWrapper>
        <List
          visible={visible}
          value={value}
          onChange={handleChange}
          columns={columns}
          allowMultiple
        />
      </Dropdown.Dialog>
    </CoreList.List>
  );
};

interface SelectMultipleInputProps<K extends Key, E extends KeyedRenderableRecord<K, E>>
  extends Omit<Input.TextProps, "value"> {
  selected: readonly E[];
  tagKey: keyof E;
  visible: boolean;
  renderTag?: RenderProp<SelectMultipleTagProps<K, E>>;
  onTagDragStart?: (e: React.DragEvent<HTMLDivElement>, key: K) => void;
  onTagDragEnd?: (e: React.DragEvent<HTMLDivElement>, key: K) => void;
}

const MultipleInput = <K extends Key, E extends KeyedRenderableRecord<K, E>>({
  selected,
  onChange,
  onFocus,
  visible,
  tagKey,
  renderTag = componentRenderProp(SelectMultipleTag),
  placeholder = "Search...",
  onTagDragStart,
  onTagDragEnd,
  className,
  ...props
}: SelectMultipleInputProps<K, E>): ReactElement => {
  const {
    select: { onSelect, clear },
  } = CoreList.useContext<K, E>();
  const [value, setValue] = useState("");
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
    <Align.Pack
      align="stretch"
      {...props}
      grow
      className={CSS(CSS.B("select-multiple"), className)}
    >
      <Input.Text
        ref={ref}
        className={CSS(CSS.BE("select-multiple", "input"), CSS.visible(visible))}
        placeholder={placeholder}
        value={value}
        onChange={handleChange}
        onFocus={onFocus}
      />
      <Align.Space
        direction="x"
        className={CSS.BE("select-multiple", "tags")}
        align="center"
        grow
      >
        {selected.map((e, i) =>
          renderTag({
            key: e.key,
            tagKey,
            entry: e,
            color: "var(--pluto-border-color)",
            onClose: () => onSelect?.(e.key),
            onDragStart: (ev) => onTagDragStart?.(ev, e.key),
            onDragEnd: (ev) => onTagDragEnd?.(ev, e.key),
          })
        )}
      </Align.Space>
      <ClearButton onClick={clear} />
    </Align.Pack>
  );
};

interface SelectMultipleTagProps<K extends Key, E extends KeyedRenderableRecord<K, E>> {
  key: K;
  tagKey: keyof E;
  entry: E;
  color: Color.Crude;
  onClose?: () => void;
  onDragStart: (e: React.DragEvent<HTMLDivElement>) => void;
  onDragEnd?: (e: React.DragEvent<HTMLDivElement>) => void;
}

const SelectMultipleTag = <K extends Key, E extends KeyedRenderableRecord<K, E>>({
  tagKey,
  entry,
  ...props
}: SelectMultipleTagProps<K, E>): ReactElement => (
  <Tag.Tag size="small" variant="outlined" draggable {...props}>
    {convertRenderV(entry[tagKey])}
  </Tag.Tag>
);
