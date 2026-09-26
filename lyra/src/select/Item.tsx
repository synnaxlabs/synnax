// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/select/Item.css";

import { type record } from "@synnaxlabs/x";
import { type ReactNode, useLayoutEffect, useState } from "react";
import { createPortal } from "react-dom";

import { Button } from "@/button";
import { CSS } from "@/css";
import { List } from "@/list";
import { CONTEXT_SELECTED, CONTEXT_TARGET } from "@/menu/types";
import { useItemState, useReselectNoop } from "@/select/Context";
import { useRegistryContext, useSearchContext, useSlot } from "@/select/registry";
import { type ButtonsContextValue, useButtonsContext, useClosed } from "@/select/scope";

/** Props for {@link Item}. */
export type ItemProps<
  K extends record.Key = record.Key,
  E extends Button.ElementType = "div",
> = List.ItemProps<K, E>;

const BlockItem = <K extends record.Key, E extends Button.ElementType>(
  props: ItemProps<K, E>,
): ReactNode => {
  const { selected, hovered, onSelect, sole } = useItemState(props.itemKey);
  const reselectNoop = useReselectNoop();
  return (
    <List.Item<K, E>
      role="option"
      selected={selected}
      hovered={hovered}
      onSelect={onSelect}
      preventClick={reselectNoop && sole ? true : undefined}
      {...props}
    />
  );
};

const useRegister = (
  key: record.Key,
  element: HTMLElement | null,
  hidden: boolean,
): void => {
  const registry = useRegistryContext("Select.Item");
  useLayoutEffect(
    () => registry.setItem(key, { element, hidden }),
    [registry, key, element, hidden],
  );
  useLayoutEffect(() => () => registry.removeItem(key), [registry, key]);
};

interface ButtonItemProps<K extends record.Key> extends Pick<
  ItemProps<K>,
  | "itemKey"
  | "className"
  | "children"
  | "tooltip"
  | "size"
  | "justify"
  | "disabled"
  | "square"
> {
  buttons: ButtonsContextValue;
}

const ButtonItem = <K extends record.Key>({
  itemKey,
  className,
  buttons: { preview, variant },
  ...rest
}: ButtonItemProps<K>): ReactNode => {
  const { selected, onSelect } = useItemState(itemKey);
  const [element, setElement] = useState<HTMLElement | null>(null);
  useRegister(itemKey, element, false);
  if (preview && !selected) return null;
  return (
    <Button.Toggle
      preview={preview}
      variant={variant}
      {...rest}
      ref={setElement}
      id={itemKey.toString()}
      onChange={onSelect}
      value={selected}
      className={CSS.cls(
        className,
        CSS.B("select-btn"),
        CSS.selected(selected),
        selected && CONTEXT_SELECTED,
        CONTEXT_TARGET,
      )}
    />
  );
};

const matches = (text: string, term: string): boolean =>
  term === "" || text.toLowerCase().includes(term.toLowerCase());

const FixedItem = <K extends record.Key, E extends Button.ElementType>(
  props: ItemProps<K, E>,
): ReactNode => {
  const { itemKey, children } = props;
  const closed = useClosed();
  const { term } = useSearchContext("Select.Item");
  const { selected, hovered, onSelect, sole } = useItemState(itemKey);
  const reselectNoop = useReselectNoop();
  const slot = useSlot(itemKey);
  const [element, setElement] = useState<HTMLElement | null>(null);
  const [text, setText] = useState("");
  // The search matches the rendered text, which is only known after mount.
  useLayoutEffect(() => {
    const next = element?.textContent ?? "";
    if (next !== text) setText(next);
  });
  const hidden = !matches(text, term);
  useRegister(itemKey, element, hidden);
  const label = selected && slot != null ? createPortal(children, slot) : null;
  if (closed) return label;
  return (
    <>
      <List.Item<K, E>
        role="option"
        selected={selected}
        hovered={hovered}
        onSelect={onSelect}
        preventClick={reselectNoop && sole ? true : undefined}
        hidden={hidden}
        {...props}
        ref={setElement}
      />
      {label}
    </>
  );
};

/**
 * One option of a selection. Rendered by an {@link Items} block, it is a row of the
 * frame's data. Inside {@link Buttons} it is a toggle button. Anywhere else it is a fixed option: the arrow keys reach it in page
 * order, the search filters it by its text, and a trigger shows its children when it is
 * selected.
 */
export const Item = <
  K extends record.Key = record.Key,
  E extends Button.ElementType = "div",
>(
  props: ItemProps<K, E>,
): ReactNode => {
  const inItems = List.useInItems();
  const buttons = useButtonsContext();
  if (inItems) return <BlockItem<K, E> {...props} />;
  if (buttons != null) return <ButtonItem<K> {...props} buttons={buttons} />;
  return <FixedItem<K, E> {...props} />;
};
