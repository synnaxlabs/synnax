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
  type MouseEvent,
  type ReactNode,
  type RefObject,
  useLayoutEffect,
  useRef,
} from "react";
import { createPortal } from "react-dom";

import { Button } from "@/button";
import { CSS } from "@/css";
import { Input } from "@/input";
import { List } from "@/list";
import { CONTEXT_SELECTED, CONTEXT_TARGET } from "@/menu/types";
import { useItemState, useReselectNoop, useSelectedAmong } from "@/select/Context";
import { type ItemOptions, useIsHidden, useRegistryContext } from "@/select/registry";
import {
  type ButtonsContextValue,
  useButtonsContext,
  useClosed,
  useInList,
} from "@/select/scope";
import { hasModifier } from "@/select/use";

/** Props for {@link Item}. */
export type ItemProps<
  K extends record.Key = record.Key,
  E extends Button.ElementType = "div",
> = List.ItemProps<K, E> & {
  /**
   * The item's name as plain text, for children that are not plain text. The search
   * matches it, and a collapsed trigger shows it in its tooltip. Defaults to the
   * children when they are a string.
   */
  textValue?: string;
  /**
   * Selects the item only through a leading checkbox or a click with Shift, Control,
   * or Command, for items that hold their own controls. A plain click calls
   * `onActivate`. Applies inside an {@link Items} block only.
   */
  checkable?: boolean;
  /** The main action of a checkable item, such as opening it, on a plain click. */
  onActivate?: () => void;
  /**
   * What a trigger shows while this fixed item is selected, for children that hold more
   * than its name. Defaults to the children.
   */
  label?: ReactNode;
};

interface BlockItemProps<
  K extends record.Key,
  E extends Button.ElementType,
> extends Pick<ItemProps<K, E>, "checkable" | "onActivate"> {
  item: List.ItemProps<K, E>;
}

const BlockItem = <K extends record.Key, E extends Button.ElementType>({
  item,
  checkable = false,
  onActivate,
}: BlockItemProps<K, E>): ReactNode => {
  const { selected, hovered, onSelect, onHover, sole } = useItemState(item.itemKey);
  const reselectNoop = useReselectNoop();
  if (!checkable)
    return (
      <List.Item<K, E>
        role="option"
        selected={selected}
        hovered={hovered}
        onHover={onHover}
        onSelect={onSelect}
        preventClick={reselectNoop && sole ? true : undefined}
        {...item}
      />
    );
  const handleSelect = (_: K, e: MouseEvent<HTMLElement>): void => {
    if (hasModifier(e)) onSelect();
    else onActivate?.();
  };
  return (
    <List.Item<K, E>
      role="option"
      selected={selected}
      hovered={hovered}
      onHover={onHover}
      {...item}
      className={CSS.cls(CSS.BEM("select", "item", "checkable"), item.className)}
      onSelect={handleSelect}
    >
      <Input.Checkbox
        value={selected}
        onChange={onSelect}
        size="medium"
        variant="text"
        reveal={!selected}
        disabled={item.disabled}
        preview={item.preview}
        aria-label="Select"
      />
      {item.children}
    </List.Item>
  );
};

// The element is read after every commit, so the registry follows it without a render.
const useRegister = (
  key: record.Key,
  ref: RefObject<HTMLElement | null> | null,
  options?: ItemOptions,
): void => {
  const registry = useRegistryContext("Select.Item");
  useLayoutEffect(() => registry.setItem(key, ref?.current ?? null, options));
  useLayoutEffect(() => () => registry.removeItem(key), [registry, key]);
};

interface ButtonItemProps<K extends record.Key>
  extends Button.ExtensionProps, Pick<ItemProps<K>, "itemKey" | "className" | "style"> {
  buttons: ButtonsContextValue;
}

const ButtonItem = <K extends record.Key>({
  itemKey,
  className,
  buttons: { preview, variant, ...itemStatus },
  ...rest
}: ButtonItemProps<K>): ReactNode => {
  const { selected, onSelect } = useItemState(itemKey);
  const ref = useRef<HTMLButtonElement>(null);
  useRegister(itemKey, ref);
  if (preview && !selected) return null;
  return (
    <Button.Toggle
      preview={preview}
      variant={variant}
      {...itemStatus}
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

// A closed item draws nothing but its label, so it skips the hooks an open item needs.
const ClosedFixedItem = <K extends record.Key>({
  itemKey,
  label,
}: Pick<ItemProps<K>, "itemKey" | "label">): ReactNode => {
  const keys = [itemKey];
  const selected = useSelectedAmong(keys) != null;
  const registry = useRegistryContext("Select.Item");
  if (!selected) return null;
  return createPortal(label, registry.getLabel(itemKey));
};

interface FixedItemProps<K extends record.Key, E extends Button.ElementType> {
  item: List.ItemProps<K, E>;
  text: string | undefined;
  label: ReactNode;
}

interface OpenFixedItemProps<
  K extends record.Key,
  E extends Button.ElementType,
> extends Pick<FixedItemProps<K, E>, "item" | "label"> {
  ref: RefObject<HTMLElement | null>;
}

const OpenFixedItem = <K extends record.Key, E extends Button.ElementType>({
  item,
  label,
  ref,
}: OpenFixedItemProps<K, E>): ReactNode => {
  const { itemKey } = item;
  const { selected, hovered, onSelect, onHover, sole } = useItemState(itemKey);
  const reselectNoop = useReselectNoop();
  const registry = useRegistryContext("Select.Item");
  const hidden = useIsHidden(itemKey);
  return (
    <>
      <List.Item<K, E>
        role="option"
        selected={selected}
        hovered={hovered}
        onHover={onHover}
        onSelect={onSelect}
        preventClick={reselectNoop && sole ? true : undefined}
        hidden={hidden}
        {...item}
        ref={ref}
      />
      {selected && createPortal(label, registry.getLabel(itemKey))}
    </>
  );
};

const FixedItem = <K extends record.Key, E extends Button.ElementType>({
  item,
  text,
  label,
}: FixedItemProps<K, E>): ReactNode => {
  const ref = useRef<HTMLElement>(null);
  const { itemKey, children } = item;
  const pinned = !useInList();
  useRegister(itemKey, ref, {
    text: text ?? (typeof children === "string" ? children : undefined),
    pinned,
  });
  return useClosed() ? (
    <ClosedFixedItem itemKey={itemKey} label={label} />
  ) : (
    <OpenFixedItem item={item} label={label} ref={ref} />
  );
};

/**
 * One option of a selection. Rendered by an {@link Items} block, it is a row of the
 * frame's data. Inside {@link Buttons} it is a toggle button. Anywhere else it is a
 * fixed option: the arrow keys reach it in page order, and a trigger shows its `label`
 * when it is selected. Inside a `Select.List` the search filters it by its text.
 * Outside one, such as in a footer, it is pinned and the search never hides it.
 */
export const Item = <
  K extends record.Key = record.Key,
  E extends Button.ElementType = "div",
>(
  props: ItemProps<K, E>,
): ReactNode => {
  const inItems = List.useInItems();
  const buttons = useButtonsContext();
  const { textValue, checkable, onActivate, label, ...rest } = props;
  if (inItems)
    return <BlockItem item={rest} checkable={checkable} onActivate={onActivate} />;
  if (buttons != null) return <ButtonItem {...rest} buttons={buttons} />;
  return <FixedItem item={rest} text={textValue} label={label ?? rest.children} />;
};
