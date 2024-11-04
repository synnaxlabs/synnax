// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/select/Multiple.css";

import { Icon } from "@synnaxlabs/media";
import {
  type AsyncTermSearcher,
  compare,
  convertRenderV,
  type Key,
  type Keyed,
  type RenderableValue,
  toArray,
} from "@synnaxlabs/x";
import {
  type ReactElement,
  ReactNode,
  useCallback,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
} from "react";

import { Align } from "@/align";
import { Button } from "@/button";
import { type Color } from "@/color";
import { CSS } from "@/css";
import { Dropdown } from "@/dropdown";
import { useAsyncEffect } from "@/hooks";
import { Input } from "@/input";
import { List as CoreList } from "@/list";
import {
  selectValueIsZero,
  type UseSelectMultipleProps,
  UseSelectOnChangeExtra,
} from "@/list/useSelect";
import { ClearButton } from "@/select/ClearButton";
import { Core } from "@/select/List";
import { DEFAULT_PLACEHOLDER } from "@/select/Single";
import { Tag } from "@/tag";
import { componentRenderProp, type RenderProp } from "@/util/renderProp";

export interface MultipleProps<K extends Key = Key, E extends Keyed<K> = Keyed<K>>
  extends Omit<Dropdown.DialogProps, "visible" | "onChange" | "children" | "close">,
    Omit<UseSelectMultipleProps<K, E>, "data">,
    Omit<CoreList.ListProps<K, E>, "children">,
    Pick<Input.TextProps, "placeholder">,
    Partial<Pick<CoreList.VirtualCoreProps<K, E>, "itemHeight">> {
  columns?: Array<CoreList.ColumnSpec<K, E>>;
  searcher?: AsyncTermSearcher<string, K, E>;
  entryRenderKey?: keyof E | ((e: E) => string | number);
  renderTag?: RenderProp<MultipleTagProps<K, E>>;
  onTagDragStart?: (e: React.DragEvent<HTMLDivElement>, key: K) => void;
  onTagDragEnd?: (e: React.DragEvent<HTMLDivElement>, key: K) => void;
  children?: CoreList.VirtualCoreProps<K, E>["children"];
  dropdownVariant?: Dropdown.Variant;
  addPlaceholder?: ReactNode;
  actions?: Input.ExtensionProps["children"];
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
 * @param props.entryRenderKey - The option field rendered for each tag when selected in the
 * input group. Defaults to "key".
 * @param props.location - Whether to render the dropdown above or below the select
 * component. Defaults to "below".
 * @param props.onChange - The callback to be invoked when the selected value changes.
 * @param props.value - The currently selected value.
 */
export const Multiple = <K extends Key = Key, E extends Keyed<K> = Keyed<K>>({
  onChange,
  value,
  className,
  data,
  columns = [],
  entryRenderKey = "key",
  emptyContent,
  searcher,
  renderTag = componentRenderProp(MultipleTag<K, E>),
  placeholder,
  onTagDragStart,
  allowMultiple,
  allowNone,
  replaceOnSingle,
  onTagDragEnd,
  style,
  addPlaceholder = "Add",
  children,
  dropdownVariant = "connected",
  actions,
  ...props
}: MultipleProps<K, E>): ReactElement => {
  const { visible, open, close } = Dropdown.use();
  const [selected, setSelected] = useState<readonly E[]>([]);
  const [loading, setLoading] = useState(false);
  const searchMode = searcher != null;

  // This hook makes sure we have the selected entries fetched to render their tags
  // properly.
  useAsyncEffect(async () => {
    setLoading(true);
    if (selectValueIsZero(value)) setSelected([]);
    const inSelected = selected.map((v) => v.key);
    const nextValue = toArray(value);
    if (compare.unorderedPrimitiveArrays(inSelected, nextValue) === compare.EQUAL)
      return;
    let nextSelected: E[] = [];
    if (searchMode) {
      // Wrap this in a try-except clause just in case the searcher throws an error.
      try {
        nextSelected = await searcher.retrieve(nextValue);
      } finally {
        setLoading(false);
      }
    } else if (data != null)
      nextSelected = data.filter((v) => nextValue.includes(v.key));
    setSelected(nextSelected);
  }, [searcher, searchMode, value, data]);

  const handleChange = useCallback(
    (v: K | K[] | null, extra: UseSelectOnChangeExtra<K, E>) => {
      if (v == null) return;
      setSelected(extra.entries);
      onChange(toArray(v), extra);
    },
    [onChange],
  );

  const InputWrapper = useMemo(
    () => (searchMode ? CoreList.Search : CoreList.Filter),
    [searchMode],
  );

  let searchInput: ReactElement | undefined;
  let trigger: ReactElement;
  if (dropdownVariant === "connected") {
    trigger = (
      <InputWrapper<K, E> searcher={searcher}>
        {({ onChange, value: inputValue }) => (
          <MultipleInput<K, E>
            value={inputValue}
            selectedKeys={value}
            className={className}
            onChange={onChange}
            loading={loading}
            selected={selected}
            onFocus={open}
            entryRenderKey={entryRenderKey}
            visible={visible}
            renderTag={renderTag}
            placeholder={placeholder}
            onTagDragStart={onTagDragStart}
            onTagDragEnd={onTagDragEnd}
            style={style}
          >
            {actions}
          </MultipleInput>
        )}
      </InputWrapper>
    );
  } else {
    const arrValue = toArray(value);
    trigger = (
      <Align.Space direction="x" align="center" grow style={style} size="small">
        {arrValue.map((k) => {
          const e = selected.find((v) => v.key === k);
          return renderTag({
            key: k,
            entryKey: k,
            entryRenderKey,
            loading,
            entry: e,
            onClose: () => {
              const next = arrValue.filter((v) => v !== k);
              onChange(next, {
                clicked: null,
                clickedIndex: -1,
                entries: [],
              });
            },
            onDragStart: (ev) => onTagDragStart?.(ev, k),
            onDragEnd: (ev) => onTagDragEnd?.(ev, k),
          });
        })}
        {arrValue.length > 0 ? (
          <Button.Icon onClick={open}>
            <Icon.Add style={{ color: "var(--pluto-gray-l7)" }} />
          </Button.Icon>
        ) : (
          <Button.Button
            onClick={open}
            shade={7}
            startIcon={<Icon.Add />}
            variant="text"
          >
            {addPlaceholder}
          </Button.Button>
        )}
      </Align.Space>
    );
    searchInput = (
      <InputWrapper<K, E> searcher={searcher}>
        {(p) => (
          <Input.Text autoFocus placeholder="Search" {...p}>
            {actions}
          </Input.Text>
        )}
      </InputWrapper>
    );
  }

  return (
    <Core<K, E>
      close={close}
      open={open}
      data={data}
      emptyContent={emptyContent}
      visible={visible}
      value={value}
      onChange={handleChange}
      allowNone={allowNone}
      replaceOnSingle={replaceOnSingle}
      columns={columns}
      allowMultiple
      listItem={children}
      trigger={trigger}
      extraDialogContent={searchInput}
      variant={dropdownVariant}
      {...props}
    />
  );
};

interface SelectMultipleInputProps<K extends Key, E extends Keyed<K>>
  extends Omit<Input.TextProps, "onFocus"> {
  loading: boolean;
  selectedKeys: K | K[];
  selected: readonly E[];
  entryRenderKey: keyof E | ((e: E) => string | number);
  visible: boolean;
  renderTag: RenderProp<MultipleTagProps<K, E>>;
  onTagDragStart?: (e: React.DragEvent<HTMLDivElement>, key: K) => void;
  onTagDragEnd?: (e: React.DragEvent<HTMLDivElement>, key: K) => void;
  onFocus?: () => void;
}

const MultipleInput = <K extends Key, E extends Keyed<K>>({
  selectedKeys,
  loading,
  selected,
  onChange,
  onFocus,
  visible,
  entryRenderKey,
  renderTag,
  placeholder = DEFAULT_PLACEHOLDER,
  onTagDragStart,
  onTagDragEnd,
  value,
  className,
  children,
  ...props
}: SelectMultipleInputProps<K, E>): ReactElement => {
  const { onSelect, clear } = CoreList.useSelectionUtils();
  const ref = useRef<HTMLInputElement>(null);

  useLayoutEffect(() => {
    if (visible) ref.current?.focus();
  }, [visible, selected]);

  const handleFocus: Input.TextProps["onFocus"] = () => {
    if (!visible) onChange("");
    onFocus?.();
  };

  const handleClick = (): void => {
    if (visible) return;
    onFocus?.();
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
      onChange={onChange}
      onFocus={handleFocus}
      autoComplete="off"
      autoCapitalize="off"
      autoCorrect="off"
      onClick={handleClick}
      variant="button"
      {...props}
    >
      <Align.Space
        direction="x"
        className={CSS.BE("select-multiple", "tags")}
        align="center"
        grow
        size="small"
      >
        {toArray(selectedKeys).map((k) => {
          const e = selected.find((v) => v.key === k);
          return renderTag({
            key: k,
            entryKey: k,
            entryRenderKey,
            loading,
            entry: e,
            onClose: () => onSelect?.(k),
            onDragStart: (ev) => onTagDragStart?.(ev, k),
            onDragEnd: (ev) => onTagDragEnd?.(ev, k),
          });
        })}
      </Align.Space>
      {children}
      <ClearButton onClick={clear} />
    </Input.Text>
  );
};

export interface MultipleTagProps<K extends Key, E extends Keyed<K>> {
  key: K;
  entryKey: K;
  entryRenderKey: keyof E | ((e: E) => string | number);
  entry?: E;
  color?: Color.Crude;
  loading: boolean;
  onClose?: () => void;
  onDragStart: (e: React.DragEvent<HTMLDivElement>) => void;
  onDragEnd?: (e: React.DragEvent<HTMLDivElement>) => void;
}

export const MultipleTag = <K extends Key, E extends Keyed<K>>({
  entryKey,
  entryRenderKey,
  entry,
  loading,
  ...props
}: MultipleTagProps<K, E>): ReactElement => {
  let v: RenderableValue = entryKey;
  if (entry != null)
    v =
      typeof entryRenderKey === "function"
        ? entryRenderKey(entry)
        : (entry[entryRenderKey] as RenderableValue);
  return (
    <Tag.Tag
      size="small"
      variant="outlined"
      className={CSS(
        entry == null && !loading && CSS.BEM("select-multiple", "tag", "invalid"),
      )}
      draggable
      {...props}
      key={entryKey.toString()}
    >
      {convertRenderV(v)}
    </Tag.Tag>
  );
};
