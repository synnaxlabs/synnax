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
import {
  type ReactNode,
  type RefObject,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { createPortal } from "react-dom";

import { Button } from "@/button";
import { CSS } from "@/css";
import { List } from "@/list";
import { CONTEXT_SELECTED, CONTEXT_TARGET } from "@/menu/types";
import { useItemState, useReselectNoop, useSelectedAmong } from "@/select/Context";
import { useRegistryContext, useSearchContext } from "@/select/registry";
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

// The element is read after every commit, so the registry follows it without a render.
const useRegister = (
  key: record.Key,
  ref: RefObject<HTMLElement | null> | null,
  hidden: boolean,
): void => {
  const registry = useRegistryContext("Select.Item");
  useLayoutEffect(() =>
    registry.setItem(key, { element: ref?.current ?? null, hidden }),
  );
  useLayoutEffect(() => () => registry.removeItem(key), [registry, key]);
};

interface ButtonItemProps<K extends record.Key>
  extends Button.ExtensionProps, Pick<ItemProps<K>, "itemKey" | "className" | "style"> {
  buttons: ButtonsContextValue;
}

const ButtonItem = <K extends record.Key>({
  itemKey,
  className,
  buttons: { preview, variant },
  ...rest
}: ButtonItemProps<K>): ReactNode => {
  const { selected, onSelect } = useItemState(itemKey);
  const ref = useRef<HTMLButtonElement>(null);
  useRegister(itemKey, ref, false);
  if (preview && !selected) return null;
  return (
    <Button.Toggle
      preview={preview}
      variant={variant}
      {...rest}
      ref={ref}
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

// A closed item draws nothing but its label, so it skips the hooks an open item needs.
const ClosedFixedItem = <K extends record.Key>({
  itemKey,
  children,
}: Pick<ItemProps<K>, "itemKey" | "children">): ReactNode => {
  const keys = useMemo(() => [itemKey], [itemKey]);
  const selected = useSelectedAmong(keys) != null;
  const registry = useRegistryContext("Select.Item");
  useRegister(itemKey, null, false);
  if (!selected) return null;
  return createPortal(children, registry.getLabel(itemKey));
};

const OpenFixedItem = <K extends record.Key, E extends Button.ElementType>(
  props: ItemProps<K, E>,
): ReactNode => {
  const { itemKey, children } = props;
  const { term } = useSearchContext("Select.Item");
  const { selected, hovered, onSelect, sole } = useItemState(itemKey);
  const reselectNoop = useReselectNoop();
  const registry = useRegistryContext("Select.Item");
  const ref = useRef<HTMLElement>(null);
  // The search matches the rendered text, which is only known after a commit. Null
  // until a term needs it, so opening the dialog measures nothing.
  const [text, setText] = useState<string | null>(null);
  useLayoutEffect(() => {
    if (term === "") return;
    const next = ref.current?.textContent ?? "";
    if (next !== text) setText(next);
  });
  const hidden = text != null && !matches(text, term);
  useRegister(itemKey, ref, hidden);
  const label = selected ? createPortal(children, registry.getLabel(itemKey)) : null;
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
        ref={ref}
      />
      {label}
    </>
  );
};

const FixedItem = <K extends record.Key, E extends Button.ElementType>(
  props: ItemProps<K, E>,
): ReactNode =>
  useClosed() ? (
    <ClosedFixedItem<K> itemKey={props.itemKey}>{props.children}</ClosedFixedItem>
  ) : (
    <OpenFixedItem<K, E> {...props} />
  );

/**
 * One option of a selection. Rendered by an {@link Items} block, it is a row of the
 * frame's data. Inside {@link Buttons} it is a toggle button. Anywhere else it is a
 * fixed option: the arrow keys reach it in page order, the search filters it by its
 * text, and a trigger shows its children when it is selected.
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
