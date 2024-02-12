// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type ReactElement,
  useCallback,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
} from "react";

import {
  convertRenderV,
  type Key,
  type KeyedRenderableRecord,
  type AsyncTermSearcher,
  compare,
  toArray,
  type RenderableValue,
} from "@synnaxlabs/x";

import { Align } from "@/align";
import { type Color } from "@/color";
import { CSS } from "@/css";
import { Dropdown } from "@/dropdown";
import { useAsyncEffect } from "@/hooks";
import { selectValueIsZero, type UseSelectMultipleProps } from "@/hooks/useSelect";
import { Input } from "@/input";
import { List as CoreList } from "@/list";
import { ClearButton } from "@/select/ClearButton";
import { List } from "@/select/List";
import { Tag } from "@/tag";
import { type RenderProp, componentRenderProp } from "@/util/renderProp";

import "@/select/Multiple.css";

export interface MultipleProps<
  K extends Key = Key,
  E extends KeyedRenderableRecord<K, E> = KeyedRenderableRecord<K>,
> extends Omit<Dropdown.DialogProps, "visible" | "onChange" | "children">,
    Omit<UseSelectMultipleProps<K, E>, "data">,
    Omit<CoreList.ListProps<K, E>, "children">,
    Pick<Input.TextProps, "placeholder"> {
  columns?: Array<CoreList.ColumnSpec<K, E>>;
  searcher?: AsyncTermSearcher<string, K, E>;
  tagKey?: keyof E | ((e: E) => string | number);
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
  E extends KeyedRenderableRecord<K, E> = KeyedRenderableRecord<K>,
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
  allowMultiple,
  allowNone,
  replaceOnSingle,
  onTagDragEnd,
  style,
  ...props
}: MultipleProps<K, E>): ReactElement => {
  const { ref, visible, open } = Dropdown.use();
  const [selected, setSelected] = useState<readonly E[]>([]);
  const searchMode = searcher != null;

  // This hook makes sure we have the selected entries fetched to render their tags
  // properly.
  useAsyncEffect(async () => {
    if (selectValueIsZero(value)) setSelected([]);
    const inSelected = selected.map((v) => v.key);
    const nextValue = toArray(value);
    if (compare.unorderedPrimitiveArrays(inSelected, nextValue) === compare.EQUAL)
      return;
    let nextSelected: E[] = [];
    if (searchMode) nextSelected = await searcher.retrieve(nextValue);
    else if (data != null) nextSelected = data.filter((v) => nextValue.includes(v.key));
    setSelected(nextSelected);
  }, [searcher, searchMode, value, data]);

  const handleChange: UseSelectMultipleProps<K, E>["onChange"] = useCallback(
    (v, extra) => {
      setSelected(extra.entries);
      onChange(v, extra);
    },
    [onChange],
  );

  const InputWrapper = useMemo(
    () => (searchMode ? CoreList.Search : CoreList.Filter),
    [searchMode],
  );

  return (
    <CoreList.List<K, E> data={data} emptyContent={emptyContent}>
      <Dropdown.Dialog
        ref={ref}
        visible={visible}
        location={location}
        {...props}
        matchTriggerWidth
      >
        <InputWrapper<K, E> searcher={searcher}>
          {({ onChange, value: inputValue }) => (
            <MultipleInput<K, E>
              value={inputValue}
              selectedKeys={value}
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
        <List<K, E>
          visible={visible}
          value={value}
          onChange={handleChange}
          allowNone={allowNone}
          replaceOnSingle={replaceOnSingle}
          columns={columns}
          allowMultiple
        />
      </Dropdown.Dialog>
    </CoreList.List>
  );
};

interface SelectMultipleInputProps<K extends Key, E extends KeyedRenderableRecord<K, E>>
  extends Input.TextProps {
  selectedKeys: K | K[];
  selected: readonly E[];
  tagKey: keyof E | ((e: E) => string | number);
  visible: boolean;
  renderTag?: RenderProp<SelectMultipleTagProps<K, E>>;
  onTagDragStart?: (e: React.DragEvent<HTMLDivElement>, key: K) => void;
  onTagDragEnd?: (e: React.DragEvent<HTMLDivElement>, key: K) => void;
}

const MultipleInput = <K extends Key, E extends KeyedRenderableRecord<K, E>>({
  selectedKeys,
  selected,
  onChange,
  onFocus,
  visible,
  tagKey,
  renderTag = componentRenderProp(SelectMultipleTag),
  placeholder = "Select...",
  onTagDragStart,
  onTagDragEnd,
  value,
  className,
  ...props
}: SelectMultipleInputProps<K, E>): ReactElement => {
  const {
    select: { onSelect, clear },
  } = CoreList.useContext<K, E>();
  const ref = useRef<HTMLInputElement>(null);

  useLayoutEffect(() => {
    if (visible) ref.current?.focus();
    // // Notice how we don't call onChange with an empty value here. This is so
    // // we preserve the previous search result in the list even after we clear
    // // the box when a value is selected.
    // else setValue("");
  }, [visible, selected]);

  const handleChange = (v: string): void => {
    onChange(v);
  };

  const handleFocus: Input.TextProps["onFocus"] = (e) => {
    if (!visible) onChange("");
    onFocus?.(e);
  };

  const handleClick = (e: React.MouseEvent<HTMLDivElement>): void => {
    if (visible) return;
    onFocus?.(e);
  };

  const handleBlur = (): void => {
    if (visible) return;
    onChange("");
  };

  return (
    <Input.Text
      ref={ref}
      className={CSS(
        CSS.BE("select-multiple", "input"),
        CSS.visible(visible),
        className,
      )}
      onBlur={handleBlur}
      placeholder={placeholder}
      value={value}
      onChange={handleChange}
      onFocus={handleFocus}
      autoComplete="off"
      autoCapitalize="off"
      autoCorrect="off"
      onClick={handleClick}
      {...props}
    >
      <Align.Space
        direction="x"
        className={CSS.BE("select-multiple", "tags")}
        align="center"
        grow
      >
        {toArray(selectedKeys).map((k, i) => {
          const e = selected[i];
          return renderTag({
            key: k,
            entryKey: k,
            tagKey,
            entry: e,
            onClose: () => onSelect?.(k),
            onDragStart: (ev) => onTagDragStart?.(ev, k),
            onDragEnd: (ev) => onTagDragEnd?.(ev, k),
          });
        })}
      </Align.Space>
      <ClearButton onClick={clear} />
    </Input.Text>
  );
};

interface SelectMultipleTagProps<K extends Key, E extends KeyedRenderableRecord<K, E>> {
  key: K;
  entryKey: K;
  tagKey: keyof E | ((e: E) => string | number);
  entry?: E;
  color?: Color.Crude;
  onClose?: () => void;
  onDragStart: (e: React.DragEvent<HTMLDivElement>) => void;
  onDragEnd?: (e: React.DragEvent<HTMLDivElement>) => void;
}

const SelectMultipleTag = <K extends Key, E extends KeyedRenderableRecord<K, E>>({
  entryKey,
  tagKey,
  entry,
  ...props
}: SelectMultipleTagProps<K, E>): ReactElement => {
  let v: RenderableValue = entryKey;
  if (entry != null) v = typeof tagKey === "function" ? tagKey(entry) : entry[tagKey];
  return (
    <Tag.Tag
      size="small"
      variant="outlined"
      draggable
      {...props}
      key={entryKey.toString()}
    >
      {convertRenderV(v)}
    </Tag.Tag>
  );
};
