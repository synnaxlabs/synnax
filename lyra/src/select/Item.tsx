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
  type ReactElement,
  type ReactNode,
  type RefObject,
  useLayoutEffect,
  useRef,
  useState,
} from "react";
import { createPortal } from "react-dom";

import { Button } from "@/button";
import { CSS } from "@/css";
import { Dialog } from "@/dialog";
import { Icon } from "@/icon";
import { Input } from "@/input";
import { List } from "@/list";
import { nodeText, useMatches, usePathChildren } from "@/menu/filter";
import { CLOSES_ATTRIBUTE, CONTEXT_SELECTED, CONTEXT_TARGET } from "@/menu/types";
import { useItemState, useReselectNoop, useSelectedAmong } from "@/select/Context";
import {
  type ItemOptions,
  useHasName,
  useIsHidden,
  useRegistryContext,
  useTerm,
} from "@/select/registry";
import {
  type ButtonsContextValue,
  useButtonsContext,
  useClosed,
  useInline,
  useInList,
  useShownContext,
} from "@/select/scope";
import { hasModifier } from "@/select/use";
import { Text } from "@/text";

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

interface InlineItemProps<K extends record.Key, E extends Button.ElementType> {
  item: List.ItemProps<K, E>;
  text?: string;
}

// In its menu, an option is a menu item, and its click closes the menu.
const InlineItem = <K extends record.Key, E extends Button.ElementType>({
  item,
  text,
}: InlineItemProps<K, E>): ReactNode => {
  const { selected, hovered, onSelect, onHover, sole } = useItemState(item.itemKey);
  const reselectNoop = useReselectNoop();
  const children = usePathChildren(item.children);
  // The search of the list answers late, so the menu filter checks each option itself.
  const matches = useMatches(text ?? nodeText(item.children));
  const setShown = useShownContext();
  useLayoutEffect(() => {
    if (!matches || setShown == null) return;
    setShown((count) => count + 1);
    return () => setShown((count) => count - 1);
  }, [matches, setShown]);
  if (!matches) return null;
  return (
    <List.Item<K, E>
      role="menuitem"
      propagateClick
      selected={selected}
      hovered={hovered}
      onHover={onHover}
      onSelect={onSelect}
      preventClick={reselectNoop && sole ? true : undefined}
      {...item}
    >
      {children}
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
  const inline = useInline();
  const { textValue, checkable, onActivate, label, ...rest } = props;
  if (inItems && inline) return <InlineItem item={rest} text={textValue} />;
  if (inItems)
    return <BlockItem item={rest} checkable={checkable} onActivate={onActivate} />;
  if (buttons != null) return <ButtonItem {...rest} buttons={buttons} />;
  return <FixedItem item={rest} text={textValue} label={label ?? rest.children} />;
};

/** Props for {@link ItemDescription}. */
export type ItemDescriptionProps = Text.TextProps;

/**
 * A short line under the name of an {@link Item} that says what the option does. The
 * item's first child is its icon, which then sits beside the name. Give the item a
 * `label` and a `textValue`, so the trigger and the search use only the name.
 *
 * @example
 * <Select.Item itemKey="high" label="High" textValue="High">
 *   <Icon.Meter.High />
 *   High
 *   <Select.ItemDescription>Most detail.</Select.ItemDescription>
 * </Select.Item>
 */
export const ItemDescription = ({
  className,
  ...rest
}: ItemDescriptionProps): ReactElement => (
  <Text.Text
    className={CSS.cls(CSS.BE("select", "item-description"), className)}
    color={9}
    level="small"
    overflow="wrap"
    {...rest}
  />
);

const CREATE_KEY = "lyra-select-create";

/** Props for {@link CreateItem}. */
export interface CreateItemProps {
  /** The name of what the item creates, such as "label". */
  resourceName: string;
  /** Whether every create happens in a modal, a search term included. */
  modal?: boolean;
  /**
   * Called on a click or Enter with the trimmed search term. A create in a modal, as
   * from an empty term, closes the dialog and any menu that holds it.
   */
  onCreate: (name: string) => void;
}

interface CreateOptionProps extends Pick<CreateItemProps, "onCreate"> {
  name: string;
  text: string;
  leaves: boolean;
}

const CreateOption = ({
  name,
  text,
  leaves,
  onCreate,
}: CreateOptionProps): ReactNode => {
  const ref = useRef<HTMLDivElement>(null);
  useRegister(CREATE_KEY, ref, { text, pinned: true });
  const { hovered, onHover } = useItemState<record.Key>(CREATE_KEY);
  const { close } = Dialog.useContext();
  const handleClick = (): void => {
    if (leaves) close();
    onCreate(name);
  };
  return (
    <List.Item<record.Key>
      role="option"
      itemKey={CREATE_KEY}
      hovered={hovered}
      onHover={onHover}
      onClick={handleClick}
      ref={ref}
      propagateClick={leaves}
      {...{ [CLOSES_ATTRIBUTE]: leaves ? true : undefined }}
    >
      <Icon.Add />
      {text}
    </List.Item>
  );
};

const OpenCreateItem = <
  K extends record.Key,
  E extends record.Keyed<K> & { name: string },
>({
  resourceName,
  modal = false,
  onCreate,
}: CreateItemProps): ReactNode => {
  const name = useTerm().trim();
  const lower = name.toLowerCase();
  // A name created in place stays hidden until the search changes, so a second click
  // while the item saves makes no copy.
  const [picked, setPicked] = useState<string | null>(null);
  if (picked != null && picked !== lower) setPicked(null);
  const { data, getItem } = List.useData<K, E>();
  const fixed = useHasName(name);
  const listed = data.some((k) => getItem?.(k)?.name.toLowerCase() === lower);
  if (name !== "" && (fixed || listed || picked === lower)) return null;
  const text = name === "" ? `New ${resourceName}` : `Create ${resourceName} "${name}"`;
  const leaves = modal || name === "";
  const handleCreate = (next: string): void => {
    if (!leaves) setPicked(lower);
    onCreate(next);
  };
  return (
    <CreateOption name={name} text={text} leaves={leaves} onCreate={handleCreate} />
  );
};

/**
 * A pinned option that creates an item from the search. It reads "New label" with no
 * search and Create label "text" with one, and it hides while an option, fixed or not,
 * has that name in any case. It never changes the selection, so `onCreate` picks what
 * it creates.
 */
export const CreateItem = <
  K extends record.Key,
  E extends record.Keyed<K> & { name: string },
>(
  props: CreateItemProps,
): ReactNode => (useClosed() ? null : <OpenCreateItem<K, E> {...props} />);
